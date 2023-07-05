package main

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/miekg/dns"
	"go.devnw.com/ttl"
	"golang.org/x/exp/slog"
)

//func Test_loadZoneFile(t *testing.T) {
//	zone, err := loadZoneFile(slog.Default(), "")
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	client := &dns.Client{
//		Net: "udp",
//	}
//
//	for i := 0; i < 10; i++ {
//		msg := &dns.Msg{
//			Question: []dns.Question{
//				{
//					Name:   "com.",
//					Qtype:  dns.TypeNS,
//					Qclass: dns.ClassINET,
//				},
//			},
//		}
//
//		resp, _, err := client.Exchange(msg, net.JoinHostPort(record.Value, "53"))
//		if err != nil {
//			t.Fatal(err)
//		}
//
//		//			spew.Config.DisableMethods = true
//		//			spew.Dump(resp)
//
//		if resp.Authoritative {
//			t.Logf("Authoritative: %+v", resp)
//		}
//
//		for _, answer := range resp.Extra {
//			t.Logf("%+v", answer)
//
//			if ns, ok := answer.(*dns.NS); ok {
//				t.Logf("%+v", ns)
//				continue
//			}
//
//			if a, ok := answer.(*dns.A); ok {
//				t.Logf("%+v", a)
//
//				msg := &dns.Msg{
//					Question: []dns.Question{
//						{
//							Name:   "benjiv.com.",
//							Qtype:  dns.TypeNS,
//							Qclass: dns.ClassINET,
//						},
//					},
//				}
//
//				resp, _, err := client.Exchange(msg, net.JoinHostPort(a.A.String(), "53"))
//				if err != nil {
//					t.Fatal(err)
//				}
//
//				fmt.Println(resp)
//				continue
//			}
//
//			if aaaa, ok := answer.(*dns.AAAA); ok {
//				t.Logf("%+v", aaaa)
//				continue
//			}
//
//			if cname, ok := answer.(*dns.CNAME); ok {
//				t.Logf("%+v", cname)
//				continue
//			}
//
//			t.Logf("%+v", answer)
//		}
//
//		//			t.Logf("%+v", resp)
//		//
//		//			if len(resp.Answer) == 0 {
//		//				t.Fatal("no answer")
//		//			}
//		//
//		//			msg := &dns.Msg{
//		//				Question: []dns.Question{
//		//					{
//		//						Name:   "example.com.",
//		//						Qtype:  dns.TypeNS,
//		//						Qclass: dns.ClassINET,
//		//					},
//		//				},
//		//			}
//		//
//		//			resp, _, err := client.Exchange(msg, net.JoinHostPort(resp.Answer[0], "53"))
//		//			if err != nil {
//		//				t.Fatal(err)
//		//			}
//		//
//	}
//}

func Test_resolve(t *testing.T) {
	name := "test.test.www.benjiv.com."
	ctx := context.Background()

	zone, err := loadZoneFile(slog.Default(), "")
	if err != nil {
		t.Fatal(err)
	}

	r := &recursive{
		root: zone.Msg(),
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

	auth, err := r.authoritative(ctx, &dns.Msg{
		Question: []dns.Question{
			{
				Name:   name,
				Qtype:  dns.TypeA,
				Qclass: dns.ClassINET,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var ns string
	for _, rr := range resp.Extra {
		if rr.Header().Rrtype == dns.TypeA {
			ns = rr.(*dns.A).A.String()
			break
		}
	}

	resp, _, err := r.client.Exchange(
		q, net.JoinHostPort(ns, "53"),
	)
	if err != nil {
		return nil, err
	}
}

func Test_root_msg(t *testing.T) {
	zones, err := loadZoneFile(slog.Default(), "")
	if err != nil {
		t.Fatal(err)
	}

	spew.Dump(zones)
}
