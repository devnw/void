package main

import (
	"context"
	"regexp"
	"testing"
	"unsafe"
)

func Test_RegexFile_ReadRegex(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	regex := ReadRegex(ctx, "testdata/remote/")

	for _, reggy := range regex {
		t.Logf("%+v", reggy)
	}

	t.Logf("%d Regexes: size %d", len(regex), sizeR(regex))
}

func sizeR(records []*regexp.Regexp) uintptr {
	total := uintptr(0)
	for _, record := range records {
		total += uintptr(unsafe.Sizeof(*record))
	}

	return total
}
