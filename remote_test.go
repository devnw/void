package main

import (
	"context"
	"testing"
)

func Test_Get(t *testing.T) {

	data, err := Get(context.Background(), "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts")
	if err != nil {
		t.Error(err)
	}

	t.Log(string(data))
}
