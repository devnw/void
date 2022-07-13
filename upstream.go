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
	"strconv"
	"time"

	"github.com/miekg/dns"
)

var protoReg = `(tcp|udp|tcp-tls){0,1}(?:\:\/\/){0,1}`
var ipv4Reg = `(?:[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})`
var ipv6Reg = `(?:(?:[0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,7}:|(?:[0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,5}(?::[0-9a-fA-F]{1,4}){1,2}|(?:[0-9a-fA-F]{1,4}:){1,4}(?::[0-9a-fA-F]{1,4}){1,3}|(?:[0-9a-fA-F]{1,4}:){1,3}(?::[0-9a-fA-F]{1,4}){1,4}|(?:[0-9a-fA-F]{1,4}:){1,2}(?::[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:(?:(?::[0-9a-fA-F]{1,4}){1,6})|:(?:(?::[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(?::[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(?:ffff(?::0{1,4}){0,1}:){0,1}(?:(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9])|(?:[0-9a-fA-F]{1,4}:){1,4}:(?:(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`
var portReg = `\:{0,1}([0-9]{1,5}){0,1}`

// addrReg is a regular expression for matching the supported
// address formats
// <proto>://<server>[:<port>]
var addrReg = regexp.MustCompile(
	fmt.Sprintf(`^%s(%s|%s)%s$`, protoReg, ipv4Reg, ipv6Reg, portReg),
)

// Network is a type alias of string for categorizing
// protocols for a DNS server
type Network string

const (
	// UDP is the network type for UDP
	UDP Network = "udp"

	// TCP is the network type for TCP
	TCP Network = "tcp"

	// TLS is the network type for TLS over TCP
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
func Up(ctx context.Context, addresses ...string) ([]Upstream, error) {
	upstreams := make([]Upstream, 0, len(addresses))

	for _, address := range addresses {
		matches := addrReg.FindStringSubmatch(address)
		if len(matches) != 4 {
			return nil, fmt.Errorf("invalid address [%s]", address)
		}

		network := UDP
		if matches[1] != "" {
			network = Network(matches[1])
		}

		port := 53
		if matches[3] != "" {
			var err error
			port, err = strconv.Atoi(matches[3])
			if err != nil {
				return nil, err
			}
		}

		upstreams = append(upstreams, Upstream{
			Network: network,
			Address: net.ParseIP(matches[2]),
			Port:    uint16(port),
		})
	}

	return upstreams, nil
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
