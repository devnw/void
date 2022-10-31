package main

import (
	"context"
	"regexp"
	"testing"
	"unsafe"

	"go.devnw.com/event"
)

func Test_RegexFile_ReadRegex(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pub := event.NewPublisher(ctx)

	regex := ReadRegex(ctx, pub, "testdata/remote/")

	for _, reggy := range regex {
		t.Logf("%+v", reggy)
	}

	t.Logf("%d Regexes: size %d", len(regex), sizeR(regex))
}

func sizeR(records []*regexp.Regexp) uintptr {
	total := uintptr(0)
	for _, record := range records {
		total += unsafe.Sizeof(*record)
	}

	return total
}
