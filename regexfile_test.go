package main

import (
	"context"
	"testing"
)

func Test_RegexFile_ReadRegex(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := &NOOPLogger{}

	regex := ReadRegex(ctx, logger, "testdata/remote/")

	for _, reggy := range regex {
		t.Logf("%+v", reggy)
	}

	t.Logf("%d Regexes: size %d", len(regex), size(regex))
}
