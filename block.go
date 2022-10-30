package main

import (
	"context"
	"fmt"

	"github.com/miekg/dns"
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
	*Matcher
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
	if record == nil {
		// No match continue to next resolver
		return req, true
	}

	// Matched a blocked record
	err := req.Block()
	if err != nil {
		b.pub.ErrorFunc(ctx, func() error {
			return fmt.Errorf("failed to block request: %v", err)
		})
	}

	b.pub.EventFunc(ctx, func() event.Event {
		return &Event{
			Msg:      "query found in block list",
			Name:     req.r.Question[0].Name,
			Type:     dns.Type(req.r.Question[0].Qtype),
			Category: BLOCK,
			Record:   record,
		}
	})

	return nil, false
}
