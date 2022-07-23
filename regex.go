package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	stream "go.atomizer.io/stream"
	"go.devnw.com/event"
)

func Match(
	ctx context.Context,
	pub *event.Publisher,
	records ...*Record,
) (*Regex, error) {
	requests := make(chan matcher)

	patternChans := make([]chan<- matcher, 0, len(records))

	for _, record := range records {
		var exp *regexp.Regexp
		var err error

		if record.Eval != REGEX && record.Eval != WILDCARD {
			continue
		}

		if record.Eval == WILDCARD {
			exp, err = Wildcard(record.Pattern)
			if err != nil {
				pub.ErrorFunc(ctx, func() error {
					return fmt.Errorf("%s: %s", record.Pattern, err)
				})

				continue
			}
		} else {
			exp, err = regexp.Compile(record.Pattern)
			if err != nil {
				pub.ErrorFunc(ctx, func() error {
					return fmt.Errorf("%s: %s", record.Pattern, err)
				})

				continue
			}
		}

		in := make(chan matcher)

		// Append the pattern to the list of patterns
		// for the fan-out
		patternChans = append(patternChans, in)

		// Setup the pattern so it can scale to handle load
		s := stream.Scaler[matcher, struct{}]{
			Wait: time.Nanosecond,
			Life: time.Millisecond,
			Fn:   (&expr{record, exp}).match,
		}

		_, err = s.Exec(ctx, in)
		if err != nil {
			pub.ErrorFunc(ctx, func() error {
				return fmt.Errorf("%s: %s", record.Pattern, err)
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
) <-chan *Record {
	out := make(chan *Record, 1)

	go func() {
		defer close(out)

		// Collapse request immediately when there are no patterns
		if r.patterns == 0 {
			return
		}

		ctx, cancel := context.WithTimeout(ctx, timeout)
		detection := make(chan *Record)

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
			case record, ok := <-detection:
				if !ok {
					return
				}

				if record == nil {
					continue
				}

				out <- record
			}
		}
	}()

	return out
}

type matchReq struct {
	ctx    context.Context
	cancel context.CancelFunc
	data   string
	match  chan *Record
}

func (m *matchReq) Data() (context.Context, string) {
	return m.ctx, m.data
}

func (m *matchReq) Matched(ctx context.Context, record *Record) {
	select {
	case <-ctx.Done():
		return
	case <-m.ctx.Done():
		return
	case m.match <- record:
	}
}

type matcher interface {
	Data() (context.Context, string)
	Matched(ctx context.Context, record *Record)
}

type expr struct {
	record  *Record
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
		var matched *Record
		if e.pattern.MatchString(data) {
			matched = e.record
		}

		req.Matched(ctx, matched)
	}

	return struct{}{}, false
}

const wrapper = `(\.|^)%s\.%s$`

var ErrWildcard = fmt.Errorf("invalid wildcard")
var ErrTLD = fmt.Errorf("invalid TLD")
var ErrDomain = fmt.Errorf("invalid domain")

func Wildcard(entry string) (*regexp.Regexp, error) {
	wild := strings.LastIndex(entry, "*")
	if wild != 0 {
		return nil, ErrWildcard
	}

	tldi := strings.LastIndex(entry, ".")
	if tldi == -1 || tldi == len(entry)-1 {
		return nil, ErrTLD
	}

	body := entry[wild+1 : tldi]
	tld := entry[tldi+1:]
	if len(body) == 0 {
		return nil, ErrDomain
	}

	return regexp.Compile(fmt.Sprintf(wrapper, body, tld))
}
