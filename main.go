package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/miekg/dns"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.atomizer.io/stream"
	"go.devnw.com/ttl"
	"golang.org/x/exp/slog"
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
	logger := configLogger() //.Sugar()

	var localSrcs Sources
	err := viper.UnmarshalKey("dns.local", &localSrcs)
	if err != nil {
		logger.ErrorCtx(ctx,
			"failed to unmarshal local sources",
			slog.String("error", err.Error()),
		)
	}

	var allowSrcs Sources
	err = viper.UnmarshalKey("dns.allow", &allowSrcs)
	if err != nil {
		logger.ErrorCtx(ctx,
			"failed to unmarshal allow sources",
			slog.String("error", err.Error()),
		)
	}

	var blockSrcs Sources
	err = viper.UnmarshalKey("dns.block", &blockSrcs)
	if err != nil {
		logger.ErrorCtx(ctx,
			"failed to unmarshal block sources",
			slog.String("error", err.Error()),
		)
	}

	port := uint16(viper.GetUint("dns.port"))
	upstreams := viper.GetStringSlice("dns.upstream")

	cacheDir := viper.GetString("dns.cache")
	if cacheDir != "" {
		err := os.MkdirAll(cacheDir, 0o755)
		if err != nil {
			logger.ErrorCtx(ctx,
				"failed to create cache directory",
				slog.String("error", err.Error()),
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
		logger.ErrorCtx(ctx,
			"failed to initialize upstreams",
			slog.String("error", err.Error()),
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
		logger.ErrorCtx(ctx,
			"failed to create local resolver",
			slog.String("error", err.Error()),
		)
	}

	allow, err := AllowResolver(ctx, logger, upStreamFan, allowSrcs.Records(ctx, logger, cacheDir)...)
	if err != nil {
		logger.ErrorCtx(ctx,
			"failed to create allow resolver",
			slog.String("error", err.Error()),
		)
	}

	block, err := BlockResolver(ctx, logger, blockSrcs.Records(ctx, logger, cacheDir)...)
	if err != nil {
		logger.ErrorCtx(ctx,
			"failed to create block resolver",
			slog.String("error", err.Error()),
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

	logger.InfoCtx(ctx,
		"dns service initialized",
		slog.Int("port", int(port)),
		slog.String("upstream", strings.Join(upstreams, ",")),
	)

	go func() {
		<-ctx.Done()
		err := server.Shutdown()
		if err != nil {
			logger.ErrorCtx(ctx,
				"failed to gracefully shutdown server",
				slog.String("error", err.Error()),
			)
		}
	}()

	err = server.ListenAndServe()
	if err != nil {
		logger.ErrorCtx(ctx,
			"failed to start server",
			slog.String("error", err.Error()),
		)
	}
}

type Initializer[T, U any] struct {
	logger *slog.Logger
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
		i.logger.ErrorCtx(ctx,
			"error executing scaler",
			slog.String("error", err.Error()),
		)
	}

	return out
}
