package main

import (
	"context"
	"testing"
)

func Test_Get(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	in := make(chan []byte)
	out, err := Extract(context.Background(), in)
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
		case line, ok := <-out:
			if !ok {
				return
			}

			t.Log(line)
		}
	}
}
