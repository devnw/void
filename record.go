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
	Pattern  string
	Type     Type
	Eval     EvalType
	IP       net.IP
	Category string
	Tags     []string
	Source   string
	Comment  string
}

// TODO: Add String()

// MarshalJSON implements the json.Marshaler interface
func (r *Record) MarshalJSON() ([]byte, error) {
	d := struct {
		Domain   string   `json:"domain"`
		Type     string   `json:"type,omitempty"`
		Eval     string   `json:"evalType"`
		IP       string   `json:"ip,omitempty"`
		Category string   `json:"category,omitempty"`
		Tags     []string `json:"tags,omitempty"`
		Source   string   `json:"source,omitempty"`
		Comment  string   `json:"comment,omitempty"`
	}{
		Domain:   r.Pattern,
		Type:     r.Type.String(),
		Eval:     r.Eval.String(),
		IP:       r.IP.String(),
		Category: r.Category,
		Tags:     r.Tags,
		Source:   r.Source,
		Comment:  r.Comment,
	}

	return json.Marshal(d)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (r *Record) UnmarshalJSON(data []byte) error {
	d := struct {
		Domain   string   `json:"domain"`
		Type     string   `json:"type"`
		Eval     string   `json:"evalType"`
		IP       string   `json:"ip"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
		Source   string   `json:"source"`
		Comment  string   `json:"comment"`
	}{}

	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}

	r.Pattern = d.Domain
	r.Type = Type(StringToType[d.Type])
	r.Eval = EvalStringToType(d.Eval)
	r.IP = net.ParseIP(d.IP)
	r.Category = d.Category
	r.Tags = d.Tags
	r.Source = d.Source
	r.Comment = d.Comment

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
