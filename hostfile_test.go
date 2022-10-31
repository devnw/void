package main

import (
	"context"
	"testing"
	"unsafe"

	"go.devnw.com/event"
)

func Test_HostFile_ReadHosts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pub := event.NewPublisher(ctx)

	hosts := ReadHosts(ctx, pub, DIRECT, "testdata/direct/")

	records := hosts.Records("remote", "block", "pihole")
	for _, host := range records {
		t.Logf("%+v", host)
	}

	t.Logf("%d Hosts: size %d", len(records), size(records))
}

func size(records []*Record) uintptr {
	total := uintptr(0)
	for _, record := range records {
		total += unsafe.Sizeof(*record)
	}

	return total
}

func Test_HostFile_GetHost(t *testing.T) {
}

func Test_HostFile_extract(t *testing.T) {
}

func Test_Host_Record(t *testing.T) {
}

func Test_Host_Len(t *testing.T) {
}

func Test_Host_Less(t *testing.T) {
}

func Test_Host_Swap(t *testing.T) {
}

func Test_Host_sort(t *testing.T) {
}
