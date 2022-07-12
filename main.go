package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/miekg/dns"
	"github.com/spf13/cobra"
	"go.devnw.com/alog"
)

var version string

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port := 5300 // 53
	root := &cobra.Command{
		Use:     "void [flags]",
		Short:   "void is a simple cluster based dns provider/sink",
		Version: version,
		Run:     exec(ctx, port),
	}

	err := root.Execute()
	if err != nil {
		fmt.Println(err)
		// nolint:gocritic
		os.Exit(1)
	}
}

func exec(ctx context.Context, port int) func(cmd *cobra.Command, _ []string) {
	return func(cmd *cobra.Command, _ []string) {
		server := &dns.Server{
			Addr: ":" + strconv.Itoa(port),
			Net:  "udp",
		}

		//	client := &dns.Client{}

		handler, requests := Convert(ctx)

		// Register the handler into the dns server
		dns.HandleFunc(".", handler)

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case r, ok := <-requests:
					if !ok {
						return
					}

					fmt.Println(r.Record())
					err := r.Block()
					if err != nil {
						fmt.Printf("ERROR: %s\n", err)
					}
				}
			}
		}()

		err := server.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}

		//		(&local{
		//		ctx,
		//		map[string]net.IP{},
		//		sync.RWMutex{},
		//	}).Handler(
		//		(&void{
		//			ctx:   ctx,
		//			allow: map[string]*Record{},
		//			deny:  map[string]*Record{},
		//		}).Handler(
		//			(&cached{
		//				ctx,
		//				ttl.NewCache[string, *dns.Msg](ctx, time.Minute, true),
		//			}).Handler(
		//				func(w dns.ResponseWriter, req *dns.Msg) {
		//					log.Printf("%s\n", req.Question[0].Name)
		//
		//					res, rtt, err := client.Exchange(req, "1.1.1.1:53")
		//					if err != nil {
		//						log.Printf("%s\n", err)
		//						return
		//					}
		//
		//					log.Printf("%+v\n", res)
		//					log.Printf("%+v\n", rtt)
		//
		//					err = w.WriteMsg(res)
		//					if err != nil {
		//						// TODO: Handle error
		//						fmt.Printf("Error: %s\n", err)
		//					}
		//				}),
		//		),
		//	))
	}
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
