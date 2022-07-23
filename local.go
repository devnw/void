package main

import (
	"context"
	"sync"
	"time"

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

	// TODO: Add future support for specific record types
	regex := []*Record{}
	local := map[string]*Record{}
	for _, r := range records {
		switch r.Eval {
		case REGEX, WILDCARD:
			regex = append(regex, r)
		case DIRECT:
			local[r.Pattern] = r
		}
	}

	var regexMatcher *Regex
	if len(regex) > 0 {
		regexMatcher, err = Match(ctx, pub, regex...)
		if err != nil {
			return nil, err
		}
	}

	return &Local{
		ctx:     ctx,
		pub:     pub,
		records: local,
		regex:   regexMatcher,
	}, nil
}

// Local is the local DNS resolver implementation which handles the locally
// configured DNS records. This does NOT include any blocked or allowed
// records nor does it handle caching upstream DNS records. This is strictly
// for local DNS records.
type Local struct {
	ctx       context.Context
	pub       *event.Publisher
	records   map[string]*Record
	recordsMu sync.RWMutex
	regex     *Regex
}

// Add adds a local record to the local resolver
// TODO: Setup parallel routine to store records in local LocalResolver
// configuration file
func (l *Local) Add(r *Record) {
	l.recordsMu.Lock()
	l.records[r.Pattern] = r
	l.recordsMu.Unlock()
}

// Intercept implements the stream.InterceptFunc which
// can then be used throughout the stream library and
// responds to DNS requests for local DNS records
func (l *Local) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	// No local records to check
	if len(l.records) == 0 {
		return req, true
	}

	// Only support A, AAAA, and CNAME records for local
	// records for now
	if req.r.Question[0].Qtype != dns.TypeA &&
		req.r.Question[0].Qtype != dns.TypeAAAA && // This is ipv6
		req.r.Question[0].Qtype != dns.TypeCNAME {
		return req, true
	}

	// Found in allow list, continue with next handler
	l.recordsMu.RLock()
	r, ok := l.records[req.Record()]
	l.recordsMu.RUnlock()

	// Not found in the direct list, check the regex
	if !ok && l.regex != nil {
		select {
		case <-ctx.Done():
			return req, true
		// TODO: Add configurable timeout
		case r, ok = <-l.regex.Match(ctx, req.Record(), time.Second):
		}
	}

	if ok && len(r.IP) > 0 {
		req.r.Answer = append(req.r.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   req.r.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: r.IP,
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
	}

	return req, true
}
