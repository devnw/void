package main

import (
	"context"

	"golang.org/x/exp/slog"
)

// SLD (Second Level Domain) is a domain name with the TLD removed
// e.g. www.example.com -> example

type SLD struct {
	ctx    context.Context
	logger *slog.Logger
}
