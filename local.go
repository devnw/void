package main

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/miekg/dns"
)

func LocalResolver() (*Local, error) {
	return &Local{}, nil
}

type Local struct {
	ctx     context.Context
	local   map[string]net.IP
	localMu sync.RWMutex
}

// Intercept implements the stream.InterceptFunc which
// can then be used throughout the stream library and
// responds to DNS requests for local DNS records
func (l *Local) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	// Found in allow list, continue with next handler
	l.localMu.RLock()
	ip, ok := l.local[req.Record()]
	l.localMu.RUnlock()

	if ok && len(ip) > 0 {
		_, err := req.Answer(ip)
		if err != nil {
			// TODO: Handle error
			fmt.Printf("Error: %s\n", err)
		}

		return nil, false
	}

	return req, true
}

func (l *Local) Handler(next HandleFunc) HandleFunc {
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
