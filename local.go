package main

import (
	"context"
	"sync"

	"go.devnw.com/event"
)

func LocalResolver(
	ctx context.Context,
	pub *event.Publisher,
	records ...*Record,
) (*Local, error) {
	err := checkNil(ctx, pub)
	if err != nil {
		return nil, err
	}

	local := map[string]*Record{}
	for _, r := range records {
		local[r.Domain] = r
	}

	return &Local{
		ctx:   ctx,
		pub:   pub,
		local: local,
	}, nil
}

// Local is the local DNS resolver implementation which handles the locally
// configured DNS records. This does NOT include any blocked or allowed
// records nor does it handle caching upstream DNS records. This is strictly
// for local DNS records.
type Local struct {
	ctx     context.Context
	pub     *event.Publisher
	local   map[string]*Record
	localMu sync.RWMutex
}

// Add adds a local record to the local resolver
// TODO: Setup parallel routine to store records in local LocalResolver
// configuration file
func (l *Local) Add(r *Record) {
	l.localMu.Lock()
	l.local[r.Domain] = r
	l.localMu.Unlock()
}

// Intercept implements the stream.InterceptFunc which
// can then be used throughout the stream library and
// responds to DNS requests for local DNS records
func (l *Local) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	// No local records to check
	if len(l.local) == 0 {
		return req, true
	}

	// Found in allow list, continue with next handler
	l.localMu.RLock()
	r, ok := l.local[req.Record()]
	l.localMu.RUnlock()

	if ok && len(r.IP) > 0 {
		_, err := req.Answer(r.IP)
		if err != nil {
			l.pub.ErrorFunc(ctx, func() error {
				return Error{
					Category: LOCAL,
					Server:   "local-resolver",
					Msg:      "failed to answer request",
					Inner:    err,
					Domain:   req.Record(),
				}
			})
		}

		return nil, false
	}

	return req, true
}
