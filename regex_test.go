package main

import (
	"context"
	"testing"
	"time"

	"go.devnw.com/event"
)

func Test_Match(t *testing.T) {
	tests := map[string]struct {
		regex   string
		input   string
		matched bool
	}{
		"match-ipv4": {
			regex:   "^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$",
			input:   "1.1.1.1",
			matched: true,
		},
		"match-ipv4_": {
			regex:   ipv4Reg,
			input:   "1.1.1.1",
			matched: true,
		},
		"match-ipv4_bad": {
			regex:   ipv4Reg,
			input:   "asdf1.1.1",
			matched: false,
		},
		"wildcard_domain": {
			regex:   `(\.|^)domain\.tld$`,
			input:   "domain.tld",
			matched: true,
		},
		"wildcard_domain_sub": {
			regex:   `(\.|^)domain\.tld$`,
			input:   "test.domain.tld",
			matched: true,
		},
		"wildcard_domain_multi_sub": {
			regex:   `(\.|^)domain\.tld$`,
			input:   "test2.test.domain.tld",
			matched: true,
		},
		"wildcard_domain_mismatch": {
			regex:   `(\.|^)domain\.tld$`,
			input:   "void.tld",
			matched: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			pub := event.NewPublisher(ctx)
			defer pub.Close()

			r, err := Match(ctx, pub, test.regex)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			match, ok := <-r.Match(ctx, test.input, time.Second)
			if ok != test.matched {
				t.Errorf("expected %v, got %v", test.matched, ok)
			}

			// Check to see if the match is supposed to fail and the
			// output matches the expected value
			if !test.matched == (len(match) == 0) {
				return
			}

			if match != test.regex {
				t.Errorf("expected %v, got %v", test.regex, match)
			}
		})
	}
}
