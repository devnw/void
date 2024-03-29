package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// TODO: Add range validation for numbers in both ipv4 and ipv6
// TODO: Add fuzz tests

const (
	portReg  = `(\:{1}[0-9]{1,5}){0,1}`
	protoReg = `(tcp|udp|tcp-tls){0,1}(?:\:\/\/){0,1}`
	ipv4Reg  = `(?:[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})`
	//nolint:lll
	ipv6Reg  = `(?:(?:[0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,7}:|(?:[0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,5}(?::[0-9a-fA-F]{1,4}){1,2}|(?:[0-9a-fA-F]{1,4}:){1,4}(?::[0-9a-fA-F]{1,4}){1,3}|(?:[0-9a-fA-F]{1,4}:){1,3}(?::[0-9a-fA-F]{1,4}){1,4}|(?:[0-9a-fA-F]{1,4}:){1,2}(?::[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:(?:(?::[0-9a-fA-F]{1,4}){1,6})|:(?:(?::[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(?::[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(?:ffff(?::0{1,4}){0,1}:){0,1}(?:(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9])|(?:[0-9a-fA-F]{1,4}:){1,4}:(?:(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`
	matchLen = 4
)

// addrReg is a regular expression for matching the supported
// address formats
// <proto>://<server>[:<port>].
var addrReg = regexp.MustCompile(
	fmt.Sprintf(`^%s(%s|%s)%s$`, protoReg, ipv4Reg, ipv6Reg, portReg),
)

// Protocol is a type alias of string for categorizing
// protocols for a DNS server.
type Protocol string

const (
	// UDP is the network type for UDP.
	UDP Protocol = "udp"

	// TCP is the network type for TCP.
	TCP Protocol = "tcp"

	// TLS is the network type for TLS over TCP.
	TLS Protocol = "tcp-tls"
)

// TLSConfig load a preset tls configuration adding a custom CA certificate
// to the system trust store if provided.
func TLSConfig(caCert []byte) (*tls.Config, error) {
	// Load the system certificate pool
	caPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	// If a CA file is provided, load it and add it
	// to the system certificate pool
	if len(caCert) > 0 {
		// TODO: Move this elsewhere
		// Load the CA certificate
		// var caCert []byte
		// caCert, err = os.ReadFile(ca)
		// if err != nil {
		//	return nil, err
		//}

		ok := caPool.AppendCertsFromPEM(caCert)
		if !ok {
			return nil, errors.New("failed to parse root certificate")
		}
	}

	return &tls.Config{
		MinVersion:               tls.VersionTLS13,
		RootCAs:                  caPool,
		PreferServerCipherSuites: true,
	}, nil
}

// Up creates a new DNS client to an Upstream server as defined
// by the address. The address should follow the format:.
func Up(
	ctx context.Context,
	logger Logger,
	addresses ...string,
) ([]*Upstream, error) {
	err := checkNil(ctx, logger)
	if err != nil {
		return nil, err
	}

	upstreams := make([]*Upstream, 0, len(addresses))

	for _, address := range addresses {
		matches := addrReg.FindStringSubmatch(address)
		if len(matches) != matchLen {
			return nil, fmt.Errorf("invalid address [%s]", address)
		}

		proto := UDP
		if matches[1] != "" {
			proto = Protocol(matches[1])
		}

		port := defaultPort
		p := strings.TrimPrefix(matches[3], ":")
		if p != "" {
			newport, err := strconv.Atoi(p)
			if err != nil {
				return nil, err
			}

			if newport < 1 || newport > 65535 {
				return nil, fmt.Errorf("invalid port [%s]", matches[3])
			}

			port = newport
		}

		// load the appropriate tls configuration
		// if the network is TLS
		var tlsConfig *tls.Config
		if proto == TLS {
			tlsConfig, err = TLSConfig(nil)
			if err != nil {
				return nil, err
			}
		}

		u := &Upstream{
			proto:     proto,
			address:   net.ParseIP(matches[2]),
			port:      uint16(port),
			logger:    logger,
			reconnect: time.Minute,
			client: &dns.Client{
				Net:       string(proto),
				TLSConfig: tlsConfig,
			},
		}

		// Initialize the upstream connection

		upstreams = append(upstreams, u)
	}

	return upstreams, nil
}

// Upstream handles the exchanging of DNS requests with the
// upstream server for a specific request.
type Upstream struct {
	// address of the upstream server
	address net.IP
	port    uint16

	// network indicates the proto to use
	// for the upstream server
	//
	// examples:
	// 		"udp"
	// 		"tcp"
	// 		"tcp-tls"
	proto Protocol

	// Client instance
	client *dns.Client

	// upstream connection

	// Time before reconnecting the client
	reconnect time.Duration
	logger    Logger
}

func (u *Upstream) String() string {
	return fmt.Sprintf(
		"%s://%s",
		u.proto,
		u.addr(),
	)
}

func (u *Upstream) addr() string {
	return net.JoinHostPort(
		u.address.String(),
		strconv.Itoa(int(u.port)),
	)
}

// TODO: Determine if upstream should be sequential or
// in parallel

func (u *Upstream) Intercept(
	ctx context.Context,
	req *Request,

	// Named variables allow for implicit return since this
	// implementation will never pass down the request
) (s *Request, cont bool) {
	select {
	case <-ctx.Done():
		return
	// case conn, ok := <-u.conn:
	// if !ok {
	//	return
	//}
	default:

		// Send the Request
		// TODO: Log RTT
		resp, _, err := u.client.ExchangeContext(ctx, req.r, u.addr())
		// If the connection was broken, reconnect and retry
		//	select {
		//	case <-ctx.Done():
		//		return
		//	case conn, ok := <-u.new:
		//		if !ok {
		//			return
		//		}
		//		resp, _, err = u.client.ExchangeWithConn(req.r, conn)
		//	}
		//}
		if err != nil {
			u.logger.Errorw(
				"failed to exchange request",
				"category", UPSTREAM,
				"server", u.String(),
				"error", err,
				"record", req.String(),
			)
			return
		}

		err = req.w.WriteMsg(resp)
		if err != nil {
			u.logger.Errorw(
				"failed to write response",
				"category", UPSTREAM,
				"server", u.String(),
				"error", err,
				"record", req.String(),
			)
		}

		u.logger.Debugw(
			"sent response",
			"category", UPSTREAM,
			"server", u.String(),
			"record", req.String(),
		)

		return
	}
}
