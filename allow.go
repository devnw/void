package main

import (
	"context"

	"github.com/miekg/dns"
	"go.devnw.com/event"
)

func AllowResolver(
	ctx context.Context,
	pub *event.Publisher,
	upstream chan<- *Request,
	records ...*Record,
) (*Allow, error) {
	err := checkNil(ctx, pub)
	if err != nil {
		return nil, err
	}

	m, err := NewMatcher(ctx, pub, records...)
	if err != nil {
		return nil, err
	}

	return &Allow{
		Matcher:  m,
		ctx:      ctx,
		pub:      pub,
		upstream: upstream,
	}, nil
}

type Allow struct {
	Matcher
	ctx      context.Context
	pub      *event.Publisher
	upstream chan<- *Request
}

func (a *Allow) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	// Check for match
	record := a.Match(ctx, req.Record())
	if record == nil {
		// No match continue to next resolver
		return req, true
	}

	// Matched, send to upstream DNS servers
	select {
	case <-a.ctx.Done():
	case <-ctx.Done():
	case a.upstream <- req:
		a.pub.EventFunc(ctx, func() event.Event {
			return &Event{
				Msg:      "query found in allow list",
				Name:     req.r.Question[0].Name,
				Type:     dns.Type(req.r.Question[0].Qtype),
				Category: ALLOW,
				Record:   record,
			}
		})
	}

	return nil, false
}
