package main

import (
	"context"
	_ "embed"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
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
		cache: ttl.NewCache[string, *dns.Msg](
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

type recursive struct {
	root   *dns.Msg
	cache  *ttl.Cache[string, *dns.Msg]
	client *dns.Client
}

func (r *recursive) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	return nil, false
}

func (r *recursive) authoritative(
	ctx context.Context,
	q *dns.Msg,
) (*dns.Msg, error) {
	msg, ok := r.cache.Get(ctx, key(q))
	if ok {
		return msg, nil
	}

	msg = &dns.Msg{
		Question: []dns.Question{
			{
				Name:   q.Question[0].Name,
				Qtype:  dns.TypeNS,
				Qclass: dns.ClassINET,
			},
		},
	}

	if q.Question[0].Name == "" {
		return r.root, nil
	}

	fmt.Println("recursing", msg.Question[0].Name)

	i := strings.Index(q.Question[0].Name, ".")
	if i > 0 {
		msg.Question[0].Name = q.Question[0].Name[i+1:]
	}

	resp, err := r.authoritative(ctx, msg)
	if err != nil {
		return nil, err
	}

	fmt.Println("resolving", msg.Question[0].Name)

	if resp.Authoritative {
		fmt.Println("++++++++++++++++++++++ AUTH")
		return resp, nil
	}

	spew.Dump(resp)

	var ns string
	for _, rr := range resp.Extra {
		if rr.Header().Rrtype == dns.TypeA {
			ns = rr.(*dns.A).A.String()
			break
		}
	}

	fmt.Printf("ns: %s\n", ns)

	next, _, err := r.client.Exchange(
		q, net.JoinHostPort(ns, "53"),
	)
	if err != nil {
		return nil, err
	}

	k := key(next)
	if k == "" {
		return nil, fmt.Errorf("unable to calculate cache key")
	}

	ttl := time.Second * DEFAULTTTL

	if len(next.Extra) > 0 && next.Extra[0].Header() != nil {
		ttl = time.Second * time.Duration(next.Extra[0].Header().Ttl)
	}

	err = r.cache.SetTTL(ctx, k, next, ttl)
	if err != nil {
		return nil, err
	}

	return next, nil

}
