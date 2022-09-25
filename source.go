package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Source struct {
	Path     string
	Lists    bool
	Format   Type
	Sync     *time.Duration
	Category string
	Tags     []string
}

func get(ctx context.Context, path string) (io.ReadCloser, error) {
	c := http.DefaultClient

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("%s: %s", path, resp.Status)
	}

	return resp.Body, nil
}

func (s *Source) Records(ctx context.Context) ([]*Record, error) {
	records := make([]*Record, 0)

	if strings.HasPrefix(s.Path, "http") {
		r, err := s.Remote(ctx)
		if err != nil {
			return nil, err
		}

		records = append(records, r...)
		return records, nil
	}

	r, err := s.Local(ctx)
	if err != nil {
		return nil, err
	}

	records = append(records, r...)
	return records, nil
}

func (s *Source) Local(ctx context.Context) ([]*Record, error) {
	srcs := []*Source{s}

	if s.Lists {
		srcs = []*Source{}

		f, err := os.Open(s.Path)
		if err != nil {
			log.Fatal(err)
		}

		srcs = readSrcs(s, f)
	}

	records := make([]*Record, 0)
	for _, src := range srcs {
		f, err := os.Open(src.Path)
		if err != nil {
			log.Fatal(err)
		}

		defer f.Close()
		entries := Parse(ctx, src.Format, f).Records(
			src.Path,
			src.Category,
			src.Tags...,
		)

		fmt.Printf("loaded %d entries from %s\n", len(entries), src.Path)
		records = append(records, entries...)
	}

	return records, nil
}

func (s *Source) Remote(ctx context.Context) ([]*Record, error) {
	srcs := []*Source{s}

	if s.Lists {
		srcs = []*Source{}

		body, err := get(ctx, s.Path)
		if err != nil {
			return nil, err
		}

		srcs = readSrcs(s, body)
	}

	records := make([]*Record, 0)

	for _, src := range srcs {
		body, err := get(ctx, src.Path)
		if err != nil {
			fmt.Println(err)
			continue
		}

		entries := Parse(ctx, src.Format, body).Records(
			src.Path,
			src.Category,
			src.Tags...,
		)

		fmt.Printf("loaded %d entries from %s\n", len(entries), src.Path)
		records = append(records, entries...)
	}

	return records, nil
}

func readSrcs(parent *Source, body io.ReadCloser) []*Source {
	srcs := make([]*Source, 0)

	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return srcs
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		srcs = append(srcs, &Source{
			Path:     line,
			Format:   parent.Format,
			Sync:     parent.Sync,
			Category: parent.Category,
			Tags:     parent.Tags,
		})
	}

	return srcs
}

type Sources []Source

func (s Sources) Records(ctx context.Context) []*Record {
	records := make([]*Record, 0)

	for _, src := range s {
		r, err := src.Records(ctx)
		if err != nil {
			log.Println(err)
			continue
		}

		records = append(records, r...)
	}

	return records
}
