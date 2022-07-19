package main

import (
	"context"
	"fmt"
	"regexp"
	"time"

	stream "go.atomizer.io/stream"
	"go.devnw.com/event"
)

func Match(
	ctx context.Context,
	pub *event.Publisher,
	patterns ...string,
) (*Regex, error) {
	requests := make(chan matcher)

	patternChans := make([]chan<- matcher, 0, len(patterns))

	for _, pattern := range patterns {
		exp, err := regexp.Compile(pattern)
		if err != nil {
			pub.ErrorFunc(ctx, func() error {
				return fmt.Errorf("%s: %s", pattern, err)
			})
		}

		in := make(chan matcher)

		// Append the pattern to the list of patterns
		// for the fan-out
		patternChans = append(patternChans, in)

		// Setup the pattern so it can scale to handle load
		s := stream.Scaler[matcher, struct{}]{
			Wait: time.Nanosecond,
			Life: time.Millisecond,
			Fn:   (&expr{exp}).match,
		}

		_, err = s.Exec(ctx, in)
		if err != nil {
			pub.ErrorFunc(ctx, func() error {
				return fmt.Errorf("%s: %s", pattern, err)
			})
		}
	}

	if len(patternChans) == 0 {
		return nil, fmt.Errorf("no patterns provided")
	}

	go stream.FanOut(ctx, requests, patternChans...)

	return &Regex{len(patternChans), requests}, nil
}

type Regex struct {
	patterns int
	requests chan<- matcher
}

func (r *Regex) Match(
	ctx context.Context,
	data string,
	timeout time.Duration,
) <-chan string {
	out := make(chan string, 1)

	go func() {
		defer close(out)

		ctx, cancel := context.WithTimeout(ctx, timeout)
		detection := make(chan string)

		// Push the match request
		select {
		case <-ctx.Done():
			return
		case r.requests <- &matchReq{
			ctx:    ctx,
			cancel: cancel,
			data:   data,
			match:  detection,
		}:
		}

		for i := 0; i < r.patterns; i++ {
			// Wait for the match
			select {
			case <-ctx.Done():
				return
			case pattern, ok := <-detection:
				if !ok {
					return
				}

				if pattern == "" {
					continue
				}

				out <- pattern
			}
		}
	}()

	return out
}

type matchReq struct {
	ctx    context.Context
	cancel context.CancelFunc
	data   string
	match  chan string
}

func (m *matchReq) Data() (context.Context, string) {
	return m.ctx, m.data
}

func (m *matchReq) Matched(ctx context.Context, pattern string) {
	select {
	case <-ctx.Done():
		return
	case <-m.ctx.Done():
		return
	case m.match <- pattern:
	}
}

type matcher interface {
	Data() (context.Context, string)
	Matched(ctx context.Context, pattern string)
}

type expr struct {
	pattern *regexp.Regexp
}

func (e *expr) match(
	ctx context.Context,
	req matcher,
) (struct{}, bool) {
	ctx, data := req.Data()
	if data == "" {
		return struct{}{}, false
	}

	select {
	case <-ctx.Done():
		return struct{}{}, false
	default:
		match := ""
		if e.pattern.MatchString(data) {
			match = e.pattern.String()
		}

		req.Matched(ctx, match)
	}

	return struct{}{}, false
}
