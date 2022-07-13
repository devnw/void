package main

import (
	"context"
	"net"
	"testing"
)

func Test_Up(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tests := map[string]struct {
		address  string
		expected []Upstream
		error    error
	}{
		"valid-ipv4-no-proto-no-port": {
			address: "1.1.1.1",
			expected: []Upstream{{
				Network: TCP,
				Address: net.ParseIP("1.1.1.1"),
				Port:    53,
			}},
			error: nil,
		},
		"valid-ipv4-no-proto": {
			address: "1.1.1.1:53",
			expected: []Upstream{{
				Network: UDP,
				Address: net.ParseIP("1.1.1.1"),
				Port:    53,
			}},
			error: nil,
		},
		"valid-ipv4-no-port-tcp": {
			address: "tcp://1.1.1.1",
			expected: []Upstream{{
				Network: TCP,
				Address: net.ParseIP("1.1.1.1"),
				Port:    53,
			}},
			error: nil,
		},
		"valid-ipv4-no-port-tcp-tls": {
			address: "tcp-tls://1.1.1.1",
			expected: []Upstream{{
				Network: TLS,
				Address: net.ParseIP("1.1.1.1"),
				Port:    53,
			}},
			error: nil,
		},
		"valid-ipv4-no-port-udp": {
			address: "udp://1.1.1.1",
			expected: []Upstream{{
				Network: UDP,
				Address: net.ParseIP("1.1.1.1"),
				Port:    53,
			}},
			error: nil,
		},
		"valid-ipv6-no-proto-no-port": {
			address: "2606:4700:4700::1111",
			expected: []Upstream{{
				Network: TCP,
				Address: net.ParseIP("2606:4700:4700::1111"),
				Port:    53,
			}},
			error: nil,
		},
		"valid-ipv6-no-proto": {
			address: "2606:4700:4700::1111:53",
			expected: []Upstream{{
				Network: UDP,
				Address: net.ParseIP("2606:4700:4700::1111"),
				Port:    53,
			}},
			error: nil,
		},
		"valid-ipv6-no-port-tcp": {
			address: "tcp://2606:4700:4700::1111",
			expected: []Upstream{{
				Network: TCP,
				Address: net.ParseIP("2606:4700:4700::1111"),
				Port:    53,
			}},
			error: nil,
		},
		"valid-ipv6-no-port-tcp-tls": {
			address: "tcp-tls://2606:4700:4700::1111",
			expected: []Upstream{{
				Network: TLS,
				Address: net.ParseIP("2606:4700:4700::1111"),
				Port:    53,
			}},
			error: nil,
		},
		"valid-ipv6-no-port-udp": {
			address: "udp://2606:4700:4700::1111",
			expected: []Upstream{{
				Network: UDP,
				Address: net.ParseIP("2606:4700:4700::1111"),
				Port:    53,
			}},
			error: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			upstreams, err := Up(ctx, test.address)
			if err != nil {
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
