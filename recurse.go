package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
	"go.atomizer.io/stream"
	"go.devnw.com/ttl"
	"golang.org/x/exp/slog"
)

//go:generate wget -O named.root https://www.internic.net/domain/named.root

//go:embed named.root
var namedRoot []byte

func Recursive(
	ctx context.Context,
	zonefile string,
) (stream.InterceptFunc[*Request, *Request], error) {
	zone, err := loadZoneFile(slog.Default(), zonefile)
	if err != nil {
		return nil, err
	}

	r := &recursive{
		zones: zone.Records(ctx),
		cache: ttl.NewCache[string, *dns.Msg](
			ctx,
			time.Second*time.Duration(DEFAULTTTL),
			false,
		),
	}

	return r.Intercept, nil
}

type recursive struct {
	zones <-chan RootRecord
	cache *ttl.Cache[string, *dns.Msg]
}

func (r *recursive) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	return nil, false
}

type RootZone []RootRecord

type RootRecord struct {
	Name  string
	TTL   time.Duration
	Class string
	Value string
}

func (r *RootZone) Records(ctx context.Context) <-chan RootRecord {
	out := make(chan RootRecord)

	go func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case out <- r.next():
			}
		}
	}()

	return out
}

func (r RootZone) next() RootRecord {
	// Get a random number on the index of the records
	index := rand.Intn(len(r))

	// Get the record at that index
	return r[index]
}

func loadZoneFile(logger *slog.Logger, filepath string) (RootZone, error) {
	var r io.Reader = bytes.NewReader(namedRoot)

	if filepath != "" {
		logger.Info("loading zone file", slog.String("path", filepath))

		file, err := os.Open(filepath)
		if err != nil {
			slog.Error("failed to open zone file", slog.String("error", err.Error()))
			return nil, err
		}
		defer slog.Debug("zone file loaded", slog.String("path", filepath))
		defer file.Close()

		r = file
	}

	zone := RootZone{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		slog.Debug("scanning line", slog.String("line", scanner.Text()))
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, ";") {
			slog.Debug("skipping line", slog.String("line", line))
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 4 {
			slog.Warn("invalid line", slog.String("line", line))
			continue
		}

		ttl, err := strconv.Atoi(fields[1])
		if err != nil {
			slog.Warn("invalid ttl", slog.String("line", line))
			continue
		}

		record := RootRecord{
			Name:  fields[0],
			TTL:   time.Duration(ttl) * time.Second, // Convert to time.Duration
			Class: fields[2],
			Value: fields[3],
		}

		slog.Debug("adding record", slog.String("record", fmt.Sprintf("%+v", record)))

		zone = append(zone, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return zone, nil
}
