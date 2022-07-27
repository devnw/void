package main

import (
	"context"
	"log"
	"os"
	"time"
)

type Format string

const (
	HOSTS Format = "hosts"
	REG   Format = "regex"
	WILD  Format = "wild"
	VOID  Format = "void"
	DIR   Format = "direct"
	LIST  Format = "list"
)

type LocType string

const (
	LOC LocType = "local"
	REM LocType = "remote"
)

type Source struct {
	Path     string
	Type     LocType
	Format   Format
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
				parseFile(ctx, f).Records(
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
