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

const portReg = `(\:{1}[0-9]{1,5}){0,1}`
const protoReg = `(tcp|udp|tcp-tls){0,1}(?:\:\/\/){0,1}`
const ipv4Reg = `(?:[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})`
const ipv6Reg = `(?:(?:[0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,7}:|(?:[0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,5}(?::[0-9a-fA-F]{1,4}){1,2}|(?:[0-9a-fA-F]{1,4}:){1,4}(?::[0-9a-fA-F]{1,4}){1,3}|(?:[0-9a-fA-F]{1,4}:){1,3}(?::[0-9a-fA-F]{1,4}){1,4}|(?:[0-9a-fA-F]{1,4}:){1,2}(?::[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:(?:(?::[0-9a-fA-F]{1,4}){1,6})|:(?:(?::[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(?::[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(?:ffff(?::0{1,4}){0,1}:){0,1}(?:(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9])|(?:[0-9a-fA-F]{1,4}:){1,4}:(?:(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`

// addrReg is a regular expression for matching the supported
// address formats
// <proto>://<server>[:<port>]
var addrReg = regexp.MustCompile(
	fmt.Sprintf(`^%s(%s|%s)%s$`, protoReg, ipv4Reg, ipv6Reg, portReg),
)

// Protocol is a type alias of string for categorizing
// protocols for a DNS server
type Protocol string

const (
	// UDP is the network type for UDP
	UDP Protocol = "udp"

	// TCP is the network type for TCP
	TCP Protocol = "tcp"

	// TLS is the network type for TLS over TCP
	TLS Protocol = "tcp-tls"
)

// TLSConfig load a preset tls configuration adding a custom CA certificate
// to the system trust store if provided
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
		//var caCert []byte
		//caCert, err = os.ReadFile(ca)
		//if err != nil {
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
// by the address. The address should follow the format:
func Up(ctx context.Context, addresses ...string) ([]Upstream, error) {
	upstreams := make([]Upstream, 0, len(addresses))

	for _, address := range addresses {
		matches := addrReg.FindStringSubmatch(address)
		if len(matches) != 4 {
			return nil, fmt.Errorf("invalid address [%s]", address)
		}

		proto := UDP
		if matches[1] != "" {
			proto = Protocol(matches[1])
		}

		port := 53
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
			var err error
			tlsConfig, err = TLSConfig(nil)
			if err != nil {
				return nil, err
			}
		}

		upstreams = append(upstreams, Upstream{
			proto:   proto,
			address: net.ParseIP(matches[2]),
			port:    uint16(port),
			client: &dns.Client{
				Net:       string(proto),
				TLSConfig: tlsConfig,
			},
		})
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
	conn <-chan *dns.Conn

	// Time before reconnecting the client
	reconnect time.Duration
}

// init instantiates the internal routine for managing the connection
// re-instantiation for the upstream server.
func (u *Upstream) init(ctx context.Context) <-chan *dns.Conn {
	out := make(chan *dns.Conn)

	go func() {
		ticker := time.NewTicker(u.reconnect)
		defer ticker.Stop()
		defer close(out)

		// Initial connection
		conn, err := u.client.DialContext(ctx, u.String())
		if err != nil {
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Attempt Reconnect
				newConn, err := u.client.DialContext(ctx, u.String())
				if err != nil {
					continue
				}

				// Close the connection
				conn.Close()

				// Update the connection
				conn = newConn
			case out <- conn:
			}
		}
	}()

	return out
}

func (u *Upstream) String() string {
	return fmt.Sprintf(
		"%s://%s:%d",
		u.proto,
		u.address,
		u.port,
	)
}

// TODO: Determine if upstream should be sequential or
// in parallel

func (u *Upstream) Intercept(
	ctx context.Context,
	req *Request,

	// Named variables allow for implicit return since this
	// implementation will never pass down the request
) (s struct{}, cont bool) {
	select {
	case <-ctx.Done():
		return
	case conn, ok := <-u.conn:
		if !ok {
			return
		}

		// TODO: add logging and rtt calculation

		// Send the Request
		resp, _, err := u.client.ExchangeWithConn(req.r, conn)
		if err != nil {
			return
		}

		err = req.w.WriteMsg(resp)
		if err != nil {
			return
		}

		return
	}
}
