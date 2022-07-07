package main

import "sync"

type SMap[U comparable, T any] sync.Map
