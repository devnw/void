package main

import (
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func Test_ParseZone(t *testing.T) {
	zone, err := os.Open("named.root")
	if err != nil {
		t.Fatal(err)
	}

	msg := ParseZone(zone)

	spew.Dump(msg)
}
