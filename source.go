package main

import (
	"context"
	"log"
	"os"
	"time"
)

type LocType string

const (
	LOC LocType = "local"
	REM LocType = "remote"
)

type Source struct {
	Path     string
	Type     LocType
	Format   Type
	Sync     *time.Duration
	Category string
	Tags     []string
}

type Sources []Source

func (s Sources) Records(ctx context.Context) []*Record {
	records := make([]*Record, 0)

	for _, src := range s {
		switch src.Type {
		case LOC:
			f, err := os.Open(src.Path)
			if err != nil {
				log.Fatal(err)
			}

			defer f.Close()
			records = append(
				records,
				Parse(ctx, src.Format, f).Records(
					src.Path,
					src.Category,
					src.Tags...,
				)...,
			)
		case REM:
		}

	}

	return records
}
