package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/miekg/dns"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.atomizer.io/stream"
	"go.devnw.com/alog"
	"go.devnw.com/event"
	"go.devnw.com/ttl"
)

// DEFAULTTTL defines the default ttl for records that either do not
// provide a TTL, are blocked, or are local records.
const DEFAULTTTL = 3600

func init() {
	viper.SetEnvPrefix("VOID")
}

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	err := root.ExecuteContext(ctx)
	if err != nil {
		fmt.Println(err)
		// nolint:gocritic
		os.Exit(1)
	}
}

func exec(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()

	pub := event.NewPublisher(ctx)

	var localSrcs Sources
	err := viper.UnmarshalKey("dns.local", &localSrcs)
	if err != nil {
		log.Fatal(err)
	}

	spew.Dump(localSrcs.Records(ctx))

	var allowSrcs Sources
	err = viper.UnmarshalKey("DNS.Allow", &allowSrcs)
	if err != nil {
		log.Fatal(err)
	}

	spew.Dump(allowSrcs)

	var blockSrcs Sources
	err = viper.UnmarshalKey("DNS.Block", &blockSrcs)
	if err != nil {
		log.Fatal(err)
	}

	spew.Dump(blockSrcs.Records(ctx))

	port := uint16(viper.GetUint("DNS.Port"))
	upstreams := viper.GetStringSlice("DNS.Upstream")

	i := &Initializer[*Request, *Request]{pub}

	if viper.GetBool("verbose") {
		alog.Printc(ctx, pub.ReadEvents(0).Interface())
	}

	alog.Errorc(ctx, pub.ReadErrors(0).Interface())

	server := &dns.Server{
		Addr: ":" + strconv.Itoa(int(port)),
		Net:  "udp",
	}

	//	client := &dns.Client{}

	handler, requests := Convert(ctx, pub, true)

	// Register the handler into the dns server
	dns.HandleFunc(".", handler)

	upstream, err := Up(
		ctx,
		pub,
		upstreams...,
	)
	if err != nil {
		log.Fatal(err)
	}

	up := make([]chan<- *Request, 0, len(upstream))
	for _, u := range upstream {
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

	cache := &Cache{
		ctx,
		pub,
		ttl.NewCache[string, *dns.Msg](ctx, time.Minute, false),
	}

	local, err := LocalResolver(ctx, pub,
		&Record{
			Pattern: "dns.kolhar.net",
			Type:    DIRECT,
			IP:      net.ParseIP("192.168.0.15"),
			Tags:    []string{"local", "kolhar"},
			Source:  "local",
		},
		&Record{
			Pattern: "*kolhar.net",
			Type:    WILDCARD,
			IP:      net.ParseIP("192.168.0.3"),
			Tags:    []string{"local", "kolhar"},
			Source:  "local",
		})
	if err != nil {
		log.Fatal(err)
	}

	allow, err := AllowResolver(ctx, pub, upStreamFan,
		&Record{
			Pattern: "google.com",
			Type:    DIRECT,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	block, err := BlockResolver(ctx, pub, blockSrcs.Records(ctx)...)
	if err != nil {
		log.Fatal(err)
	}

	go stream.Pipe( // Upstream FanOut
		ctx,
		i.Scale( // Block
			ctx,
			i.Scale( // Allow
				ctx,
				i.Scale( // Local
					ctx,
					i.Scale( // Cache
						ctx,
						requests,
						cache.Intercept,
					),
					local.Intercept,
				),
				allow.Intercept,
			),
			block.Intercept,
		),
		upStreamFan,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(
		os.Stderr,
		"void listening on port %v; upstream [%s]\n",
		port,
		strings.Join(upstreams, ", "),
	)
	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
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
		Wait: time.Millisecond,
		Life: time.Millisecond * 100,
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
