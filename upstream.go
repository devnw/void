package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"time"

	"github.com/miekg/dns"
)

// addrReg is a regular expression for matching the supported
// address formats
// <proto>://<server>[:<port>]
var addrReg = regexp.MustCompile(
	`^(tcp|udp|tcp-tls){1}:\/\/([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})\:{0,1}([0-9]{1,5}){0,1}$`,
)

type Network string

const (
	UDP Network = "udp"
	TCP Network = "tcp"
	TLS Network = "tcp-tls"
)

// TLSConfig load a preset tls configuration adding a custom CA certificate
// to the system trust store if provided
func TLSConfig(ca string) (*tls.Config, error) {
	// Load the system certificate pool
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	// If a CA file is provided, load it and add it
	// to the system certificate pool
	if ca != "" {
		// Load the CA certificate
		var caCert []byte
		caCert, err = os.ReadFile(ca)
		if err != nil {
			return nil, err
		}

		ok := caCertPool.AppendCertsFromPEM(caCert)
		if !ok {
			return nil, errors.New("failed to parse root certificate")
		}
	}

	return &tls.Config{
		MinVersion:               tls.VersionTLS13,
		RootCAs:                  caCertPool,
		ClientCAs:                caCertPool,
		PreferServerCipherSuites: true,
	}, nil
}

// Up creates a new DNS client to an Upstream server as defined
// by the address. The address should follow the format:
func Up(ctx context.Context, address string) error {
	matches := addrReg.FindStringSubmatch(address)
	if len(matches) != 4 {
		return errors.New("invalid address")
	}

	network := Network(matches[1])
	server := matches[2]
	port := matches[3]

	fmt.Printf("%s %s %s\n", network, server, port)

	return nil
}

type Upstream struct {
	// Address of the upstream server
	Address net.IP `json:"address"`
	Port    uint16 `json:"port"`

	// network indicates the protocol to use
	// for the upstream server
	//
	// examples:
	// 		"udp"
	// 		"tcp"
	// 		"tcp-tls"
	Network Network `json:"network"`

	// Time before reconnecting the client
	reconnect time.Duration

	// Client instance
	client *dns.Client
}

func (u *Upstream) String() string {
	return fmt.Sprintf(
		"%s://%s:%d",
		u.Network,
		u.Address,
		u.Port,
	)
}

func (u *Upstream) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {

	return req, true
}
