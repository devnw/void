package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
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
		root: zone.Msg(),
		cache: ttl.NewCache[string, *dns.Msg](
			ctx,
			time.Second*time.Duration(DEFAULTTTL),
			false,
		),
		client: &dns.Client{
			Net:     "udp",
			Timeout: time.Second * 5,
		},
	}

	return r.Intercept, nil
}

type recursive struct {
	root   *dns.Msg
	cache  *ttl.Cache[string, *dns.Msg]
	client *dns.Client
}

func (r *recursive) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	return nil, false
}

func (r *recursive) authoritative(
	ctx context.Context,
	q *dns.Msg,
) (*dns.Msg, error) {
	msg := &dns.Msg{
		Question: []dns.Question{
			{
				Name:   q.Question[0].Name,
				Qtype:  dns.TypeNS,
				Qclass: dns.ClassINET,
			},
		},
	}

	if q.Question[0].Name == "" {
		return r.root, nil
	}

	fmt.Println("recursing", msg.Question[0].Name)

	i := strings.Index(q.Question[0].Name, ".")
	if i > 0 {
		msg.Question[0].Name = q.Question[0].Name[i+1:]
	}

	resp, err := r.authoritative(ctx, msg)
	if err != nil {
		return nil, err
	}

	fmt.Println("resolving", msg.Question[0].Name)

	if resp.Authoritative {
		fmt.Println("++++++++++++++++++++++ AUTH")
		return resp, nil
	}

	spew.Dump(resp)

	var ns string
	for _, rr := range resp.Extra {
		if rr.Header().Rrtype == dns.TypeA {
			ns = rr.(*dns.A).A.String()
			break
		}
	}

	fmt.Printf("ns: %s\n", ns)

	next, _, err := r.client.Exchange(
		q, net.JoinHostPort(ns, "53"),
	)
	if err != nil {
		return nil, err
	}

	k := key(next)
	if k == "" {
		return nil, fmt.Errorf("unable to calculate cache key")
	}

	ttl := time.Second * DEFAULTTTL

	if len(next.Extra) > 0 && next.Extra[0].Header() != nil {
		ttl = time.Second * time.Duration(next.Extra[0].Header().Ttl)
	}

	err = r.cache.SetTTL(ctx, k, next, ttl)
	if err != nil {
		return nil, err
	}

	return next, nil

}

type RootZone []RootRecord

type RootRecord struct {
	Name  string
	TTL   time.Duration
	Class string
	Value string
}

func (r RootZone) Msg() *dns.Msg {
	msg := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: false,
		},
		Question: []dns.Question{
			{
				Name:   ".",
				Qtype:  dns.TypeNS,
				Qclass: dns.ClassINET,
			},
		},
	}

	for _, rr := range r {
		if rr.Class == "AAAA" {
			msg.Extra = append(msg.Extra, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   rr.Name,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    uint32(rr.TTL.Seconds()),
				},
				AAAA: net.ParseIP(rr.Value),
			})

			continue
		}

		if rr.Class == "A" {
			msg.Extra = append(msg.Extra, &dns.A{
				Hdr: dns.RR_Header{
					Name:   rr.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    uint32(rr.TTL.Seconds()),
				},
				A: net.ParseIP(rr.Value),
			})
		}
	}

	return msg
}

func loadZoneFile(logger *slog.Logger, filepath string) (RootZone, error) {
	var r io.Reader = bytes.NewReader(namedRoot)

	if filepath != "" {
		logger.Info("loading zone file", slog.String("path", filepath))

		file, err := os.Open(filepath)
		if err != nil {
			slog.Error(
				"failed to open zone file", slog.String("error", err.Error()),
			)
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

		slog.Debug(
			"adding record",
			slog.String("record", fmt.Sprintf("%+v", record)),
		)

		zone = append(zone, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return zone, nil
}
