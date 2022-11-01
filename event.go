package main

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

type Event struct {
	Msg      string   `json:"msg"`
	Name     string   `json:"name"`
	Type     dns.Type `json:"type"`
	Client   string   `json:"client,omitempty"`
	Server   string   `json:"server,omitempty"`
	Record   *Record  `json:"record,omitempty"`
	Category Category `json:"category"`
	Source   string   `json:"source"`
}

func (e *Event) String() string {
	msg := ""
	if e.Msg != "" {
		msg = fmt.Sprintf("%s: ", e.Msg)
	}

	srv := ""
	if e.Server != "" {
		srv = fmt.Sprintf(" | server: %s;", e.Server)
	}

	client := ""
	if e.Client != "" {
		client = fmt.Sprintf(" | client: %s;", e.Client)
	}

	src := ""
	if e.Source != "" {
		src = fmt.Sprintf(" | source: %s;", e.Source)
	}

	rec := ""
	if e.Record != nil {
		rec = fmt.Sprintf(
			" | record type: %s: %s, tags: [%s]",
			e.Record.Type,
			e.Record.Pattern,
			strings.Join(e.Record.Tags, ", "),
		)
	}

	return fmt.Sprintf(
		"%s name: %s | type: %s%s%s%s%s",
		msg,
		e.Name,
		e.Type,
		srv,
		client,
		rec,
		src,
	)
}

func (e *Event) Event() string {
	return e.String()
}
