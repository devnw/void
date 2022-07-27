package main

import (
	gen "go.structs.dev/gen"
)

// Config defines the configuration options available for void
type Config struct {
	Port  int                     `json:"listen_port"`
	TTL   int                     `json:"ttl"`
	Local gen.Map[string, Record] `json:"local_records"`
	Allow gen.Map[string, Record] `json:"allow_records"`
	Deny  gen.Map[string, Record] `json:"deny_records"`
}

// Type indicates the type of a record to ensure proper analysis
type Type string

const (
	// DIRECT indicates a direct DNS record, compared 1 to 1
	DIRECT Type = "direct"

	// WILDCARD indicates a wildcard DNS record, (e.g. *.google.com)
	// which will be converted to the appropriate regex or matched with
	// HasSuffix check
	WILDCARD Type = "wildcard"

	// REGEX indicates a regular expression to match DNS requests
	// against for blocking many records with a single filter
	REGEX Type = "regex"

	VOID Type = "void"
)

func (t Type) String() string {
	return string(t)
}
