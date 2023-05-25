package main

//type Host struct {
//	CIDR string   `json:"cidr"`
//	IPv4 []string `json:"ipv4"`
//	IPv6 []string `json:"ipv6"`
//}
//
//type Pattern map[string]any
//
//type Direction string
//
//const (
//	INBOUND  Direction = "inbound"
//	OUTBOUND Direction = "outbound"
//)
//
//type Action string
//
//const (
//	ALLOW Action = "allow"
//	DENY  Action = "deny"
//	ALERT Action = "alert"
//)
//
//type Rule struct {
//	Hosts       []Host        `json:"hosts",yaml:"hosts"`
//	Patterns    []Pattern     `json:"patterns",yaml:"patterns"`
//	Direction   Direction     `json:"direction",yaml:"direction"`
//	Action      Action        `json:"action",yaml:"action"`
//	Priority    int           `json:"priority",yaml:"priority"`
//	Categories  []string      `json:"categories",yaml:"categories"`
//	Tags        []string      `json:"tags",yaml:"tags"`
//	TTL         time.Duration `json:"ttl",yaml:"ttl"`
//	Description string        `json:"description",yaml:"description"`
//	Source      string        `json:"source",yaml:"source"`
//}
