package main

import (
	"context"
	"fmt"
	"io"
	"log"
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
	"gopkg.in/natefinch/lumberjack.v2"
)

// DEFAULTTTL defines the default ttl for records that either do not
// provide a TTL, are blocked, or are local records.
const DEFAULTTTL = 3600

const defaultLife = time.Millisecond * 100
const defaultWait = time.Millisecond

func init() {
	viper.SetEnvPrefix("VOID")
}

func main() {
	var err error
	defer func() {
		if err != nil {
			log.Fatal(err)
		}
	}()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	err = root.ExecuteContext(ctx)
}

//nolint:funlen // this contains the CLI flags
func exec(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()

	pub := event.NewPublisher(ctx)

	logConfig := &lumberjack.Logger{}
	err := viper.UnmarshalKey("logger", logConfig)
	if err != nil {
		log.Fatal(err)
	}

	spew.Dump(logConfig)

	logger, err := configLogger(ctx, logConfig)
	if err != nil {
		log.Fatal(err)
	}

	logger.Errorc(ctx, pub.ReadErrors(0).Interface())

	if viper.GetBool("verbose") {
		logger.Printc(ctx, pub.ReadEvents(0).Interface())
	}

	var localSrcs Sources
	err = viper.UnmarshalKey("dns.local", &localSrcs)
	if err != nil {
		log.Fatal(err)
	}

	var allowSrcs Sources
	err = viper.UnmarshalKey("dns.allow", &allowSrcs)
	if err != nil {
		log.Fatal(err)
	}

	var blockSrcs Sources
	err = viper.UnmarshalKey("dns.block", &blockSrcs)
	if err != nil {
		log.Fatal(err)
	}

	port := uint16(viper.GetUint("dns.port"))
	upstreams := viper.GetStringSlice("dns.upstream")

	cacheDir := viper.GetString("dns.cache")
	if cacheDir != "" {
		err := os.MkdirAll(cacheDir, 0o755)
		if err != nil {
			log.Fatal(err)
		}
	}

	i := &Initializer[*Request, *Request]{pub}

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

	local, err := LocalResolver(ctx, pub, localSrcs.Records(ctx, pub, cacheDir)...)
	if err != nil {
		log.Fatal(err)
	}

	allow, err := AllowResolver(ctx, pub, upStreamFan, allowSrcs.Records(ctx, pub, cacheDir)...)
	if err != nil {
		log.Fatal(err)
	}

	block, err := BlockResolver(ctx, pub, blockSrcs.Records(ctx, pub, cacheDir)...)
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

	pub.EventFunc(ctx, func() event.Event {
		return &Event{
			Msg: fmt.Sprintf(
				"void listening on port %v; upstream [%s]",
				port,
				strings.Join(upstreams, ", "),
			),
		}
	})

	go func() {
		<-ctx.Done()
		err := server.Shutdown()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down server: %v\n", err)
		}
	}()

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
		Wait: defaultWait,
		Life: defaultLife,
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

func configLogger(
	ctx context.Context, jack *lumberjack.Logger,
) (alog.Logger, error) {
	eOut := io.Writer(os.Stderr)
	iOut := io.Writer(os.Stdout)
	if len(jack.Filename) > 0 && jack.Filename != ":stdout:" {
		eOut = jack
		iOut = jack
	}

	return alog.New(
		ctx,
		"",
		alog.DEFAULTTIMEFORMAT,
		time.UTC,
		0,
		[]alog.Destination{
			{
				Types:  alog.INFO | alog.DEBUG,
				Format: alog.JSON,
				Writer: iOut,
			},
			{
				Types:  alog.ERROR | alog.CRIT | alog.FATAL,
				Format: alog.JSON,
				Writer: eOut,
			},
		}...,
	)
}
