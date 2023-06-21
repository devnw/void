package main

import (
	"context"
	"testing"

	"golang.org/x/exp/slog"
)

func Test_RegexFile_ReadRegex(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	regex := ReadRegex(ctx, slog.Default(), "testdata/remote/")

	for _, reggy := range regex {
		t.Logf("%+v", reggy)
	}

	t.Logf("%d Regexes: size %d", len(regex), size(regex))
}
