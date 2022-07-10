package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	stream "go.atomizer.io/stream"
)

type Host struct {
	Domain string `json:"domain"`
	IP     net.IP `json:"ip"`
}

func Hosts(ctx context.Context, paths ...string) []Host {
	var hosts []Host
	files := make(chan string)

	for _, path := range paths {
		go stream.Pipe(
			ctx,
			ReadDirectory(ctx, path),
			files,
		)
	}

	bodies := ReadFiles(ctx, files)
	for body := range bodies {
		data, err := io.ReadAll(body)
		body.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}

		lines := strings.Split(string(data), "\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			cols := strings.Split(string(data), " ")
			if len(cols) != 2 {
				continue
			}

			hosts = append(hosts, Host{
				IP:     net.ParseIP(cols[0]),
				Domain: cols[1],
			})
		}
	}

	return hosts
}
