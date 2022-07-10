package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"

	stream "go.atomizer.io/stream"
)

// Record defines the void-specific struct for a DNS record
// indicating the type as well as category, etc...
type Record struct {
	Domain   string   `json:"domain"`
	Type     Type     `json:"type"`
	IP       net.IP   `json:"ip"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
	Source   string   `json:"source"`
	Comment  string   `json:"comment"`
}

// MarshalJSON implements the json.Marshaler interface
func (r *Record) MarshalJSON() ([]byte, error) {
	d := struct {
		Domain   string   `json:"domain"`
		IP       string   `json:"ip"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
		Source   string   `json:"source"`
	}{
		Domain:   r.Domain,
		IP:       r.IP.String(),
		Category: r.Category,
		Tags:     r.Tags,
		Source:   r.Source,
	}

	return json.Marshal(d)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (r *Record) UnmarshalJSON(data []byte) error {
	d := struct {
		Domain   string   `json:"domain"`
		IP       string   `json:"ip"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
		Source   string   `json:"source"`
	}{}

	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}

	r.Domain = d.Domain
	r.IP = net.ParseIP(d.IP)
	r.Category = d.Category
	r.Tags = d.Tags
	r.Source = d.Source

	return nil
}

// Records reads files from the provided list of directories
// and returns a slice of records
func Records(ctx context.Context, paths ...string) []Record {
	var records []Record
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

		var record Record
		if err := json.Unmarshal(data, &record); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			continue
		}
		records = append(records, record)
	}

	return records
}
