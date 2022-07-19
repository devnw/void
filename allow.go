package main

import (
	"context"
	"regexp"
	"sync"

	"go.devnw.com/event"
)

type Allow struct {
	ctx context.Context
	pub *event.Publisher

	records   map[string]*Record
	recordsMu sync.RWMutex
	patterns  []*regexp.Regexp

	upstream chan<- *Request
}

func (a *Allow) Intercept(
	ctx context.Context,
	req *Request,
) (*Request, bool) {

	return req, false
}

func (a *Allow) init() {

}

func (a *Allow) regex(record string) <-chan string {
	out := make(chan string)
	go func() {
		<-a.ctx.Done()
		close(out)
	}()

	return out
}
