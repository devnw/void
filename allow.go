package main

import (
	"context"
	"strings"

	"github.com/miekg/dns"
	"golang.org/x/exp/slog"
)

func AllowResolver(
	ctx context.Context,
	logger *slog.Logger,
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
	logger   *slog.Logger
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
		a.logger.DebugCtx(ctx, "matched",
			slog.String("category", string(ALLOW)),
			slog.Group("dns",
				slog.String("name", req.r.Question[0].Name),
				slog.String("type", dns.Type(req.r.Question[0].Qtype).String()),
				slog.String("client", req.client),
				slog.String("server", req.server),
				slog.Int("reqId", int(req.r.Id)),
			),
			slog.Group("pattern",
				slog.String("value", record.Pattern),
				slog.String("type", string(record.Type)),
				slog.String("source", record.Source),
				slog.String("tags", strings.Join(record.Tags, ",")),
				slog.String("comment", record.Comment),
			),
		)
	}

	return nil, false
}
