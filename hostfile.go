package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	stream "go.atomizer.io/stream"
)

type Host struct {
	Domain string `json:"domain"`
	IP     net.IP `json:"ip"`
}

type Hosts []Host

func (h Hosts) Len() int {
	return len(h)
}

func (h Hosts) Less(i, j int) bool {
	return h[i].Domain < h[j].Domain
}

func (h Hosts) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func ReadHosts(ctx context.Context, paths ...string) []Host {
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

func Extract(ctx context.Context, in <-chan []byte) (<-chan string, error) {
	out := make(chan string)

	s := stream.Scaler[[]byte, struct{}]{
		Wait: time.Nanosecond,
		Life: time.Millisecond,
		Fn:   extract(out),
	}

	_, err := s.Exec(ctx, in)
	if err != nil {
		panic(err)
	}

	return out, nil
}

// extract returns an intercept function which bypasses the direct
// output of the stream and instead sends the output to the given
// channel so that it can fan-out to other streams.
func extract(out chan<- string) stream.InterceptFunc[[]byte, struct{}] {
	return func(ctx context.Context, body []byte) (struct{}, bool) {
		var line string
		for _, b := range body {
			if b == '\n' {

				// Trim any spaces
				line = strings.TrimSpace(line)

				// Ignore empty or commented lines
				if line == "" ||
					strings.HasPrefix(line, "#") {
					continue
				}

				select {
				case <-ctx.Done():
					return struct{}{}, false
				case out <- line:
					continue
				}

				continue
			}
			line += string(b)
		}

		return struct{}{}, false
	}
}
