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

func (r *recursive) resolve(
	ctx context.Context, q *dns.Msg,
) (*dns.Msg, error) {
	k := key(q)
	if msg, ok := r.addrCache.Get(ctx, k); ok {
		return msg, nil
	}

	ns, err := r.ns(ctx, q.Question[0].Name)
	if err != nil {
		r.logger.ErrorCtx(
			ctx, "failed to resolve NS",
			slog.String("name", q.Question[0].Name),
			slog.String("error", err.Error()))
		return nil, err
	}

	if len(ns.Ns) == 0 {
		if len(ns.Answer) == 0 {
			return nil, fmt.Errorf("no NS records")
		}

		q.Answer = append(q.Answer, ns.Answer...)
		for _, a := range ns.Answer {
			if a.Header().Rrtype == dns.TypeCNAME {
				return r.resolve(ctx, &dns.Msg{
					Question: []dns.Question{
						{
							Name:   a.(*dns.CNAME).Target,
							Qtype:  dns.TypeA,
							Qclass: dns.ClassINET,
						},
					},
				})
			}
		}
	}

	// Return the NS record if it's what is being asked for.
	if q.Question[0].Qtype == dns.TypeNS {
		return ns, nil
	}

	// Check for an authoritative answer.
	soa, ok := ns.Ns[0].(*dns.SOA)
	if !ok {
		spew.Dump(ns)
		return nil, fmt.Errorf("no SOA record")
	}

	nsIP, err := r.resolve(ctx, &dns.Msg{
		Question: []dns.Question{
			{
				Name:   soa.Ns,
				Qtype:  dns.TypeA,
				Qclass: dns.ClassINET,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	spew.Dump(nsIP)

	if len(nsIP.Answer) == 0 {
		return nil, fmt.Errorf("no answer")
	}

	naA, ok := nsIP.Answer[rand.Intn(len(nsIP.Answer))].(*dns.A)
	if !ok {
		return nil, fmt.Errorf("no A record")
	}

	// Exchange the original query with the NS.
	resp, _, err := r.client.Exchange(q, net.JoinHostPort(naA.A.String(), "53"))
	if err != nil {
		return nil, err
	}

	return resp, nil
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
	if len(resp.Extra) > 0 {
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
	nsIPs := map[string]map[uint16][]dns.RR{}

	// NOTE: The A, and AAAA records for the specific name server
	// responses also need to be stored if they're returned in the
	// extra part of a response as well

	for _, rr := range next.Extra {
		switch rr := rr.(type) {
		case *dns.A:
			if !r.ipv4 {
				continue
			}

			rrs = append(rrs, rr)
			if _, ok := nsIPs[rr.Hdr.Name]; !ok {
				nsIPs[rr.Hdr.Name] = map[uint16][]dns.RR{}
			}

			nsIPs[rr.Hdr.Name][dns.TypeA] = append(
				nsIPs[rr.Hdr.Name][dns.TypeA], rr)
		case *dns.AAAA:
			if !r.ipv6 {
				continue
			}

			rrs = append(rrs, rr)
			if _, ok := nsIPs[rr.Hdr.Name]; !ok {
				nsIPs[rr.Hdr.Name] = map[uint16][]dns.RR{}
			}

			nsIPs[rr.Hdr.Name][dns.TypeAAAA] = append(
				nsIPs[rr.Hdr.Name][dns.TypeAAAA], rr)
		}
	}

	for k, v := range nsIPs {
		msg := &dns.Msg{
			Question: []dns.Question{
				{
					Name:   k,
					Qtype:  dns.TypeA,
					Qclass: dns.ClassINET,
				},
			},
		}

		for rrk, rr := range v {
			if rrk != dns.TypeA {
				msg.Question[0].Qtype = rrk
			}

			msg.Answer = append(msg.Answer, rr...)
		}

		r.logger.DebugCtx(ctx, "adding record to addr cache",
			slog.String("name", k),
			slog.String("type", dns.Type(msg.Question[0].Qtype).String()),
			slog.String("class", dns.Class(msg.Question[0].Qclass).String()),
			slog.String("rrs", spew.Sdump(msg.Answer)))

		r.addrCache.SetTTL(ctx, key(msg), msg, time.Second*time.Duration(ttl))
	}

	next.Extra = rrs
	r.nsCache.SetTTL(ctx, name, next, time.Second*time.Duration(ttl))

	return next, nil

}
