package main

import (
	"context"
	"net"
	"strings"

	"github.com/miekg/dns"
)

// Request encapsulates all of the request
// data for evaluation in the pipeline
type Request struct {
	ctx    context.Context
	cancel context.CancelFunc
	w      dns.ResponseWriter
	r      *dns.Msg
	record string
}

// Record returns the requested domain
func (r *Request) Record() string {
	if r.record == "" {
		r.record = strings.TrimSuffix(r.r.Question[0].Name, ".")
	}

	return r.record
}

// Block writes a block response to the request
// directly to the original response writer
func (r *Request) Block() error {
	select {
	case <-r.ctx.Done():
		return r.ctx.Err()
	default:
		// Send to the void
		return r.w.WriteMsg(
			(&dns.Msg{}).SetRcode(r.r, dns.RcodeNameError),
		)
	}
}

// Answer returns a response for a specific domain request with the
// provided IP address
func (r *Request) Answer(ip net.IP) (*dns.Msg, error) {
	res := &dns.Msg{}
	res.SetReply(r.r)
	res.Answer = append(res.Answer, &dns.A{
		Hdr: dns.RR_Header{
			Name:   r.r.Question[0].Name,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		A: ip,
	})

	select {
	case <-r.ctx.Done():
		return nil, r.ctx.Err()
	default:
		return res, r.w.WriteMsg(res)
	}
}
