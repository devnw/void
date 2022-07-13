package main

import (
	"context"
	"testing"
)

func Test_Up(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := Up(ctx, "tcp://1.1.1.1:53")
	if err != nil {
		t.Error(err)
	}
}
