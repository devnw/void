package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"go.devnw.com/ttl"
	"golang.org/x/exp/slog"
)

type Cache struct {
	ctx    context.Context
	logger *slog.Logger
	cache  *ttl.Cache[string, *dns.Msg]
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
	if len(req.r.Question) == 0 {
		err := req.Block()
		if err != nil {
			c.logger.ErrorCtx(ctx, "invalid question",
				slog.String("category", string(CACHE)),
				slog.String("error", err.Error()),
				slog.Group("dns",
					slog.String("name", req.r.Question[0].Name),
					slog.String("type", dns.Type(req.r.Question[0].Qtype).String()),
					slog.String("client", req.client),
					slog.String("server", req.server),
					slog.Int("reqId", int(req.r.Id)),
				),
			)
		}
	}

	r, ok := c.cache.Get(c.ctx, req.Key())
	if !ok || r == nil {
		// Add hook for final response to cache
		req.w = &interceptor{
			ctx:    c.ctx,
			cache:  c.cache,
			logger: c.logger,
			req:    req,
			next:   req.w.WriteMsg, // TODO: Determine if this is the correct pattern
		}

		c.logger.InfoCtx(ctx, "cache miss",
			slog.String("category", string(CACHE)),
			slog.Group("dns",
				slog.String("name", req.r.Question[0].Name),
				slog.String("type", dns.Type(req.r.Question[0].Qtype).String()),
				slog.String("client", req.client),
				slog.String("server", req.server),
				slog.Int("reqId", int(req.r.Id)),
			),
		)

		return req, true
	}

	err := req.Answer(r.SetReply(req.r))
	if err != nil {
		c.logger.ErrorCtx(ctx, "failed to set reply",
			slog.String("category", string(CACHE)),
			slog.String("error", err.Error()),
			slog.Group("dns",
				slog.String("name", req.r.Question[0].Name),
				slog.String("type", dns.Type(req.r.Question[0].Qtype).String()),
				slog.String("client", req.client),
				slog.String("server", req.server),
				slog.Int("reqId", int(req.r.Id)),
				slog.Int("resId", int(r.Id)),
			),
		)
	}

	return req, false
}

// interceptor is a dns.ResponseWriter that caches the response
// for future queries so that they are not re-requesting an updated
// IP for an address that has already been queried.
type interceptor struct {
	ctx    context.Context
	cache  *ttl.Cache[string, *dns.Msg]
	logger *slog.Logger
	req    *Request
	next   func(*dns.Msg) error
	once   sync.Once
}

func (i *interceptor) WriteMsg(res *dns.Msg) (err error) {
	i.once.Do(func() {
		ttl := time.Second * DEFAULTTTL

		if len(res.Answer) > 0 && res.Answer[0].Header() != nil {
			ttl = time.Second * time.Duration(res.Answer[0].Header().Ttl)
		}

		// Set the cache value with record specific TTL
		err = i.cache.SetTTL(i.ctx, i.req.Key(), res, ttl)
		if err != nil {
			return
		}

		i.logger.InfoCtx(i.ctx, "cache hit",
			slog.String("category", string(CACHE)),
			slog.Group("dns",
				slog.String("method", string(WRITE)),
				slog.String("question", i.req.r.Question[0].Name),
				slog.String("type", dns.Type(i.req.r.Question[0].Qtype).String()),
				slog.Duration("ttl", ttl),
				slog.String("client", i.req.client),
				slog.String("server", i.req.server),
				slog.Int("reqId", int(i.req.r.Id)),
			),
		)
	})

	if err != nil {
		return err
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

func (e *CacheEvent) String() string {
	ip := "<missing>"
	if e.Location != nil {
		ip = fmt.Sprintf(" %s %s ", e.Record, e.Location)
	}
	return fmt.Sprintf(
		"CACHE %s %s %s %s",
		strings.ToUpper(string(e.Method)),
		e.Record,
		e.TTL,
		ip,
	)
}

func (e *CacheEvent) Event() string {
	return e.String()
}
