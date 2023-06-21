package main

import (
	"context"
	"net"
	"testing"

	"github.com/miekg/dns"
	"golang.org/x/exp/slog"
)

func Question(t *testing.T, domain string, qtype uint16) *dns.Msg {
	t.Helper()
	t.Logf("%s %s", domain, dns.Type(qtype))

	m := new(dns.Msg)
	m.SetQuestion(domain, qtype)

	return m
}

type TestWriter struct {
	response *dns.Msg
}

func (tw *TestWriter) WriteMsg(res *dns.Msg) error {
	tw.response = res
	return nil
}

func Test_Local_Intercept(t *testing.T) {
	pctx, pcancel := context.WithCancel(context.Background())
	defer pcancel()

	tests := map[string]struct {
		records []*Record
		request *Request
		answer  dns.RR
		pass    bool
	}{
		"match-direct": {
			records: []*Record{{
				Pattern: "test.example.tld",
				Type:    DIRECT,
				IP:      net.ParseIP("192.168.0.1"),
			}},
			request: &Request{
				ctx:    pctx,
				cancel: pcancel,
				w:      &TestWriter{}, // test writer
				r:      Question(t, "test.example.tld.", dns.TypeA),
			},
			answer: &dns.A{
				Hdr: dns.RR_Header{
					Name:   "test.example.tld.",
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    DEFAULTTTL,
				},
				A: net.ParseIP("192.168.0.1"),
			},
		},
		"nomatch-direct": {
			records: []*Record{{
				Pattern: "test.example.tld",
				Type:    DIRECT,
				IP:      net.ParseIP("192.168.0.1"),
			}},
			request: &Request{
				ctx:    pctx,
				cancel: pcancel,
				w:      &TestWriter{}, // test writer
				r:      Question(t, "mismatch.tld.", dns.TypeA),
			},
			pass: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(pctx)
			defer cancel()

			local, err := LocalResolver(ctx, slog.Default(), test.records...)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			r, pass := local.Intercept(ctx, test.request)
			if pass {
				if !test.pass {
					t.Fatalf("expected match; got pass")
				}

				return
			}

			// Passthrough
			if r != nil {
				t.Fatalf("expected nil, got %v", r)
			}

			// Check answer
			w, ok := test.request.w.(*TestWriter)
			if !ok {
				t.Fatalf("expected TestWriter, got %T", test.request.w)
			}

			if w.response.Answer[0].String() != test.answer.String() {
				t.Fatalf(
					"expected\n[%s]\ngot\n[%s]",
					test.answer.String(),
					w.response.Answer[0].String(),
				)
			}
		})
	}
}
