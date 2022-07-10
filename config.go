package main

import (
	gen "go.structs.dev/gen"
)

type Config struct {
	Port  int                     `json:"listen_port"`
	TTL   int                     `json:"ttl"`
	Local gen.Map[string, Record] `json:"local_records"`
	Allow gen.Map[string, Record] `json:"allow_records"`
	Deny  gen.Map[string, Record] `json:"deny_records"`
}

type Type uint8

const (
	DIRECT = iota
	WILDCARD
	REGEX
)

var typeStrings = gen.FMap[Type, string]{
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
