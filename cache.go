package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"go.devnw.com/event"
	"go.devnw.com/ttl"
)

type Cache struct {
	ctx   context.Context
	pub   *event.Publisher
	cache *ttl.Cache[string, *dns.Msg]
}

// Intercept is the cache intercept func which attempts to first pull
// the response from the cache if it exists. If it is no longer in the
// cache then the request is passed down the pipeline after wrapping
// the request with an interceptor. The interceptor is responsible for
// caching the response on the way back to the client.
func (c *Cache) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	fmt.Println("Cache intercept")
	if len(req.r.Question) == 0 {
		err := req.Block()

		if err != nil {
			c.pub.ErrorFunc(ctx, func() error {
				return Error{
					Category: CACHE,
					Server:   "cache",
					Msg:      "failed to answer request",
					Inner:    err,
					Record:   req.String(),
				}
			})
		}
	}

	r, ok := c.cache.Get(c.ctx, req.Key())
	if !ok || r == nil {
		// Add hook for final response to cache
		req.w = &interceptor{
			c.ctx,
			c.cache,
			c.pub,
			req,
			req.w.WriteMsg, // TODO: Determine if this is the correct pattern
		}

		return req, true
	}

	err := req.Answer(r)
	if err != nil {
		c.pub.ErrorFunc(ctx, func() error {
			return Error{
				Category: CACHE,
				Server:   "cache",
				Msg:      "failed to answer request",
				Inner:    err,
				Record:   req.String(),
			}
		})
	}

	return req, false
}

// interceptor is a dns.ResponseWriter that caches the response
// for future queries so that they are not re-requesting an updated
// IP for an address that has already been queried
type interceptor struct {
	ctx   context.Context
	cache *ttl.Cache[string, *dns.Msg]
	pub   *event.Publisher
	req   *Request
	next  func(*dns.Msg) error
}

func (i *interceptor) WriteMsg(res *dns.Msg) error {
	if len(res.Answer) == 0 {
		return i.next(res)
	}

	// Store the response using the information from the
	// first answer only
	a := res.Answer[0]
	if a.Header() != nil {
		ttl := time.Second * time.Duration(a.Header().Ttl)

		// Set the cache value with record specific TTL
		err := i.cache.SetTTL(i.ctx, i.req.Key(), res, ttl)
		if err != nil {
			return err
		}

		i.pub.EventFunc(i.ctx, func() event.Event {
			return &CacheEvent{
				Method: WRITE,
				Record: i.req.String(),
				TTL:    ttl,
			}
		})
	}

	return i.next(res)
}

type CacheAction string

const (
	READ  CacheAction = "read"
	WRITE CacheAction = "write"
)

type CacheEvent struct {
	Method   CacheAction
	Record   string
	Location net.IP
	TTL      time.Duration
}

func (e *CacheEvent) Event() string {
	return fmt.Sprintf(
		"CACHE %s %s %s %s",
		strings.ToUpper(string(e.Method)),
		e.Location.String(),
		e.TTL,
		e.Record,
	)
}
