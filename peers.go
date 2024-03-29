package main

import (
	"context"
	"fmt"
	"regexp"
)

const peerProtoReg = `(tcp|quic|tcp-tls){0,1}(?:\:\/\/){0,1}`

// peerAddrReg is a regular expression for matching the supported
// address formats
// <proto>://<server>[:<port>].
var peerAddrReg = regexp.MustCompile(
	fmt.Sprintf(`^%s(%s|%s)%s$`, peerProtoReg, ipv4Reg, ipv6Reg, portReg),
)

func Peers(
	ctx context.Context,
	addresses ...string,
) error {
	for _, addr := range addresses {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if !peerAddrReg.MatchString(addr) {
				return fmt.Errorf("invalid peer address: %s", addr)
			}
		}
	}

	return nil
}
