package main

import (
	"context"
	"net"
	"testing"

	"go.devnw.com/event"
)

var (
	cloudflareIpv4S = "1.1.1.1"
	cloudflareIpv4  = net.ParseIP(cloudflareIpv4S)
)

var (
	cloudflareIpv6S = "2606:4700:4700::1111"
	cloudflareIpv6  = net.ParseIP(cloudflareIpv6S)
)

func Test_Up(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tests := map[string]struct {
		address  string
		expected []Upstream
		error    bool
	}{
		"valid-ipv4-no-proto-no-port": {
			address: "1.1.1.1",
			expected: []Upstream{{
				proto:   TCP,
				address: cloudflareIpv4,
				port:    53,
			}},
		},
		"valid-ipv4-no-proto": {
			address: "1.1.1.1:530",
			expected: []Upstream{{
				proto:   UDP,
				address: cloudflareIpv4,
				port:    530,
			}},
		},
		"valid-ipv4-no-port-tcp": {
			address: "tcp://1.1.1.1",
			expected: []Upstream{{
				proto:   TCP,
				address: cloudflareIpv4,
				port:    53,
			}},
		},
		"valid-ipv4-no-port-tcp-tls": {
			address: "tcp-tls://1.1.1.1",
			expected: []Upstream{{
				proto:   TLS,
				address: cloudflareIpv4,
				port:    53,
			}},
		},
		"valid-ipv4-no-port-udp": {
			address: "udp://1.1.1.1",
			expected: []Upstream{{
				proto:   UDP,
				address: cloudflareIpv4,
				port:    53,
			}},
		},
		"invalid-ipv4": {
			address:  "1.1.1.2221",
			expected: []Upstream{},
			error:    true,
		},
		"invalid-ipv4-port": {
			address:  "1.1.1.2221:542",
			expected: []Upstream{},
			error:    true,
		},
		"invalid-ipv4-proto": {
			address:  "tcp://1.1.1.2221",
			expected: []Upstream{},
			error:    true,
		},
		"invalid-ipv4-port-proto": {
			address:  "tcp://1.1.1.2221:542",
			expected: []Upstream{},
			error:    true,
		},
		"invalid-proto-ipv4": {
			address:  "notaproto://1.1.1.1",
			expected: []Upstream{},
			error:    true,
		},
		"invalid-proto-ipv4-port": {
			address:  "notaproto://1.1.1.1:539",
			expected: []Upstream{},
			error:    true,
		},
		"invalid-port-ipv4": {
			address:  "1.1.1.1:500003",
			expected: []Upstream{},
			error:    true,
		},
		"valid-ipv6-no-proto-no-port": {
			address: "2606:4700:4700::1111",
			expected: []Upstream{{
				proto:   TCP,
				address: cloudflareIpv6,
				port:    53,
			}},
		},
		"valid-ipv6-no-proto": {
			address: "2606:4700:4700::1111:5300",
			expected: []Upstream{{
				proto:   UDP,
				address: cloudflareIpv6,
				port:    5300,
			}},
		},
		"valid-ipv6-no-port-tcp": {
			address: "tcp://2606:4700:4700::1111",
			expected: []Upstream{{
				proto:   TCP,
				address: cloudflareIpv6,
				port:    53,
			}},
		},
		"valid-ipv6-no-port-tcp-tls": {
			address: "tcp-tls://2606:4700:4700::1111",
			expected: []Upstream{{
				proto:   TLS,
				address: cloudflareIpv6,
				port:    53,
			}},
		},
		"valid-ipv6-no-port-udp": {
			address: "udp://2606:4700:4700::1111",
			expected: []Upstream{{
				proto:   UDP,
				address: cloudflareIpv6,
				port:    53,
			}},
		},
		"invalid-ipv6": {
			address:  "9892606:4700:4700::1111",
			expected: []Upstream{},
			error:    true,
		},
		"invalid-ipv6-port": {
			address:  "9892606:4700:4700::1111:53",
			expected: []Upstream{},
			error:    true,
		},
		"invalid-ipv6-proto": {
			address:  "tcp://9892606:4700:4700::1111",
			expected: []Upstream{},
			error:    true,
		},
		"invalid-ipv6-port-proto": {
			address:  "tcp://9892606:4700:4700::1111:53",
			expected: []Upstream{},
			error:    true,
		},
		"invalid-proto-ipv6": {
			address:  "notaproto://2606:4700:4700::1111",
			expected: []Upstream{},
			error:    true,
		},
		"invalid-proto-ipv6-port": {
			address:  "notaproto://2606:4700:4700::1111:53",
			expected: []Upstream{},
			error:    true,
		},
		"invalid-port-ipv6": {
			address:  "2606:4700:4700::1111:500003",
			expected: []Upstream{},
			error:    true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			pub := event.NewPublisher(ctx)

			upstreams, err := Up(ctx, pub, test.address)
			if err != nil {
				if test.error {
					return
				}
				t.Error(err)
			}

			if len(upstreams) != len(test.expected) {
				t.Errorf(
					"expected %d upstream, got %d",
					len(test.expected),
					len(upstreams),
				)
			}
		})
	}
}
