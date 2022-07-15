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

type Error struct {
	Category Category `json:"category"`
	Server   string   `json:"server"`
	Msg      string   `json:"msg"`
	Inner    error    `json:"inner"`
	Domain   string   `json:"domain"`
}

func (e Error) String() string {
	msg := fmt.Sprintf("%s: %s", e.Msg, e.Inner)

	if e.Domain != "" {
		msg = fmt.Sprintf("%s | %s", e.Domain, msg)
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
	wrapped, ok := e.Inner.(wrappedErr)
	if !ok {
		return e.Inner
	}

	return wrapped.Unwrap()
}

type wrappedErr interface {
	Unwrap() error
}
