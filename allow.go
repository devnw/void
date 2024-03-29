package main

import (
	"context"

	"github.com/miekg/dns"
)

func AllowResolver(
	ctx context.Context,
	logger Logger,
	upstream chan<- *Request,
	records ...*Record,
) (*Allow, error) {
	err := checkNil(ctx, logger)
	if err != nil {
		return nil, err
	}

	m, err := NewMatcher(ctx, logger, records...)
	if err != nil {
		return nil, err
	}

	return &Allow{
		Matcher:  m,
		ctx:      ctx,
		logger:   logger,
		upstream: upstream,
	}, nil
}

type Allow struct {
	*Matcher
	ctx      context.Context
	logger   Logger
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
		a.logger.Debugw(
			"matched",
			"category", ALLOW,
			"name", req.r.Question[0].Name,
			"type", dns.Type(req.r.Question[0].Qtype),
			"record", record,
		)
	}

	return nil, false
}
