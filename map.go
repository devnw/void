package main

import "sync"

// SMap is a sync.Map type that automatically executes
// type assertions using Go generics.
type SMap[U comparable, T any] sync.Map
