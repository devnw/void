package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
	"go.devnw.com/ttl"
)

//go:generate wget -O named.root https://www.internic.net/domain/named.root

//go:embed named.root
var namedRoot []byte

type Recursive struct {
	cache ttl.Cache[string, *dns.Msg]
}

func (r *Recursive) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	return nil, false
}

type RootZone struct {
	Records []RootRecord
}

type RootRecord struct {
	Name  string
	TTL   time.Duration
	Class string
	Type  string
	Data  string
}

func loadZoneFile(filepath string) (*RootZone, error) {
	var r io.Reader = bytes.NewReader(namedRoot)

	if filepath != "" {
		file, err := os.Open(filepath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		r = file
	}

	zone := &RootZone{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		ttl, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("invalid TTL: %v", err)
		}

		record := RootRecord{
			Name:  fields[0],
			TTL:   time.Duration(ttl) * time.Second, // Convert to time.Duration
			Class: fields[2],
			Type:  fields[3],
			Data:  fields[4],
		}

		zone.Records = append(zone.Records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return zone, nil
}
