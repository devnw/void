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
)

//go:generate wget -O named.root https://www.internic.net/domain/named.root

//go:embed named.root
var namedRoot []byte

func Recursive(
	ctx context.Context,
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
		ctx:  ctx,
		root: ParseZone(zone, ipv4, ipv6),
		cache: ttl.NewCache[string, *dns.Msg](
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
	ctx    context.Context
	root   *dns.Msg
	cache  *ttl.Cache[string, *dns.Msg]
	client *dns.Client
	ipv6   bool
	ipv4   bool
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
	if name == "" || name == "." {
		return r.root, nil
	}

	ns, ok := r.cache.Get(ctx, name)
	if ok {
		fmt.Printf("found %s in cache\n", name)
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

	next, _, err := r.client.Exchange(
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

	ttl := DEFAULTTTL
	for _, rr := range next.Answer {
		if rr.Header().Rrtype == dns.TypeSOA {
			ttl = int(rr.Header().Ttl)
			break
		}

		if rr.Header().Rrtype == dns.TypeNS {
			ttl = int(rr.Header().Ttl)
			break
		}
	}

	rrs := make([]dns.RR, 0, len(next.Extra))

	// Remove any A records from the extra section
	if !r.ipv4 {
		for _, rr := range next.Extra {
			if rr.Header().Rrtype != dns.TypeA {
				rrs = append(rrs, rr)
			}
		}
	}

	// Remove any AAAA records from the extra section
	if !r.ipv6 {
		for _, rr := range next.Extra {
			if rr.Header().Rrtype != dns.TypeAAAA {
				rrs = append(rrs, rr)
			}
		}
	}

	next.Extra = rrs
	r.cache.SetTTL(ctx, name, next, time.Second*time.Duration(ttl))

	return next, nil

}
