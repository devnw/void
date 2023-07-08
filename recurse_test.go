package main

import (
	"context"
	"fmt"
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
	name := "www.benjiv.com."
	ctx := context.Background()

	zone, err := os.Open("named.root")
	if err != nil {
		t.Fatal(err)
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

	q := &dns.Msg{
		Question: []dns.Question{
			{
				Name:   name,
				Qtype:  dns.TypeA,
				Qclass: dns.ClassINET,
			},
		},
	}

	auth, err := r.exec(ctx, q)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("------------------------- AUTH -------------------------\n")

	spew.Dump(auth)

	fmt.Printf("Authoritative: %+v\n", auth.Authoritative)

	auth, err = r.exec(ctx, q)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("------------------------- AUTH -------------------------\n")

	spew.Dump(auth)

	fmt.Printf("Authoritative: %+v\n", auth.Authoritative)

	auth, err = r.exec(ctx, &dns.Msg{
		Question: []dns.Question{
			{
				Name:   "www.example.com.",
				Qtype:  dns.TypeA,
				Qclass: dns.ClassINET,
			},
		},
	})
	if err != nil {
		t.Error(err)
	}

	if auth != nil {
		fmt.Printf("------------------------- AUTH -------------------------\n")

		spew.Dump(auth)

		fmt.Printf("Authoritative: %+v\n", auth.Authoritative)
	}

	auth, err = r.exec(ctx, &dns.Msg{
		Question: []dns.Question{
			{
				Name:   "go.atomizer.io.",
				Qtype:  dns.TypeA,
				Qclass: dns.ClassINET,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("------------------------- AUTH -------------------------\n")

	spew.Dump(auth)

	fmt.Printf("Authoritative: %+v\n", auth.Authoritative)
}
