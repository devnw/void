package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"go.atomizer.io/stream"
)

func NewMatcher(
	ctx context.Context,
	logger Logger,
	records ...*Record,
) (*Matcher, error) {
	err := checkNil(ctx, logger)
	if err != nil {
		return nil, err
	}

	// TODO: Add future support for specific record types
	regex := []*Record{}
	directs := map[string]*Record{}
	for _, r := range records {
		switch r.Type {
		case REGEX, WILDCARD:
			regex = append(regex, r)
		case DIRECT:
			directs[r.Pattern] = r
		}
	}

	var regexMatcher *Regex
	if len(regex) > 0 {
		regexMatcher, err = Match(ctx, logger, regex...)
		if err != nil {
			return nil, err
		}
	}

	return &Matcher{
		ctx:     ctx,
		logger:  logger,
		records: directs,
		regex:   regexMatcher,
	}, nil
}

type Matcher struct {
	ctx       context.Context
	logger    Logger
	records   map[string]*Record
	recordsMu sync.RWMutex
	regex     *Regex
}

func (m *Matcher) Add(r *Record) {
	m.recordsMu.Lock()
	m.records[r.Pattern] = r
	m.recordsMu.Unlock()
}

func (m *Matcher) Match(ctx context.Context, domain string) *Record {
	if m.records == nil {
		return nil
	}

	m.recordsMu.RLock()
	r, ok := m.records[domain]
	m.recordsMu.RUnlock()

	if ok {
		return r
	}

	if m.regex != nil {
		select {
		case <-ctx.Done():
		// TODO: Add configurable timeout
		case r, ok = <-m.regex.Match(ctx, domain, time.Second):
			if ok {
				return r
			}
		}
	}

	return nil
}

// Record defines the void-specific struct for a DNS record
// indicating the type as well as category, etc...
type Record struct {
	Pattern  string
	Type     Type
	IP       net.IP
	Category string
	Tags     []string
	Source   string
	Comment  string
}

func (r *Record) String() string {
	comment := ""
	if r.Comment != "" {
		comment = fmt.Sprintf(" | comment (%s)", r.Comment)
	}

	return fmt.Sprintf(
		"src: %s | cat: %s | %s: %s | ip: %s | tags [%s]%s",
		r.Source,
		r.Category,
		r.Type,
		r.Pattern,
		r.IP,
		strings.Join(r.Tags, ","),
		comment,
	)
}

// MarshalJSON implements the json.Marshaler interface.
func (r *Record) MarshalJSON() ([]byte, error) {
	d := struct {
		Domain   string   `json:"domain"`
		Type     string   `json:"type,omitempty"`
		IP       string   `json:"ip,omitempty"`
		Category string   `json:"category,omitempty"`
		Tags     []string `json:"tags,omitempty"`
		Source   string   `json:"source,omitempty"`
		Comment  string   `json:"comment,omitempty"`
	}{
		Domain:   r.Pattern,
		Type:     r.Type.String(),
		IP:       r.IP.String(),
		Category: r.Category,
		Tags:     r.Tags,
		Source:   r.Source,
		Comment:  r.Comment,
	}

	return json.Marshal(d)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *Record) UnmarshalJSON(data []byte) error {
	d := struct {
		Domain   string   `json:"domain"`
		Type     string   `json:"type"`
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
	r.Type = Type(d.Type)
	r.IP = net.ParseIP(d.IP)
	r.Category = d.Category
	r.Tags = d.Tags
	r.Source = d.Source
	r.Comment = d.Comment

	return nil
}

// Records reads files from the provided list of directories
// and returns a slice of records.
func Records(
	ctx context.Context,
	logger Logger,
	paths ...string,
) []Record {
	var records []Record
	files := make(chan string)

	for _, path := range paths {
		go stream.Pipe(
			ctx,
			ReadDirectory(ctx, logger, path),
			files,
		)
	}

	bodies := ReadFiles(ctx, logger, files)
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
