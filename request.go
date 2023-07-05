package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/miekg/dns"
)

// TODO: Setup initializer for request, move to interface? etc...
// Add invalidation for the *dns.Msg in the initializer

type Writer interface {
	WriteMsg(res *dns.Msg) error
}

// Request encapsulates all of the request
// data for evaluation in the pipeline.
type Request struct {
	ctx    context.Context
	cancel context.CancelFunc
	id     uuid.UUID
	w      Writer
	r      *dns.Msg
	record string
	server string
	client string
}

// Record returns the requested domain.
func (r *Request) Record() string {
	if r.record == "" {
		r.record = strings.TrimSuffix(r.r.Question[0].Name, ".")
	}

	return r.record
}

// Key returns a unique identifier for the request which is an aggregate
// of the name, type, and class.
func (r *Request) Key() string {
	return key(r.r)
}

func key(msg *dns.Msg) string {
	// TODO: Add validation?
	q := msg.Question[0]

	return fmt.Sprintf("%s:%d:%d", q.Name, q.Qtype, q.Qclass)
}

func (r *Request) String() string {
	return fmt.Sprintf(
		"%s %s %s",
		r.r.Question[0].Name,
		dns.Type(r.r.Question[0].Qtype).String(),
		dns.Class(r.r.Question[0].Qclass).String(),
	)
}

// Block writes a block response to the request
// directly to the original response writer.
func (r *Request) Block() error {
	select {
	case <-r.ctx.Done():
		return r.ctx.Err()
	default:
		r.cancel()

		// Send to the void
		return r.w.WriteMsg(
			(&dns.Msg{}).SetRcode(r.r, dns.RcodeNameError),
		)
	}
}

// Answer returns a response for a specific domain request with the
// provided IP address.
func (r *Request) Answer(msg *dns.Msg) error {
	select {
	case <-r.ctx.Done():
		return r.ctx.Err()
	default:
		r.cancel()

		return r.w.WriteMsg(msg)
	}
}
