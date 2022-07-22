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

// Host defines the structure of a host record
// from a hosts file similar to /etc/hosts
// Example: 0.0.0.0 example.com # comment
//
// https://www.ibm.com/docs/en/aix/7.2?topic=formats-hosts-file-format-tcpip
//
// NOTE: According to the link above multiple domains are allowed per IP
// as long as they're on the same line, space separated.
//
// TODO: Determine if this is something to support (i.e. local dns resolution)
type Host struct {
	Domain  string `json:"domain"`
	IP      net.IP `json:"ip"`
	Comment string `json:"comment"`
}

// Record converts a host record to a void domain record
func (h *Host) Record(src, cat string, tags ...string) *Record {
	return &Record{
		Domain:   h.Domain,
		IP:       h.IP,
		Comment:  h.Comment,
		Eval:     DIRECT,
		Source:   src,
		Category: cat,
		Tags:     tags,
	}
}

// Hosts is a slice of Hosts
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

const columns = 2

// ReadHosts reads host files from the provided directories
func ReadHosts(ctx context.Context, path string) []*Host {
	var hosts []*Host

	count := 0
	bodies := ReadFiles(ctx, ReadDirectory(ctx, path))
	for {
		select {
		case <-ctx.Done():
			return hosts
		case body, ok := <-bodies:
			if !ok {
				return hosts
			}

			hosts = append(hosts, parseFile(ctx, body)...)
			count++
			fmt.Printf("Processed %d files\n", count)
		}
	}
}

func parseFile(ctx context.Context, body io.ReadCloser) []*Host {
	hosts := make([]*Host, 0)
	defer func() {
		r := recover()
		if r != nil {
			fmt.Printf("error parsing host file; %s\n", r)
		}
	}()

	data, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return nil
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var comment string
		commentIndex := strings.Index(line, "#")
		if commentIndex != -1 {
			// Pull the comment for the line for
			// later use.
			comment = strings.TrimSpace(
				line[commentIndex+1:],
			)

			// Trim the comment from the line.
			line = strings.TrimSpace(
				line[:commentIndex],
			)
		}

		first := strings.Index(line, " ")
		if first == -1 {
			continue
		}

		// Pull the IP
		ip := net.ParseIP(line[:first])
		if ip == nil {
			continue
		}

		tail := len(line) - 1
		if commentIndex != -1 {
			tail = commentIndex - 1
		}

		names := strings.Split(
			strings.TrimSpace(
				line[first+1:tail],
			), " ")
		if len(names) < 1 {
			continue
		}

		for _, name := range names {
			hosts = append(hosts, &Host{
				IP:      ip,
				Domain:  name,
				Comment: comment,
			})
		}
	}

	return hosts
}

// Extract pulls the record lines from host files and sends them
// to the given channel.
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
// nolint:gocritic
func extract(out chan<- string) stream.InterceptFunc[[]byte, struct{}] {
	return func(ctx context.Context, body []byte) (struct{}, bool) {
		var nextline string
		for _, b := range body {
			if b == '\n' {
				// Trim any spaces
				line := strings.TrimSpace(nextline)
				nextline = "" // Reset the buffer on newlines

				// Ignore empty or commented lines
				if line == "" ||
					strings.HasPrefix(line, "#") {
					continue
				}

				select {
				case <-ctx.Done():
					return struct{}{}, false
				case out <- line:
				}

				continue
			}
			nextline += string(b)
		}

		return struct{}{}, false
	}
}

// GetHost parses the line of host file returning the first host
//
// TODO: Add support for multiple domains on a single IP as indicated
// at the link below:
// https://www.ibm.com/docs/en/aix/7.2?topic=formats-hosts-file-format-tcpip
func GetHost(ctx context.Context, in <-chan string) (<-chan *Host, error) {
	s := stream.Scaler[string, *Host]{
		Wait: time.Nanosecond,
		Life: time.Millisecond,
		Fn: func(ctx context.Context, line string) (*Host, bool) {
			var comment string
			commentIndex := strings.Index(line, "#")
			if commentIndex != -1 {
				// Pull the comment for the line for
				// later use.
				comment = strings.TrimSpace(
					line[commentIndex+1:],
				)

				// Trim the comment from the line.
				line = strings.TrimSpace(
					line[:commentIndex],
				)
			}

			cols := strings.Split(line, " ")
			if len(cols) != columns {
				return nil, false
			}

			return &Host{
				IP:      net.ParseIP(cols[0]),
				Domain:  cols[1],
				Comment: comment,
			}, true
		},
	}

	out, err := s.Exec(ctx, in)
	if err != nil {
		return nil, err
	}

	return out, nil
}
