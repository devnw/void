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
	"go.atomizer.io/stream"
	"go.devnw.com/alog"
	"go.devnw.com/event"
)

// DEFAULTTTL defines the default ttl for records that either do not
// provide a TTL, are blocked, or are local records.
const DEFAULTTTL = 3600

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
		pub := event.NewPublisher(ctx)

		i := &Initializer[*Request, *Request]{pub}
		alog.Printc(ctx, pub.ReadEvents(0).Interface())
		alog.Errorc(ctx, pub.ReadErrors(0).Interface())

		server := &dns.Server{
			Addr: ":" + strconv.Itoa(port),
			Net:  "udp",
		}

		//	client := &dns.Client{}

		handler, requests := Convert(ctx, pub, true)

		// Register the handler into the dns server
		dns.HandleFunc(".", handler)

		//go func() {
		//	for {
		//		select {
		//		case <-ctx.Done():
		//			return
		//		case r, ok := <-requests:
		//			if !ok {
		//				return
		//			}

		//			fmt.Println(r.Record())
		//			err := r.Block()
		//			if err != nil {
		//				fmt.Printf("ERROR: %s\n", err)
		//			}
		//		}
		//	}
		//}()

		upstream, err := Up(
			ctx,
			pub,
			"tcp-tls://1.1.1.1:853",
			"tcp-tls://1.0.0.1:853",
			"1.1.1.1",
			"1.0.0.1",
			"8.8.8.8",
			"8.8.4.4",
		)
		if err != nil {
			log.Fatal(err)
		}

		up := make([]chan<- *Request, 0, len(upstream))
		for _, u := range upstream {
			fmt.Printf("[%s] connecting\n", u.String())
			toUp := make(chan *Request)
			i.Scale(
				ctx,
				toUp,
				u.Intercept,
			)
			up = append(up, toUp)
		}

		upStreamFan := make(chan *Request)
		go stream.FanOut(ctx, upStreamFan, up...)

		//cache := &Cache{
		//	ctx,
		//	pub,
		//	ttl.NewCache[string, *dns.Msg](ctx, time.Minute, false),
		//}

		//local, err := LocalResolver(ctx, pub)
		//if err != nil {
		//	log.Fatal(err)
		//}

		//allow, err := AllowResolver(ctx, pub, upStreamFan)
		//if err != nil {
		//	log.Fatal(err)
		//}

		//block, err := BlockResolver(ctx, pub)
		//if err != nil {
		//	log.Fatal(err)
		//}

		go stream.Pipe( // Upstream FanOut
			ctx,
			requests,
			//i.Scale( // Block
			//	ctx,
			//	i.Scale( // Allow
			//		ctx,
			//		i.Scale( // Local
			//			ctx,
			//			i.Scale( // Cache
			//				ctx,
			//				requests,
			//				cache.Intercept,
			//			),
			//			local.Intercept,
			//		),
			//		allow.Intercept,
			//	),
			//	block.Intercept,
			//),
			upStreamFan,
		)
		if err != nil {
			log.Fatal(err)
		}

		err = server.ListenAndServe()
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

type Initializer[T, U any] struct {
	pub *event.Publisher
}

func (i *Initializer[T, U]) Scale(
	ctx context.Context,
	in <-chan T,
	f stream.InterceptFunc[T, U],
) <-chan U {
	s := stream.Scaler[T, U]{
		Wait: time.Nanosecond,
		Life: time.Millisecond,
		Fn:   f,
	}

	out, err := s.Exec(ctx, in)
	if err != nil {
		i.pub.ErrorFunc(ctx, func() error {
			return err
		})
	}

	return out
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
