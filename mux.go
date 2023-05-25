package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime/debug"
	"strings"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/exp/slog"
)

// HandleFunc is a type alias for the handler function
// from the dns package.
type HandleFunc func(dns.ResponseWriter, *dns.Msg)

// Convert returns a handler for the DNS server as well as a
// read-only channel of requests to be pushed down the pipeline.
func Convert(
	pCtx context.Context,
	logger *slog.Logger,
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

		host, _, err := net.SplitHostPort(w.RemoteAddr().String())
		if err != nil {
			logger.ErrorCtx(ctx,
				"invalid client ip",
				slog.String("category", "metrics"),
				slog.Group("dns",
					slog.String("question", req.Question[0].Name),
					slog.String("type", dns.Type(req.Question[0].Qtype).String()),
					slog.String("client", w.RemoteAddr().String()),
					slog.String("server", w.LocalAddr().String()),
					slog.Int("reqId", int(req.Id)),
				),
				slog.String("error", err.Error()),
			)
			return
		}

		r := &Request{
			ctx:    ctx,
			cancel: cancel,
			w:      writer,
			r:      req,
			server: fmt.Sprintf(
				"%s://%s",
				w.LocalAddr().Network(),
				w.LocalAddr().String(),
			),
			client: fmt.Sprintf(
				"%s://%s",
				w.RemoteAddr().Network(),
				host,
			),
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
	logger *slog.Logger
	req    *dns.Msg
	start  time.Time
	next   func(*dns.Msg) error
}

func (m *metricWriter) WriteMsg(res *dns.Msg) error {
	if res == nil {
		return errors.New("nil response")
	}

	defer func() {
		r := recover()
		if r != nil {
			m.logger.ErrorCtx(m.ctx,
				"panic in writer",
				slog.String("category", "metrics"),
				slog.Group("dns",
					slog.String("question", m.req.Question[0].Name),
					slog.String("type", dns.Type(m.req.Question[0].Qtype).String()),
					slog.Int("reqId", int(m.req.Id)),
					slog.Int("resId", int(res.Id)),
				),
				slog.String("error", fmt.Sprintf("%v", r)),
				slog.String("stack", string(debug.Stack())),
			)
			return
		}

		answers := []string{}
		for _, a := range res.Answer {
			answers = append(answers, a.String())
		}

		m.logger.DebugCtx(m.ctx,
			"wrote response",
			slog.String("category", "metrics"),
			slog.Duration("duration", time.Since(m.start)),
			slog.Group("dns",
				slog.String("question", m.req.Question[0].Name),
				slog.String("type", dns.Type(m.req.Question[0].Qtype).String()),
				slog.String("answers", strings.Join(answers, ", ")),
				slog.Int("reqId", int(m.req.Id)),
				slog.Int("resId", int(res.Id)),
			),
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
