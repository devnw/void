package main

import (
	"encoding/json"
	"net"

	. "go.structs.dev/gen"
)

type Config struct {
	Port  int                 `json:"listen_port"`
	TTL   int                 `json:"ttl"`
	Local Map[string, Record] `json:"local_records"`
	Allow Map[string, Record] `json:"allow_records"`
	Deny  Map[string, Record] `json:"deny_records"`
}

type Type uint8

const (
	DIRECT = iota
	WILDCARD
	REGEX
)

var typeStrings = FMap[Type, string]{
	DIRECT:   "direct",
	WILDCARD: "wildcard",
	REGEX:    "regex",
}

var typeStringsR = typeStrings.Flip()

func StringToType(str string) Type {
	return typeStringsR[str]
}

func (t Type) String() string {
	return typeStrings[t]
}

type Record struct {
	Domain   string   `json:"domain"`
	Type     Type     `json:"type"`
	IP       net.IP   `json:"ip"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
	Source   string   `json:"source"`
}

func (r *Record) MarshalJSON() ([]byte, error) {
	d := struct {
		Domain   string   `json:"domain"`
		IP       string   `json:"ip"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
		Source   string   `json:"source"`
	}{
		Domain:   r.Domain,
		IP:       r.IP.String(),
		Category: r.Category,
		Tags:     r.Tags,
		Source:   r.Source,
	}

	return json.Marshal(d)
}

func (r *Record) UnmarshalJSON(data []byte) error {
	d := struct {
		Domain   string   `json:"domain"`
		IP       string   `json:"ip"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
		Source   string   `json:"source"`
	}{}

	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}

	r.Domain = d.Domain
	r.IP = net.ParseIP(d.IP)
	r.Category = d.Category
	r.Tags = d.Tags
	r.Source = d.Source

	return nil
}
