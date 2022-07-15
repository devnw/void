package main

import "fmt"

type Category string

const (
	LOCAL    Category = "local"
	ALLOW    Category = "allow"
	BLOCK    Category = "block"
	CACHE    Category = "cache"
	UPSTREAM Category = "upstream"
)

type Error struct {
	Category Category `json:"category"`
	Server   string   `json:"server"`
	Msg      string   `json:"msg"`
	Inner    error    `json:"inner"`
}

func (e Error) String() string {
	msg := fmt.Sprintf("%s: %s", e.Msg, e.Inner)

	if e.Server != "" {
		msg = fmt.Sprintf("%s | %s", e.Server, msg)
	}

	if e.Category != "" {
		msg = fmt.Sprintf("%s | %s", e.Category, msg)
	}

	return msg
}

func (e Error) Error() string {
	return e.String()
}

func (e Error) Unwrap() error {
	wrapped, ok := e.Inner.(wrappedErr)
	if !ok {
		return e.Inner
	}

	return wrapped.Unwrap()
}

type wrappedErr interface {
	Unwrap() error
}
