package main

import (
	"context"
	"fmt"

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
	fmt.Println("AllowResolver.Intercept")
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
	}

	return nil, false
}
