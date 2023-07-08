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
		cache: ttl.NewCache[string, []dns.RR](
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
	v := fmt.Sprintf("%s:%d", r.Header().Name, r.Header().Class)

	fmt.Printf("rkey: %s\n", v)

	return v
}

func qkey(q dns.Question) string {
	v := fmt.Sprintf("%s:%d", q.Name, q.Qclass)

	fmt.Printf("qkey: %s\n", v)

	return v
}

func soaKey(ns string, class uint16) string {
	v := fmt.Sprintf("%s:%d", ns, class)

	fmt.Printf("soaKey: %s\n", v)

	return v
}

type recursive struct {
	root   *dns.Msg
	cache  *ttl.Cache[string, []dns.RR]
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
	if q.Question[0].Name == "" {
		return r.root, nil
	}

	rr, ok := r.cache.Get(ctx, qkey(q.Question[0]))
	if ok {
		q.Answer = rr
	}

	msg := &dns.Msg{
		Question: []dns.Question{
			{
				Name:   q.Question[0].Name,
				Qtype:  dns.TypeNS,
				Qclass: dns.ClassINET,
			},
		},
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

	ttl := time.Second * DEFAULTTTL
	if len(next.Extra) > 0 && next.Extra[0].Header() != nil {
		ttl = time.Second * time.Duration(next.Extra[0].Header().Ttl)
	}

	answers := map[string][]dns.RR{}
	for _, rr := range next.Extra {
		attl := time.Duration(rr.Header().Ttl) * time.Second
		if attl != ttl {
			ttl = attl
		}

		k := rkey(rr)

		answers[k] = append(answers[k], rr)
	}

	fmt.Printf("ttl: %s\n", ttl)

	for answer, rr := range answers {
		r.cache.SetTTL(ctx, answer, rr, ttl)
	}

	return next, nil

}
