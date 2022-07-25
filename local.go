package main

import (
	"context"
	"fmt"

	"github.com/miekg/dns"
	"go.devnw.com/event"
)

// TODO: Should I have a local regex resolver?

func LocalResolver(
	ctx context.Context,
	pub *event.Publisher,
	records ...*Record,
) (*Local, error) {
	err := checkNil(ctx, pub)
	if err != nil {
		return nil, err
	}

	m, err := NewMatcher(ctx, pub, records...)
	if err != nil {
		return nil, err
	}

	return &Local{
		Matcher: m,
		ctx:     ctx,
		pub:     pub,
	}, nil
}

// Local is the local DNS resolver implementation which handles the locally
// configured DNS records. This does NOT include any blocked or allowed
// records nor does it handle caching upstream DNS records. This is strictly
// for local DNS records.
type Local struct {
	Matcher
	ctx context.Context
	pub *event.Publisher
}

// Intercept implements the stream.InterceptFunc which
// can then be used throughout the stream library and
// responds to DNS requests for local DNS records
func (l *Local) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	fmt.Println("LocalResolver.Intercept")
	// Only support A, AAAA, and CNAME records for local
	// records for now
	if req.r.Question[0].Qtype != dns.TypeA &&
		req.r.Question[0].Qtype != dns.TypeAAAA && // This is ipv6
		req.r.Question[0].Qtype != dns.TypeCNAME {
		return req, true
	}

	record := l.Match(ctx, req.Record())
	if record == nil || record.IP == nil {
		return req, true
	}
	//rr, err := dns.NewRR(fmt.Sprintf("%s %v IN A %s", req.r.Question[0].Name, DEFAULTTTL, record.IP.String()))
	//if err != nil {
	//	return req, true
	//}

	req.r.Answer = append(req.r.Answer, &dns.A{
		Hdr: dns.RR_Header{
			Name:     req.r.Question[0].Name,
			Rrtype:   uint16(record.Type),
			Class:    dns.ClassINET,
			Ttl:      DEFAULTTTL,
			Rdlength: uint16(len(record.IP)),
		},
		A: record.IP,
	})

	err := req.Answer(req.r)
	if err != nil {
		l.pub.ErrorFunc(ctx, func() error {
			return Error{
				Category: LOCAL,
				Server:   "local-resolver",
				Msg:      "failed to answer request",
				Inner:    err,
				Record:   req.String(),
			}
		})
	}

	// TODO: Add Event for local record
	return nil, false
}
