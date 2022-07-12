package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"go.devnw.com/ttl"
)

// Convert returns a handler for the DNS server as well as a
// read-only channel of requests to be pushed down the pipeline
func Convert(pCtx context.Context) (HandleFunc, <-chan *Request) {
	out := make(chan *Request)
	go func() {
		// Cleanup the channel when the system exists
		<-pCtx.Done()
		close(out)
	}()

	return func(w dns.ResponseWriter, req *dns.Msg) {
		ctx, cancel := context.WithCancel(pCtx)
		r := &Request{
			ctx:    ctx,
			cancel: cancel,
			w:      w,
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

// TODO: Add regex handler

// TODO: Convert wildcard to regex
// (\.|^)domain\.tld$

// HandleFunc is a type alias for the handler function
// from the dns package
type HandleFunc func(dns.ResponseWriter, *dns.Msg)

type local struct {
	ctx     context.Context
	local   map[string]net.IP
	localMu sync.RWMutex
}

// Intercept implements the stream.InterceptFunc which
// can then be used throughout the stream library and
// responds to DNS requests for local DNS records
func (l *local) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	// Found in allow list, continue with next handler
	l.localMu.RLock()
	ip, ok := l.local[req.Record()]
	l.localMu.RUnlock()

	if ok && ip != nil {
		_, err := req.Answer(ip)
		if err != nil {
			// TODO: Handle error
			fmt.Printf("Error: %s\n", err)
		}

		return nil, false
	}

	return req, true
}

func (l *local) Handler(next HandleFunc) HandleFunc {
	l.localMu.Lock()
	l.local["cisco1.kolhar.net"] = net.ParseIP("10.10.10.10")
	l.localMu.Unlock()

	return func(w dns.ResponseWriter, req *dns.Msg) {
		ctx, cancel := context.WithCancel(l.ctx)
		r := &Request{
			ctx:    ctx,
			cancel: cancel,
			w:      w,
			r:      req,
		}

		// Found in allow list, continue with next handler
		l.localMu.RLock()
		addr, ok := l.local[r.Record()]
		l.localMu.RUnlock()
		if !ok || addr == nil {
			next(w, req)
			return
		}

		_, err := r.Answer(addr)
		if err != nil {
			// TODO: Handle error
			fmt.Printf("Error: %s\n", err)
		}
	}
}

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
		Type:     DIRECT,
		Category: "advertising",
		Tags:     []string{"advertising", "google"},
	}

	a := &Record{
		Domain:   "google.com",
		Type:     DIRECT,
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

type cached struct {
	ctx   context.Context
	cache *ttl.Cache[string, *dns.Msg]
}

func (c *cached) Handler(next HandleFunc) HandleFunc {
	return func(w dns.ResponseWriter, req *dns.Msg) {
		record := strings.TrimSuffix(req.Question[0].Name, ".")

		fmt.Printf("%s\n", record)
		r, ok := c.cache.Get(c.ctx, record)
		if !ok || r == nil {
			next(&interceptor{w, c.ctx, c.cache}, req)
			return
		}

		resp := r.SetReply(req)

		err := w.WriteMsg(resp)
		if err != nil {
			// TODO: Handle error
			fmt.Printf("Error: %s\n", err)
		}
	}
}

// interceptor is a dns.ResponseWriter that caches the response
// for future queries so that they are not re-requesting an updated
// IP for an address that has already been queried
type interceptor struct {
	dns.ResponseWriter
	ctx   context.Context
	cache *ttl.Cache[string, *dns.Msg]
}

func (i *interceptor) WriteMsg(res *dns.Msg) error {
	record := strings.TrimSuffix(res.Question[0].Name, ".")

	err := i.cache.Set(i.ctx, record, res)
	if err != nil {
		return err
	}

	return i.ResponseWriter.WriteMsg(res)
}
