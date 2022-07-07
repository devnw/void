package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/spf13/cobra"
	"go.devnw.com/alog"
	"go.devnw.com/ttl"
)

var version string

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port := 5300 //53
	root := &cobra.Command{
		Use:     "void [flags]",
		Short:   "void is a simple cluster based dns provider/sink",
		Version: version,
		Run:     exec(ctx, port),
	}

	err := root.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func exec(ctx context.Context, port int) func(cmd *cobra.Command, _ []string) {
	server := &dns.Server{
		Addr: ":" + strconv.Itoa(port),
		Net:  "udp",
	}

	client := &dns.Client{}

	dns.HandleFunc(".", (&local{
		ctx,
		map[string]net.IP{},
		sync.RWMutex{},
	}).Handler(
		(&void{
			ctx:   ctx,
			allow: map[string]*Record{},
			deny:  map[string]*Record{},
		}).Handler(
			(&cached{
				ctx,
				ttl.NewCache[string, *dns.Msg](ctx, time.Minute, true),
			}).Handler(
				func(w dns.ResponseWriter, req *dns.Msg) {
					log.Printf("%s\n", req.Question[0].Name)

					res, rtt, err := client.Exchange(req, "1.1.1.1:53")
					if err != nil {
						log.Printf("%s\n", err)
						return
					}

					log.Printf("%+v\n", res)
					log.Printf("%+v\n", rtt)
					w.WriteMsg(res)
				}),
		),
	))

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func configLogger(ctx context.Context, prefix string) error {
	return alog.Global(
		ctx,
		prefix,
		alog.DEFAULTTIMEFORMAT,
		time.UTC,
		0,
		[]alog.Destination{
			{
				Types:  alog.INFO | alog.DEBUG,
				Format: alog.JSON,
				Writer: os.Stdout,
			},
			{
				Types:  alog.ERROR | alog.CRIT | alog.FATAL,
				Format: alog.JSON,
				Writer: os.Stderr,
			},
		}...,
	)
}
