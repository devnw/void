package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/miekg/dns"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.atomizer.io/stream"
	"go.devnw.com/ttl"
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
		r := recover()
		if r != nil {
			err = errors.Join(fmt.Errorf("panic: %v", r), err)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
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
	logger := configLogger().Sugar()

	// create the influxdb client
	if viper.IsSet("influxdb") {
		client := influxdb2.NewClientWithOptions(
			viper.GetString("influxdb.api.addr"),
			viper.GetString("influxdb.api.token"),
			influxdb2.DefaultOptions().SetBatchSize(
				viper.GetUint("influxdb.api.batch"),
			),
		)

		writeAPI := client.WriteAPI(
			viper.GetString("influxdb.org"),
			viper.GetString("influxdb.bucket"),
		)

		defer func() {
			// Force all unwritten data to be sent
			writeAPI.Flush()

			// Ensures background processes finishes
			client.Close()
		}()

		ctx = context.WithValue(ctx, "influxdb.writer", writeAPI)
	}

	var localSrcs Sources
	err := viper.UnmarshalKey("dns.local", &localSrcs)
	if err != nil {
		logger.Fatalw(
			"failed to unmarshal local sources",
			"error", err,
		)
	}

	var allowSrcs Sources
	err = viper.UnmarshalKey("dns.allow", &allowSrcs)
	if err != nil {
		logger.Fatalw(
			"failed to unmarshal allow sources",
			"error", err,
		)
	}

	var blockSrcs Sources
	err = viper.UnmarshalKey("dns.block", &blockSrcs)
	if err != nil {
		logger.Fatalw(
			"failed to unmarshal block sources",
			"error", err,
		)
	}

	port := uint16(viper.GetUint("dns.port"))
	upstreams := viper.GetStringSlice("dns.upstream")

	cacheDir := viper.GetString("dns.cache")
	if cacheDir != "" {
		err := os.MkdirAll(cacheDir, 0o755)
		if err != nil {
			logger.Fatalw(
				"failed to create cache directory",
				"error", err,
			)
		}
	}

	i := &Initializer[*Request, *Request]{logger}

	server := &dns.Server{
		Addr: ":" + strconv.Itoa(int(port)),
		Net:  "udp",
	}

	//	client := &dns.Client{}

	handler, requests := Convert(
		ctx,
		logger,
		true,
	)

	// Register the handler into the dns server
	dns.HandleFunc(".", handler)

	upstream, err := Up(
		ctx,
		logger,
		upstreams...,
	)
	if err != nil {
		logger.Fatalw(
			"failed to initialize upstreams",
			"error", err,
		)
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
		logger,
		ttl.NewCache[string, *dns.Msg](ctx, time.Minute, false),
	}

	local, err := LocalResolver(ctx, logger, localSrcs.Records(ctx, logger, cacheDir)...)
	if err != nil {
		logger.Fatalw(
			"failed to create local resolver",
			"error", err,
		)
	}

	allow, err := AllowResolver(ctx, logger, upStreamFan, allowSrcs.Records(ctx, logger, cacheDir)...)
	if err != nil {
		logger.Fatalw(
			"failed to create allow resolver",
			"error", err,
		)
	}

	block, err := BlockResolver(ctx, logger, blockSrcs.Records(ctx, logger, cacheDir)...)
	if err != nil {
		logger.Fatalw(
			"failed to create block resolver",
			"error", err,
		)
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

	logger.Infow(
		"dns service initialized",
		"port", port,
		"upstream", upstreams,
	)

	go func() {
		<-ctx.Done()
		err := server.Shutdown()
		if err != nil {
			logger.Errorw("failed to gracefully shutdown server", "error", err)
		}
	}()

	err = server.ListenAndServe()
	if err != nil {
		logger.Errorw("failed to start server", "error", err)
	}
}

type Initializer[T, U any] struct {
	logger Logger
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
		i.logger.Errorw(
			"error executing scaler",
			"error", err,
		)
	}

	return out
}
