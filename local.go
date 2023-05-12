package main

import (
	"context"

	"github.com/miekg/dns"
)

// TODO: Should I have a local regex resolver?

func LocalResolver(
	ctx context.Context,
	logger Logger,
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
	logger Logger
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
		l.logger.Errorw(
			"failed to answer request",
			"server", "local-resolver",
			"category", LOCAL,
			"error", err,
			"record", req.String(),
		)
	}

	l.logger.Debugw(
		"answered request",
		"server", "local-resolver",
		"category", LOCAL,
		"name", req.r.Question[0].Name,
		"type", dns.Type(req.r.Question[0].Qtype),
		"record", record,
	)

	return nil, false
}
