package main

import (
	"context"
	_ "embed"
	"fmt"
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
) (stream.InterceptFunc[*Request, *Request], error) {
	zone, err := os.Open("named.root")
	if err != nil {
		return nil, err
	}

	r := &recursive{
		root: ParseZone(zone),
		nsCache: ttl.NewCache[string, []dns.RR](
			ctx,
			time.Second*time.Duration(DEFAULTTTL),
			false,
		),
		client: &dns.Client{
			Net:     "udp",
			Timeout: time.Second * 5,
		},
	}

	return r.Intercept, nil
}

func rkey(r dns.RR) string {
	v := fmt.Sprintf("%s:%d", strings.ToLower(r.Header().Name), r.Header().Class)
	return v
}

func qkey(q dns.Question) string {
	v := fmt.Sprintf("%s:%d", strings.ToLower(q.Name), q.Qclass)
	return v
}

func soaKey(ns string, class uint16) string {
	v := fmt.Sprintf("%s:%d", strings.ToLower(ns), class)
	return v
}

type recursive struct {
	root    *dns.Msg
	nsCache *ttl.Cache[string, []dns.RR]
	client  *dns.Client
}

func (r *recursive) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	return nil, false
}

func (r *recursive) exec(
	ctx context.Context,
	q *dns.Msg,
) (*dns.Msg, error) {
	if q.Question[0].Name == "" {
		return r.root, nil
	}

	if q.Question[0].Qtype == dns.TypeNS {
		k := qkey(q.Question[0])
		fmt.Printf("checking cache for %s\n", k)
		rr, ok := r.nsCache.Get(ctx, k)
		if ok {
			fmt.Printf("found %s in cache\n", k)
			q.Extra = rr
			return q, nil
		}
	}

	qName := q.Question[0].Name
	i := strings.Index(qName, ".")
	if i > 0 {
		qName = qName[i+1:]
	}

	resp, err := r.exec(ctx, &dns.Msg{
		Question: []dns.Question{
			{
				Name:   qName,
				Qtype:  dns.TypeNS,
				Qclass: dns.ClassINET,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	fmt.Printf("Resolving: %s\n", qkey(q.Question[0]))

	if resp.Authoritative {
		return resp, nil
	}

	// TODO: this should work for any record coming fom the response
	var ns string
	for _, rr := range resp.Extra {
		if rr.Header().Rrtype == dns.TypeA {
			ns = rr.(*dns.A).A.String()
			break
		}
	}

	next, _, err := r.client.Exchange(
		q, net.JoinHostPort(ns, "53"),
	)
	if err != nil {
		return nil, err
	}

	// Cache the NS records
	ttl := time.Second * DEFAULTTTL
	if len(next.Extra) > 0 && next.Extra[0].Header() != nil {
		ttl = time.Second * time.Duration(next.Extra[0].Header().Ttl)
	}

	nsk := qkey(next.Question[0])

	fmt.Printf("Caching: %s\n", nsk)
	r.nsCache.SetTTL(ctx, nsk, next.Extra, ttl)

	return next, nil

}
