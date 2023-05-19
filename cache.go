package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/miekg/dns"
	"go.devnw.com/ttl"
)

type Cache struct {
	ctx    context.Context
	logger Logger
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
			c.logger.Errorw(
				"invalid question",
				"category", CACHE,
				"request", req.String(),
				"error", err,
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

		return req, true
	}

	err := req.Answer(r.SetReply(req.r))
	if err != nil {
		c.logger.Errorw(
			"failed to set reply",
			"category", CACHE,
			"request", req.String(),
			"error", err,
		)
	}

	w, ok := (ctx.Value("influxdb.writer")).(api.WriteAPI)
	if !ok {
		c.logger.Errorw(
			"failed to get influxdb writer",
			"category", BLOCK,
			"record", req.String(),
		)
	}

	w.WritePoint(influxdb2.NewPointWithMeasurement("cache").
		AddField("server", req.server).
		AddField("client", req.client).
		AddField("question", req.r.Question[0].Name).
		AddField("type", dns.Type(req.r.Question[0].Qtype).String()).
		AddField("class", dns.Class(req.r.Question[0].Qclass).String()).
		// TODO:
		// AddField("rtt", rtt).
		SetTime(time.Now()))

	return req, false
}

// interceptor is a dns.ResponseWriter that caches the response
// for future queries so that they are not re-requesting an updated
// IP for an address that has already been queried.
type interceptor struct {
	ctx    context.Context
	cache  *ttl.Cache[string, *dns.Msg]
	logger Logger
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

		i.logger.Debugw(
			"cache",
			"method", WRITE,
			"record", i.req.r.Question[0].Name,
			"ttl", ttl,
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
