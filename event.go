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
	Server   string   `json:"server,omitempty"`
	Record   *Record  `json:"record"`
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
		srv = fmt.Sprintf(" | server: %s", e.Server)
	}

	src := ""
	if e.Source != "" {
		src = fmt.Sprintf(" | source: %s", e.Source)
	}
	return fmt.Sprintf(
		"%sname: %s | type: %s%s; %s: %s, tags: [%s]%s",
		msg,
		e.Name,
		e.Type,
		srv,
		e.Record.Type,
		e.Record.Pattern,
		strings.Join(e.Record.Tags, ","),
		src,
	)
}

func (e *Event) Event() string {
	return e.String()
}
