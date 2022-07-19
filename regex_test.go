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

			if match != test.regex {
				t.Errorf("expected %v, got %v", test.regex, match)
			}
		})
	}
}
