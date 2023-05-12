package main

import (
	"context"

	"github.com/miekg/dns"
)

func BlockResolver(
	ctx context.Context,
	logger Logger,
	records ...*Record,
) (*Block, error) {
	err := checkNil(ctx, logger)
	if err != nil {
		return nil, err
	}

	m, err := NewMatcher(ctx, logger, records...)
	if err != nil {
		return nil, err
	}

	return &Block{
		Matcher: m,
		ctx:     ctx,
		logger:  logger,
	}, nil
}

type Block struct {
	*Matcher
	ctx    context.Context
	logger Logger
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
		b.logger.Errorw(
			"failed to block",
			"category", BLOCK,
			"name", req.r.Question[0].Name,
			"type", dns.Type(req.r.Question[0].Qtype),
			"record", record,
			"client", req.client,
			"server", req.server,
		)
	}

	b.logger.Infow(
		"matched",
		"category", BLOCK,
		"name", req.r.Question[0].Name,
		"type", dns.Type(req.r.Question[0].Qtype),
		"record", record,
		"client", req.client,
		"server", req.server,
	)

	return nil, false
}
