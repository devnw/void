package main

import (
	"context"
	"net"
	"testing"
)

var cloudflareIpv4S = "1.1.1.1"
var cloudflareIpv4 = net.ParseIP(cloudflareIpv4S)

var cloudflareIpv6S = "2606:4700:4700::1111"
var cloudflareIpv6 = net.ParseIP(cloudflareIpv6S)

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
				Network: TCP,
				Address: cloudflareIpv4,
				Port:    53,
			}},
		},
		"valid-ipv4-no-proto": {
			address: "1.1.1.1:53",
			expected: []Upstream{{
				Network: UDP,
				Address: cloudflareIpv4,
				Port:    53,
			}},
		},
		"valid-ipv4-no-port-tcp": {
			address: "tcp://1.1.1.1",
			expected: []Upstream{{
				Network: TCP,
				Address: cloudflareIpv4,
				Port:    53,
			}},
		},
		"valid-ipv4-no-port-tcp-tls": {
			address: "tcp-tls://1.1.1.1",
			expected: []Upstream{{
				Network: TLS,
				Address: cloudflareIpv4,
				Port:    53,
			}},
		},
		"valid-ipv4-no-port-udp": {
			address: "udp://1.1.1.1",
			expected: []Upstream{{
				Network: UDP,
				Address: cloudflareIpv4,
				Port:    53,
			}},
		},
		"valid-ipv6-no-proto-no-port": {
			address: "2606:4700:4700::1111",
			expected: []Upstream{{
				Network: TCP,
				Address: cloudflareIpv6,
				Port:    53,
			}},
		},
		"valid-ipv6-no-proto": {
			address: "2606:4700:4700::1111:5300",
			expected: []Upstream{{
				Network: UDP,
				Address: cloudflareIpv6,
				Port:    5300,
			}},
		},
		"valid-ipv6-no-port-tcp": {
			address: "tcp://2606:4700:4700::1111",
			expected: []Upstream{{
				Network: TCP,
				Address: cloudflareIpv6,
				Port:    53,
			}},
		},
		"valid-ipv6-no-port-tcp-tls": {
			address: "tcp-tls://2606:4700:4700::1111",
			expected: []Upstream{{
				Network: TLS,
				Address: cloudflareIpv6,
				Port:    53,
			}},
		},
		"valid-ipv6-no-port-udp": {
			address: "udp://2606:4700:4700::1111",
			expected: []Upstream{{
				Network: UDP,
				Address: cloudflareIpv6,
				Port:    53,
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
			upstreams, err := Up(ctx, test.address)
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
