package main

import (
	"github.com/miekg/dns"
	"go.structs.dev/gen"
)

// Type is a wrapper over uint16 to simplify the mapping to
// the dns.Type type in the dns package.
type Type uint16

func (t Type) String() string {
	return dns.Type(t).String()
}

var typeToString = gen.FMap[uint16, string](dns.TypeToString)

// StringToType is a map which contains the string as the key
// value of the map so that unmarshaling can easily lookup the
// proper uint16
var StringToType = typeToString.Flip()
