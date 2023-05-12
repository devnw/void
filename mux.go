package main

import (
	"context"
	"fmt"
	"time"

	"github.com/miekg/dns"
)

// HandleFunc is a type alias for the handler function
// from the dns package.
type HandleFunc func(dns.ResponseWriter, *dns.Msg)

// Convert returns a handler for the DNS server as well as a
// read-only channel of requests to be pushed down the pipeline.
func Convert(
	pCtx context.Context,
	logger Logger,
	metrics bool,
) (HandleFunc, <-chan *Request) {
	out := make(chan *Request)
	go func() {
		// Cleanup the channel when the system exists
		<-pCtx.Done()
		close(out)
	}()

	return func(w dns.ResponseWriter, req *dns.Msg) {
		ctx, cancel := context.WithCancel(pCtx)

		var writer Writer = w
		if metrics {
			writer = &metricWriter{
				ctx:    ctx,
				logger: logger,
				req:    req,
				start:  time.Now(),
				next:   w.WriteMsg,
			}
		}

		r := &Request{
			ctx:    ctx,
			cancel: cancel,
			w:      writer,
			r:      req,
			server: w.LocalAddr().String(),
			client: w.RemoteAddr().String(),
		}

		select {
		case <-pCtx.Done():
			w.Close()

		case out <- r:
			// TODO: Log request?
		}
	}, out
}

type metricWriter struct {
	ctx    context.Context
	logger Logger
	req    *dns.Msg
	start  time.Time
	next   func(*dns.Msg) error
}

func (m *metricWriter) WriteMsg(res *dns.Msg) error {
	defer func() {
		m.logger.Debugw(
			"wrote response",
			"duration", time.Since(m.start),
			"name", res.Question[0].Name,
			"type", dns.Type(res.Question[0].Qtype),
			"answers", res.Answer,
		)
	}()

	return m.next(res)
}

type Metric struct {
	Domain   string
	Duration time.Duration
}

func (m *Metric) String() string {
	return fmt.Sprintf("%s %s", m.Domain, m.Duration)
}

func (m *Metric) Event() string {
	return m.String()
}
