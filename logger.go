package main

import (
	"io"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type logconfig struct {
	*lumberjack.Logger
	Level  string `json:"level" yaml:"level"`   // debug, info, warn, error
	Format string `json:"format" yaml:"format"` // text or json
	Source bool   `json:"source" yaml:"source"` // include source location
}

func configLogger() *slog.Logger {
	cfg := &logconfig{Logger: &lumberjack.Logger{}}
	err := viper.UnmarshalKey("logger", cfg)
	if err != nil {
		log.Fatal(err)
	}

	// If the lumberjack logger is not configured, then we will use the
	// default zap logger. Otherwise, we will use the lumberjack logger
	// as the output for the zap logger.
	var w io.Writer = os.Stderr
	if len(cfg.Filename) > 0 {
		if cfg.Filename == ":stdout:" {
			w = os.Stdout
		} else {
			w = cfg
		}
	}

	level := slog.LevelError
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	}

	opts := &slog.HandlerOptions{
		AddSource: cfg.Source,
		Level:     level,
	}

	// If the format is not configured, then we will use the default
	// text format. Otherwise, we will use the configured format.
	var handler slog.Handler = slog.NewTextHandler(w, opts)
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(w, opts)
	}

	return slog.New(handler)
}
