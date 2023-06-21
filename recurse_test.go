package main

import (
	"context"
	"net"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/miekg/dns"
	"golang.org/x/exp/slog"
)

func Test_loadZoneFile(t *testing.T) {
	zone, err := loadZoneFile(slog.Default(), "")
	if err != nil {
		t.Fatal(err)
	}

	spew.Dump(zone)

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

			t.Logf("%+v", resp)

		}
	}
}
