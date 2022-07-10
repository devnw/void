package main

import (
	"context"
	"testing"
)

func Test_Get(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	in := make(chan []byte)
	lines, err := Extract(context.Background(), in)
	if err != nil {
		t.Error(err)
	}

	hosts, err := GetHost(context.Background(), lines)
	if err != nil {
		t.Error(err)
	}

	go func() {
		data, err := Get(context.Background(), "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts")
		if err != nil {
			t.Error(err)
		}

		in <- data
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case host, ok := <-hosts:
			if !ok {
				return
			}

			t.Logf("%+v\n", host)
		}
	}
}
