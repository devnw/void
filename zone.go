package main

import (
	"fmt"
	"io"

	"github.com/miekg/dns"
)

//go:generate curl https://www.internic.net/domain/root.zone -o root.zone

func ParseZone(r io.ReadCloser, ipv4, ipv6 bool) *dns.Msg {
	defer r.Close()

	// Create a new tokenizer
	t := dns.NewZoneParser(r, "", "")

	// NOTE: (miekg/dns) Callers should not assume all returned data in an
	// Resource Record is syntactically correct, e.g. illegal base64 in RRSIGs
	// will be returned as-is.

	msg := &dns.Msg{}
	for rr, ok := t.Next(); ok; rr, ok = t.Next() {
		if t.Err() != nil {
			fmt.Println("Error parsing file:", t.Err())
			return nil
		}

		switch rr.(type) {
		case *dns.A:
			if ipv4 {
				msg.Extra = append(msg.Extra, rr)
			}
		case *dns.AAAA:
			if ipv6 {
				msg.Extra = append(msg.Extra, rr)
			}
		case *dns.NS:
			msg.Ns = append(msg.Ns, rr)
		}
	}

	return msg
}

/*
The root zone file is a text file that describes the configuration of the top-level Domain Name System (DNS) zones, including the list of top-level domains and the Internet's authoritative DNS servers.

Each line of the root zone file represents a DNS resource record. A resource record is a data structure containing specific values, including the domain name it refers to, its time-to-live (TTL), record class, record type, and record-specific data.

Here are the types of records you might see in the root zone file:

SOA (Start of Authority): The SOA record starts every zone file and signifies the beginning of the list of DNS records for the domain. It contains the primary name server, the email of the domain administrator, the domain serial number, and several timers relating to refreshing the zone.

NS (Name Server): The NS records list the authoritative name servers for the zone. In the root zone file, these point to the root servers (e.g., a.root-servers.net, b.root-servers.net, etc.).

A (Address): The A records map a domain name to an IPv4 address. In the root zone file, these records map each root server's domain name to its IPv4 address.

AAAA (Address): Similar to A records but for IPv6 addresses. These records map each root server's domain name to its IPv6 address.

MX (Mail Exchanger): The MX records point to the servers that should receive email for the domain. The root zone doesn't typically contain MX records because it doesn't handle email.

TXT (Text): The TXT records contain human-readable text. They're often used for various purposes, including verifying domain ownership and SPF records for email validation. Again, you won't typically see these in the root zone.

DNSKEY (DNS Public Key): DNSKEY records contain the public keys that a DNS zone uses to sign its records. This is part of DNSSEC, an extension to DNS that adds digital signatures to DNS data to verify its authenticity and integrity.

DS (Delegation Signer): DS records are used in DNSSEC to achieve authenticated delegation. They contain a digest of a DNSKEY record which is placed in the parent zone, establishing a chain of trust.

RRSIG (Resource Record Signature): RRSIG records are used in DNSSEC and hold the digital signature of a record set. This signature helps to verify the authenticity and integrity of the data.

NSEC (Next Secure): NSEC records are used in DNSSEC to provide authenticated denial of existence for DNS records. This means they can prove that a certain record doesn't exist.

NSEC3 (Next Secure version 3) and NSEC3PARAM (NSEC3 Parameters): These are part of DNSSEC and are similar to NSEC records, but they add additional security by hashing the names of the records.

The actual data contained in these records will vary based on the type of the record. For example, an A record will contain an IPv4 address, while an NS record will contain a domain name.
*/
