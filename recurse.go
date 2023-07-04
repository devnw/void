package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"math/rand"
	"net"
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
		client: &dns.Client{
			Net:     "udp",
			Timeout: time.Second * 5,
		},
	}

	return r.Intercept, nil
}

type recursive struct {
	zones  <-chan *RootRecord
	cache  *ttl.Cache[string, *dns.Msg]
	client *dns.Client
}

func (r *recursive) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	return nil, false
}

func (r *recursive) resolve(
	ctx context.Context,
	question string,
) (*dns.Msg, error) {
	msg := &dns.Msg{
		Question: []dns.Question{
			{
				Name:   question,
				Qtype:  dns.TypeNS,
				Qclass: dns.ClassINET,
			},
		},
	}

	if question == "" {
		return rootZ, nil

		//fmt.Println("resolving", msg.Question[0].Name)

		//var zone *RootRecord
		//for {
		//	var ok bool
		//	zone, ok = <-r.zones
		//	if !ok {
		//		return nil, fmt.Errorf("no zones left")
		//	}

		//	fmt.Printf("class: [%s] | value [%v]\n", zone.Class, zone.Value)

		//	if zone.Class == "A" {
		//		break
		//	}
		//}

		//msg.Question[0].Name = "com."

		//resp, _, err := r.client.Exchange(
		//	msg, net.JoinHostPort(zone.Value, "53"),
		//)
		//spew.Config.DisableMethods = true
		//spew.Dump(resp, err)
		//fmt.Printf("response %+v, %+v, %v", resp, zone, err)
		//if err != nil {
		//	return nil, err
		//}

		//return resp, nil
	}

	fmt.Println("recursing", msg.Question[0].Name)

	i := strings.Index(question, ".")
	if i > 0 {
		question = question[i+1:]
	}

	resp, err := r.resolve(ctx, question)
	if err != nil {
		return nil, err
	}

	fmt.Println("resolving", msg.Question[0].Name)

	if resp.Authoritative {
		var ns string
		for _, rr := range resp.Extra {
			if rr.Header().Rrtype == dns.TypeA {
				ns = rr.(*dns.A).A.String()
				break
			}

			if rr.Header().Rrtype == dns.TypeAAAA {
				ns = rr.(*dns.AAAA).AAAA.String()
				break
			}
		}

		fmt.Printf("ns: %s\n", ns)

		if ns == "" {
			return nil, fmt.Errorf("no NS record found")
		}

		msg = &dns.Msg{
			Question: []dns.Question{
				{
					Name:   msg.Question[0].Name,
					Qtype:  dns.TypeA,
					Qclass: dns.ClassINET,
				},
			},
		}

		resp, _, err := r.client.Exchange(
			msg, net.JoinHostPort(ns, "53"),
		)
		if err != nil {
			return nil, err
		}

		return resp, nil
	}

	var ns string
	for _, rr := range resp.Extra {
		if rr.Header().Rrtype == dns.TypeA {
			ns = rr.(*dns.A).A.String()
			break
		}
	}

	next, _, err := r.client.Exchange(
		msg, net.JoinHostPort(ns, "53"),
	)
	if err != nil {
		return nil, err
	}

	return next, nil

}

var rootZ *dns.Msg = &dns.Msg{}

type RootZone []RootRecord

type RootRecord struct {
	Name  string
	TTL   time.Duration
	Class string
	Value string
}

func (r *RootZone) Records(ctx context.Context) <-chan *RootRecord {
	out := make(chan *RootRecord)

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

func (r RootZone) next() *RootRecord {
	// Get a random number on the index of the records
	index := rand.Intn(len(r))

	// Get the record at that index
	return &r[index]
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

		if fields[2] != "NS" {
			rootZ.Answer = append(rootZ.Answer, &dns.NS{
				Hdr: dns.RR_Header{
					Name:   fields[0],
					Rrtype: dns.TypeNS,
					Class:  dns.ClassINET,
					Ttl:    uint32(ttl),
				},
			})
		}

		if fields[2] == "A" {
			rootZ.Answer = append(rootZ.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   fields[0],
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    uint32(ttl),
				},
				A: net.ParseIP(fields[3]),
			})
		}

		if fields[2] == "AAAA" {
			rootZ.Answer = append(rootZ.Answer, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   fields[0],
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    uint32(ttl),
				},
				AAAA: net.ParseIP(fields[3]),
			})
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
