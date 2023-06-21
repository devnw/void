package main

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/exp/slog"
)

func Test_loadZoneFile(t *testing.T) {
	zone, err := loadZoneFile(slog.Default(), "")
	if err != nil {
		t.Fatal(err)
	}

	spew.Dump(zone)
}
