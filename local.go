package main

import (
	"context"
	"strings"

	"github.com/miekg/dns"
	"golang.org/x/exp/slog"
)

// TODO: Should I have a local regex resolver?

func LocalResolver(
	ctx context.Context,
	logger *slog.Logger,
	records ...*Record,
) (*Local, error) {
	m, err := NewMatcher(ctx, logger, records...)
	if err != nil {
		return nil, err
	}

	return &Local{
		Matcher: m,
		ctx:     ctx,
		logger:  logger,
	}, nil
}

// Local is the local DNS resolver implementation which handles the locally
// configured DNS records. This does NOT include any blocked or allowed
// records nor does it handle caching upstream DNS records. This is strictly
// for local DNS records.
type Local struct {
	*Matcher
	ctx    context.Context
	logger *slog.Logger
}

// Intercept implements the stream.InterceptFunc which
// can then be used throughout the stream library and
// responds to DNS requests for local DNS records.
func (l *Local) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	if req.r.Question[0].Qtype != dns.TypeA &&
		req.r.Question[0].Qtype != dns.TypeAAAA {
		return req, true
	}

	record := l.Match(ctx, req.Record())
	if record == nil || record.IP == nil {
		return req, true
	}

	req.r.Answer = append(req.r.Answer, &dns.A{
		Hdr: dns.RR_Header{
			Name:     req.r.Question[0].Name,
			Rrtype:   dns.TypeA,
			Class:    dns.ClassINET,
			Ttl:      DEFAULTTTL,
			Rdlength: uint16(len(record.IP)),
		},
		A: record.IP,
	})

	err := req.Answer(req.r)
	if err != nil {
		l.logger.ErrorCtx(ctx, "failed to answer request",
			slog.String("category", string(LOCAL)),
			slog.String("error", err.Error()),
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

	l.logger.InfoCtx(ctx, "matched",
		slog.String("category", string(LOCAL)),
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

	return nil, false
}
