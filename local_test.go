package main

import (
	"context"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/miekg/dns"
	"go.devnw.com/alog"
	"go.devnw.com/event"
)

func Question(t *testing.T, domain string, qtype uint16) *dns.Msg {
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tests := map[string]struct {
		records []*Record
		request *Request
		answer  dns.RR
		pass    bool
	}{
		"match-direct": {
			records: []*Record{{
				Pattern: "test.example.tld",
				Eval:    DIRECT,
				IP:      net.ParseIP("192.168.0.1"),
			}},
			request: &Request{
				ctx:    ctx,
				cancel: cancel,
				w:      &TestWriter{}, // test writer
				r:      Question(t, "test.example.tld.", dns.TypeA),
			},
			answer: &dns.A{
				Hdr: dns.RR_Header{
					Name:   "test.example.tld.",
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    60,
				},
				A: net.ParseIP("192.168.0.1"),
			},
		},
		"nomatch-direct": {
			records: []*Record{{
				Pattern: "test.example.tld",
				Eval:    DIRECT,
				IP:      net.ParseIP("192.168.0.1"),
			}},
			request: &Request{
				ctx:    ctx,
				cancel: cancel,
				w:      &TestWriter{}, // test writer
				r:      Question(t, "mismatch.tld.", dns.TypeA),
			},
			pass: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			pub := event.NewPublisher(ctx)
			defer pub.Close()

			logger, err := alog.New(
				ctx,
				"test",
				alog.DEFAULTTIMEFORMAT,
				time.UTC,
				0,
				alog.TestDestinations(ctx, t)...,
			)
			if err != nil {
				t.Fatal(err)
			}
			defer logger.Close()

			logger.Printc(ctx, pub.ReadEvents(0).Interface())
			logger.Errorc(ctx, pub.ReadErrors(0).Interface())

			local, err := LocalResolver(ctx, pub, test.records...)
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

			if !reflect.DeepEqual(w.response.Answer[0], test.answer) {
				t.Fatalf("expected %v, got %v", test.answer, w.response)
			}
		})
	}
}
