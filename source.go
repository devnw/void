package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

type tee struct {
	io.Reader
	req  io.Closer
	file io.Closer
}

func (t tee) Close() error {
	var err error
	if t.req != nil {
		err = t.req.Close()
	}

	if t.file != nil {
		err = t.file.Close()
	}

	return err
}

func get(
	ctx context.Context, path string, cacheDir string,
) (io.ReadCloser, error) {
	c := http.DefaultClient

	var f *os.File
	if cacheDir != "" {
		cache := filepath.Join(cacheDir, strings.ReplaceAll(path, "/", "_"))

		var err error
		f, err = os.OpenFile(cache, os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			log.Print(err)
			f = nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		if f != nil {
			return f, nil
		}

		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		if f != nil {
			return f, nil
		}

		return nil, err
	}

	if resp.StatusCode > 299 {
		if f != nil {
			return f, nil
		}

		return nil, fmt.Errorf("%s: %s", path, resp.Status)
	}

	t := &tee{
		io.TeeReader(resp.Body, f),
		resp.Body,
		f,
	}

	return t, nil
}

func (s *Source) Records(
	ctx context.Context, cacheDir string,
) ([]*Record, error) {
	records := make([]*Record, 0)

	if strings.HasPrefix(s.Path, "http") {
		r, err := s.Remote(ctx, cacheDir)
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

func (s *Source) Remote(
	ctx context.Context, cacheDir string,
) ([]*Record, error) {
	srcs := []*Source{s}

	if s.Lists {
		body, err := get(ctx, s.Path, cacheDir)
		if err != nil {
			return nil, err
		}

		srcs = readSrcs(s, body)
	}

	records := make([]*Record, 0)

	for _, src := range srcs {
		body, err := get(ctx, src.Path, cacheDir)
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

func (s Sources) Records(ctx context.Context, cacheDir string) []*Record {
	records := make([]*Record, 0)

	for _, src := range s {
		r, err := src.Records(ctx, cacheDir)
		if err != nil {
			log.Println(err)
			continue
		}

		records = append(records, r...)
	}

	return records
}
