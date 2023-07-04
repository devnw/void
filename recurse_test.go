package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/miekg/dns"
	"go.devnw.com/ttl"
	"golang.org/x/exp/slog"
)

func Test_loadZoneFile(t *testing.T) {
	zone, err := loadZoneFile(slog.Default(), "")
	if err != nil {
		t.Fatal(err)
	}

	client := &dns.Client{
		Net: "udp",
	}
	records := zone.Records(context.Background())

	for i := 0; i < 10; i++ {
		record := <-records

		if record.Class == "A" {
			msg := &dns.Msg{
				Question: []dns.Question{
					{
						Name:   "com.",
						Qtype:  dns.TypeNS,
						Qclass: dns.ClassINET,
					},
				},
			}

			resp, _, err := client.Exchange(msg, net.JoinHostPort(record.Value, "53"))
			if err != nil {
				t.Fatal(err)
			}

			//			spew.Config.DisableMethods = true
			//			spew.Dump(resp)

			if resp.Authoritative {
				t.Logf("Authoritative: %+v", resp)
			}

			for _, answer := range resp.Extra {
				t.Logf("%+v", answer)

				if ns, ok := answer.(*dns.NS); ok {
					t.Logf("%+v", ns)
					continue
				}

				if a, ok := answer.(*dns.A); ok {
					t.Logf("%+v", a)

					msg := &dns.Msg{
						Question: []dns.Question{
							{
								Name:   "benjiv.com.",
								Qtype:  dns.TypeNS,
								Qclass: dns.ClassINET,
							},
						},
					}

					resp, _, err := client.Exchange(msg, net.JoinHostPort(a.A.String(), "53"))
					if err != nil {
						t.Fatal(err)
					}

					fmt.Println(resp)
					continue
				}

				if aaaa, ok := answer.(*dns.AAAA); ok {
					t.Logf("%+v", aaaa)
					continue
				}

				if cname, ok := answer.(*dns.CNAME); ok {
					t.Logf("%+v", cname)
					continue
				}

				t.Logf("%+v", answer)
			}

			//			t.Logf("%+v", resp)
			//
			//			if len(resp.Answer) == 0 {
			//				t.Fatal("no answer")
			//			}
			//
			//			msg := &dns.Msg{
			//				Question: []dns.Question{
			//					{
			//						Name:   "example.com.",
			//						Qtype:  dns.TypeNS,
			//						Qclass: dns.ClassINET,
			//					},
			//				},
			//			}
			//
			//			resp, _, err := client.Exchange(msg, net.JoinHostPort(resp.Answer[0], "53"))
			//			if err != nil {
			//				t.Fatal(err)
			//			}
			//
		}
	}
}

func Test_resolve(t *testing.T) {
	nam := "test.test.www.benjiv.com."
	ctx := context.Background()

	zone, err := loadZoneFile(slog.Default(), "")
	if err != nil {
		t.Fatal(err)
	}

	r := &recursive{
		zones: zone.Records(ctx),
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

	_, err = r.resolve(ctx, nam)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_root_msg(t *testing.T) {
	_, err := loadZoneFile(slog.Default(), "")
	if err != nil {
		t.Fatal(err)
	}

	spew.Config.DisableMethods = true
	spew.Dump(rootZ)
}
