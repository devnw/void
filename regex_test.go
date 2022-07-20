package main

import (
	"context"
	"testing"
	"time"

	"go.devnw.com/event"
)

// Steven Black Block List: https://github.com/mmotti/pihole-regex
//go:generate curl https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts -o ./testdata/remote/stevenblack_block.hosts

// Regex Block List and False-Positive Allow List: https://github.com/mmotti/pihole-regex
//go:generate curl https://raw.githubusercontent.com/mmotti/pihole-regex/master/regex.list -o ./testdata/remote/mmotti_block.regex
//go:generate curl https://github.com/mmotti/pihole-regex/blob/master/whitelist.list -o ./testdata/remote/mmotti_allow.direct

// Firebog List of Lists: https://firebog.net/
//go:generate curl https://v.firebog.net/hosts/lists.php?type=all -o ./testdata/remote/firebog_block.lists

// Developer Dan Lists: https://github.com/lightswitch05/hosts
//go:generate curl https://www.github.developerdan.com/hosts/lists/ads-and-tracking-extended.txt -o ./testdata/remote/ddanads_block.hosts
//go:generate curl https://www.github.developerdan.com/hosts/lists/facebook-extended.txt -o ./testdata/remote/ddanfb_block.hosts
//go:generate curl https://www.github.developerdan.com/hosts/lists/amp-hosts-extended.txt -o ./testdata/remote/ddanamp_block.hosts
//go:generate curl https://www.github.developerdan.com/hosts/lists/dating-services-extended.txt -o ./testdata/remote/ddandating_block.hosts
//go:generate curl https://www.github.developerdan.com/hosts/lists/hate-and-junk-extended.txt -o ./testdata/remote/ddanhatejunk_block.hosts
//go:generaet curl https://www.github.developerdan.com/hosts/lists/tracking-aggressive-extended.txt -o ./testdata/remote/ddantracking_block.hosts

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

func Benchmark_Match(b *testing.B) {
	regex := `(\.|^)domain\.tld$`
	input := "domain.tld"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pub := event.NewPublisher(ctx)
	defer pub.Close()

	r, err := Match(ctx, pub, regex)
	if err != nil {
		b.Errorf("unexpected error: %v", err)
	}

	for n := 0; n < b.N; n++ {
		match, ok := <-r.Match(ctx, input, time.Second)
		if !ok {
			b.Fatalf("expected match")
		}

		if match != regex {
			b.Fatalf("expected %v, got %v", regex, match)
		}
	}
}

func Test_Wildcard(t *testing.T) {
	tests := map[string]struct {
		wildcard string
		expected string
		input    string
		match    bool
		err      error
	}{
		"valid": {
			wildcard: "*domain.tld",
			expected: `(\.|^)domain\.tld$`,
			input:    "test.domain.tld",
			match:    true,
			err:      nil,
		},
		"invalid": {
			wildcard: "d*domain.tld",
			err:      ErrWildcard,
		},
		"invalid_domain": {
			wildcard: "*.tld",
			err:      ErrDomain,
		},
		"invalid_tld": {
			wildcard: "*domain.",
			err:      ErrTLD,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			r, err := Wildcard(test.wildcard)
			if err != test.err {
				t.Errorf("expected %v, got %v", test.err, err)
			}

			if err != nil {
				return
			}

			if r.String() != test.expected {
				t.Errorf("expected [%v], got [%v]", test.expected, r.String())
			}

			match := r.MatchString(test.input)
			if match != test.match {
				t.Errorf("expected %v, got %v", test.match, match)
			}
		})
	}
}
