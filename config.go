package main

// Type indicates the type of a record to ensure proper analysis.
type Type string

const (
	// DIRECT indicates a direct DNS record, compared 1 to 1.
	DIRECT Type = "direct"

	// WILDCARD indicates a wildcard DNS record, (e.g. *.google.com)
	// which will be converted to the appropriate regex or matched with
	// HasSuffix check.
	WILDCARD Type = "wildcard"

	// REGEX indicates a regular expression to match DNS requests
	// against for blocking many records with a single filter.
	REGEX Type = "regex"
)

func (t Type) String() string {
	return string(t)
}
