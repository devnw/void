package main

import (
	"context"
	"testing"
	"unsafe"
)

func Test_HostFile_ReadHosts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := ReadHosts(ctx, logger, DIRECT, "testdata/direct/")

	records := hosts.Records("remote", "block", "pihole")
	for _, host := range records {
		t.Logf("%+v", host)
	}

	t.Logf("%d Hosts: size %d", len(records), size(records))
}

func size[T any](records []*T) uintptr {
	total := uintptr(0)
	for _, record := range records {
		total += unsafe.Sizeof(*record)
	}

	return total
}
