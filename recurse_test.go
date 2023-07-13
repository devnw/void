package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/miekg/dns"
	"go.devnw.com/ttl"
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

	zone, err := os.Open("named.root")
	if err != nil {
		t.Fatal(err)
	}

	r := &recursive{
		root: ParseZone(zone, true, false),
		cache: ttl.NewCache[string, *dns.Msg](
			ctx,
			time.Second*time.Duration(DEFAULTTTL),
			false,
		),
		client: &dns.Client{
			Net:     "udp",
			Timeout: time.Second * 5,
		},
		ipv4: true,
		ipv6: false,
	}

	for i := 0; i < 10; i++ {
		auth, err := r.ns(ctx, name)
		if err != nil {
			t.Fatal(err)
		}

		spew.Dump(auth)
	}
}
