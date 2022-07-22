package main

import (
	"context"
	"testing"
)

func Test_HostFile_ReadHosts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := ReadHosts(ctx, "testdata/remote/")

	for _, host := range hosts {
		t.Logf("%s", host)
	}
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
