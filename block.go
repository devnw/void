package main

import (
	"context"

	"go.devnw.com/event"
)

func BlockResolver(
	ctx context.Context,
	pub *event.Publisher,
	records ...*Record,
) (*Block, error) {
	err := checkNil(ctx, pub)
	if err != nil {
		return nil, err
	}

	m, err := NewMatcher(ctx, pub, records...)
	if err != nil {
		return nil, err
	}

	return &Block{
		Matcher: m,
		ctx:     ctx,
		pub:     pub,
	}, nil
}

type Block struct {
	Matcher
	ctx      context.Context
	pub      *event.Publisher
	upstream chan<- *Request
}

func (b *Block) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	// Check for match
	record := b.Match(ctx, req.Record())
	if record == nil || record.IP == nil {

		// No match continue to next resolver
		return req, true
	}

	// Matched a blocked record
	err := req.Block()
	if err != nil {
		// TODO: Log error
	}

	return nil, false
}
