package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"go.devnw.com/event"
)

// HandleFunc is a type alias for the handler function
// from the dns package
type HandleFunc func(dns.ResponseWriter, *dns.Msg)

// Convert returns a handler for the DNS server as well as a
// read-only channel of requests to be pushed down the pipeline
func Convert(
	pCtx context.Context,
	pub *event.Publisher,
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
				ctx:   ctx,
				pub:   pub,
				req:   req,
				start: time.Now(),
				next:  w.WriteMsg,
			}
		}

		r := &Request{
			ctx:    ctx,
			cancel: cancel,
			w:      writer,
			r:      req,
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
	ctx   context.Context
	pub   *event.Publisher
	req   *dns.Msg
	start time.Time
	next  func(*dns.Msg) error
}

func (m *metricWriter) WriteMsg(res *dns.Msg) error {
	defer func() {
		m.pub.EventFunc(m.ctx, func() event.Event {
			return &Metric{
				Domain:   m.req.Question[0].Name,
				Duration: time.Since(m.start),
			}
		})
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

// TODO: Add regex handler

// TODO: Convert wildcard to regex
// (\.|^)domain\.tld$

type void struct {
	ctx context.Context

	allow   map[string]*Record
	allowMu sync.RWMutex

	deny   map[string]*Record
	denyMu sync.RWMutex
}

func (v *void) Handler(next HandleFunc) HandleFunc {
	d := &Record{
		Domain:   "www.google.com",
		Eval:     DIRECT,
		Category: "advertising",
		Tags:     []string{"advertising", "google"},
	}

	a := &Record{
		Domain:   "google.com",
		Eval:     DIRECT,
		Category: "advertising",
		Tags:     []string{"advertising", "google"},
	}

	v.denyMu.Lock()
	v.deny[d.Domain] = d
	v.denyMu.Unlock()

	v.allowMu.Lock()
	v.allow[a.Domain] = a
	v.allowMu.Unlock()

	return func(w dns.ResponseWriter, req *dns.Msg) {
		record := strings.TrimSuffix(req.Question[0].Name, ".")

		// Found in allow list, continue with next handler
		_, ok := v.allow[record]
		if ok {
			next(w, req)
			return
		}

		_, ok = v.deny[record]
		if !ok {
			next(w, req)
			return
		}

		// Send to the void
		res := &dns.Msg{}
		res.SetRcode(req, dns.RcodeNameError)

		err := w.WriteMsg(res)
		if err != nil {
			// TODO:
			fmt.Printf("Error: %s\n", err)
		}
	}
}

//type cached struct {
//	ctx   context.Context
//	cache *ttl.Cache[string, *dns.Msg]
//}
//
//func (c *cached) Handler(next HandleFunc) HandleFunc {
//	return func(w dns.ResponseWriter, req *dns.Msg) {
//		record := strings.TrimSuffix(req.Question[0].Name, ".")
//
//		fmt.Printf("%s\n", record)
//		r, ok := c.cache.Get(c.ctx, record)
//		if !ok || r == nil {
//			next(&interceptor{w, c.ctx, c.cache}, req)
//			return
//		}
//
//		resp := r.SetReply(req)
//
//		err := w.WriteMsg(resp)
//		if err != nil {
//			// TODO: Handle error
//			fmt.Printf("Error: %s\n", err)
//		}
//	}
//}
