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
type Type uint8

const (
	// DIRECT indicates a direct DNS record, compared 1 to 1
	DIRECT = iota

	// WILDCARD indicates a wildcard DNS record, (e.g. *.google.com)
	// which will be converted to the appropriate regex or matched with
	// HasSuffix check
	WILDCARD

	// REGEX indicates a regular expression to match DNS requests
	// against for blocking many records with a single filter
	REGEX
)

var evalTypeStrings = gen.FMap[Type, string]{
	DIRECT:   "direct",
	WILDCARD: "wildcard",
	REGEX:    "regex",
}

var evalTypeStringsR = evalTypeStrings.Flip()

// EvalStringToType returns the type of a record based on the string
// representation of the type
func EvalStringToType(str string) Type {
	return evalTypeStringsR[str]
}

func (t Type) String() string {
	return evalTypeStrings[t]
}
