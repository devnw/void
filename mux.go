package main

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"
	"go.devnw.com/ttl"
)

// TODO: Add regex handler

// TODO: Add wildcard handler

type HandleFunc func(dns.ResponseWriter, *dns.Msg)

type local struct {
	ctx   context.Context
	local *ttl.Cache[string, net.IP]
}

func (l *local) Handler(next HandleFunc) HandleFunc {
	l.local.Set(l.ctx, "cisco1.kolhar.net", net.ParseIP("10.10.10.10"))

	return func(w dns.ResponseWriter, req *dns.Msg) {
		record := strings.TrimSuffix(req.Question[0].Name, ".")

		fmt.Printf("%s\n", record)

		// Found in allow list, continue with next handler
		addr, ok := l.local.Get(l.ctx, record)
		if !ok || addr == nil {
			next(w, req)
			return
		}

		res := &dns.Msg{}
		res.SetReply(req)
		res.Answer = append(res.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   req.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: addr.To4(),
		})

		w.WriteMsg(res)
	}
}

type void struct {
	ctx   context.Context
	allow *ttl.Cache[string, bool]
	deny  *ttl.Cache[string, bool]
}

func (v *void) Handler(next HandleFunc) HandleFunc {

	v.deny.Set(v.ctx, "www.google.com", true)
	v.allow.Set(v.ctx, "google.com", true)

	return func(w dns.ResponseWriter, req *dns.Msg) {
		record := strings.TrimSuffix(req.Question[0].Name, ".")

		// Found in allow list, continue with next handler
		allow, ok := v.allow.Get(v.ctx, record)
		if ok && allow {
			next(w, req)
			return
		}

		deny, ok := v.deny.Get(v.ctx, record)
		if !ok || !deny {
			next(w, req)
			return
		}

		// Send to the void
		res := &dns.Msg{}
		res.SetRcode(req, dns.RcodeNameError)

		err := w.WriteMsg(res)
		if err != nil {
			// TODO:
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

		w.WriteMsg(resp)
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

	i.cache.Set(i.ctx, record, res)

	return i.ResponseWriter.WriteMsg(res)
}
