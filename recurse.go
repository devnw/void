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
		ctx:  ctx,
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
	v := fmt.Sprintf(
		"%s:%d",
		strings.ToLower(r.Header().Name),
		r.Header().Class,
	)
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
	ctx     context.Context
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
	if q.Question[0].Name == "" || q.Question[0].Name == "." {
		r.cacheRR(r.root.Answer)
		r.cacheRR(r.root.Ns)
		r.cacheRR(r.root.Extra)
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

	if resp.Rcode != dns.RcodeSuccess {
		return resp, fmt.Errorf("rcode: %s", dns.RcodeToString[resp.Rcode])
	}

	fmt.Printf("Resolving: %s\n", qkey(q.Question[0]))

	if resp.Authoritative && q.Question[0].Qtype == dns.TypeNS {
		fmt.Printf("Returning: %s\n", qkey(q.Question[0]))
		return resp, nil
	}

	if len(resp.Ns) == 0 {
		return resp, fmt.Errorf("no NS records")
	}

	// get random NS ip from cache
	nsRR := resp.Ns[rand.Intn(len(resp.Ns))]
	spew.Dump(nsRR)

	ip, ok := r.nsCache.Get(r.ctx, rkey(nsRR))
	if !ok {
		fmt.Printf("No IP for %s\n", rkey(nsRR))
		// resolve NS ip
		nsMSG, err := r.exec(ctx, &dns.Msg{
			Question: []dns.Question{
				{
					Name:   nsRR.Header().Name,
					Qtype:  dns.TypeA,
					Qclass: dns.ClassINET,
				},
			},
		})
		if err != nil {
			return nil, err
		}

		spew.Dump(nsMSG)

		if len(nsMSG.Answer) == 0 {
			return resp, fmt.Errorf("no answer")
		}

		ip = nsMSG.Answer
	}

	v := ip[rand.Intn(len(ip))]
	var srv net.IP
	switch v.(type) {
	case *dns.A:
		srv = v.(*dns.A).A
	case *dns.AAAA:
		srv = v.(*dns.AAAA).AAAA
	}

	if srv == nil {
		return resp, fmt.Errorf("no ip")
	}

	next, _, err := r.client.Exchange(
		q, net.JoinHostPort(srv.String(), "53"),
	)
	if err != nil {
		return nil, err
	}

	r.cacheRR(next.Answer)
	r.cacheRR(next.Extra)

	return next, nil

}

func (r *recursive) cacheRR(records []dns.RR) {
	dict := make(map[string][]dns.RR)
	for _, rr := range records {
		key := rkey(rr)

		if _, ok := dict[key]; !ok {
			dict[key] = []dns.RR{}
		}

		dict[key] = append(dict[key], rr)
	}

	for k, v := range dict {
		fmt.Printf("Caching: %s | %+v\n", k, v)
		r.nsCache.Set(r.ctx, k, v)
	}
}
