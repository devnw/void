package main

import (
	"context"

	"golang.org/x/exp/slog"
)

//go:generate curl https://data.iana.org/TLD/tlds-alpha-by-domain.txt -o tlds.txt

func TLDVerify(
	ctx context.Context,
	logger *slog.Logger,
	files ...string,
) (*TLD, error) {
	// Read the files into a map as lowercase keys
	// domains are `.` separated, so we can split on `.` and check the last
	// element of the slice

	return &TLD{
		ctx:    ctx,
		logger: logger,
	}, nil
}

type TLD struct {
	ctx    context.Context
	logger *slog.Logger
}

func (t *TLD) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {
	return req, true
}
