package main

import (
	"context"
	"strings"

	"github.com/miekg/dns"
	"golang.org/x/exp/slog"
)

func BlockResolver(
	ctx context.Context,
	logger *slog.Logger,
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
	logger *slog.Logger
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
		b.logger.ErrorCtx(ctx, "failed to block",
			slog.String("category", string(BLOCK)),
			slog.String("error", err.Error()),
			slog.Group("dns",
				slog.String("question", req.r.Question[0].Name),
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

	b.logger.InfoCtx(ctx, "matched",
		slog.String("category", string(BLOCK)),
		slog.Group("dns",
			slog.String("question", req.r.Question[0].Name),
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

	return nil, false
}
