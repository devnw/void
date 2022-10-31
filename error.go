package main

import (
	"fmt"
)

// checkNil checks if any of the provided values are nil and returns
// an error if they are.
func checkNil(values ...any) error {
	for _, value := range values {
		if value == nil {
			return fmt.Errorf("nil value of type %T", value)
		}
	}

	return nil
}

type Category string

const (
	LOCAL    Category = "local"
	ALLOW    Category = "allow"
	BLOCK    Category = "block"
	CACHE    Category = "cache"
	UPSTREAM Category = "upstream"
)

func (c Category) String() string {
	return string(c)
}

func NewErr(r *Request, inner error, msg string) *Error {
	return &Error{
		Msg:    msg,
		Inner:  inner,
		Record: r.Record(),
		Server: r.server,
		Client: r.client,
	}
}

type Error struct {
	Msg      string   `json:"msg"`
	Inner    error    `json:"inner,omitempty"`
	Record   string   `json:"domain,omitempty"`
	Category Category `json:"category,omitempty"`
	Client   string   `json:"client,omitempty"`
	Server   string   `json:"server,omitempty"`
	Metadata string   `json:"metadata,omitempty"`
}

func (e Error) String() string {
	msg := fmt.Sprintf("%s: %s", e.Msg, e.Inner)

	if e.Record != "" {
		msg = fmt.Sprintf("%s | %s", e.Record, msg)
	}

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
	//nolint:errorlint // this is correctly implemented
	wrapped, ok := e.Inner.(wrappedErr)
	if !ok {
		return e.Inner
	}

	return wrapped.Unwrap()
}

type wrappedErr interface {
	Unwrap() error
}
