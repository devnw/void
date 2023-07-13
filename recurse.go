package main

import (
	"context"
	_ "embed"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
	"go.atomizer.io/stream"
	"go.devnw.com/ttl"
	"golang.org/x/exp/slog"
)

//go:generate wget -O named.root https://www.internic.net/domain/named.root

//go:embed named.root
var namedRoot []byte

func Recursive(
	ctx context.Context,
	logger *slog.Logger,
	zonefile string,
	ipv4, ipv6 bool,
) (stream.InterceptFunc[*Request, *Request], error) {
	if !ipv4 && !ipv6 {
		return nil, fmt.Errorf("must specify at least one IP protocol")
	}

	zone, err := os.Open("named.root")
	if err != nil {
		return nil, err
	}

	r := &recursive{
		ctx:    ctx,
		logger: logger,
		root:   ParseZone(zone, ipv4, ipv6),
		nsCache: ttl.NewCache[string, *dns.Msg](
			ctx,
			time.Second*time.Duration(DEFAULTTTL),
			false,
		),
		client: &dns.Client{
			Net:     "udp",
			Timeout: time.Second * 5,
		},
		ipv4: ipv4,
		ipv6: ipv6,
	}

	return r.Intercept, nil
}

type recursive struct {
	ctx       context.Context
	logger    *slog.Logger
	root      *dns.Msg
	nsCache   *ttl.Cache[string, *dns.Msg]
	addrCache *ttl.Cache[string, *dns.Msg]
	client    *dns.Client
	ipv6      bool
	ipv4      bool
}

func (r *recursive) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	return nil, false
}

func (r *recursive) ns(
	ctx context.Context,
	name string,
) (*dns.Msg, error) {
	r.logger.DebugCtx(ctx, "recursive ns resolution begin",
		slog.String("name", name))

	defer r.logger.DebugCtx(ctx, "recursive ns resolution end",
		slog.String("name", name))

	if name == "" || name == "." {
		r.logger.DebugCtx(ctx, "ns", "name",
			slog.String("name", name),
			slog.String("ns", "."))
		return r.root, nil
	}

	ns, ok := r.nsCache.Get(ctx, name)
	if ok {
		r.logger.DebugCtx(ctx, "ns cache hit", slog.String("name", name))
		return ns, nil
	}

	qName := name
	i := strings.Index(qName, ".")
	if i > 0 {
		qName = qName[i+1:]
	}

	resp, err := r.ns(ctx, qName)
	if err != nil {
		return nil, err
	}

	next := resp
	if len(resp.Extra) > 0 || !resp.Authoritative {
		var ip net.IP
		rr := resp.Extra[rand.Intn(len(resp.Extra))]
		switch rr := rr.(type) {
		case *dns.A:
			ip = rr.A
		case *dns.AAAA:
			ip = rr.AAAA
		default:
			return nil, fmt.Errorf("unknown type %T", rr)
		}

		next, _, err = r.client.Exchange(
			&dns.Msg{
				Question: []dns.Question{
					{
						Name:   name,
						Qtype:  dns.TypeNS,
						Qclass: dns.ClassINET,
					},
				},
			}, net.JoinHostPort(ip.String(), "53"),
		)
		if err != nil {
			return nil, err
		}

	}

	ttl := DEFAULTTTL
	if len(next.Answer) > 0 {
		ttl = int(next.Answer[0].Header().Ttl)
	}

	rrs := make([]dns.RR, 0, len(next.Extra))

	// NOTE: The A, and AAAA records for the specific name server
	// responses also need to be stored if they're returned in the
	// extra part of a response as well

	for _, rr := range next.Extra {
		if r.ipv4 && rr.Header().Rrtype == dns.TypeA {
			rrs = append(rrs, rr)
		}

		if r.ipv6 && rr.Header().Rrtype == dns.TypeAAAA {
			rrs = append(rrs, rr)
		}
	}

	next.Extra = rrs
	r.nsCache.SetTTL(ctx, name, next, time.Second*time.Duration(ttl))

	return next, nil

}
