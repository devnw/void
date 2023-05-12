package main

import (
	"io"
	"log"
	"os"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func configLogger() *zap.Logger {
	jack := &lumberjack.Logger{}
	err := viper.UnmarshalKey("logger", jack)
	if err != nil {
		log.Fatal(err)
	}

	// If the lumberjack logger is not configured, then we will use the
	// default zap logger. Otherwise, we will use the lumberjack logger
	// as the output for the zap logger.
	var w io.Writer = os.Stderr
	if len(jack.Filename) > 0 {
		if jack.Filename == ":stdout:" {
			w = os.Stdout
		} else {
			w = jack
		}
	}

	level := zapcore.ErrorLevel
	if viper.IsSet("logger.level") {
		l, err := zap.ParseAtomicLevel(viper.GetString("logger.level"))
		if err != nil {
			log.Fatal(err)
		}
		level = l.Level()
	}

	ec := zap.NewProductionEncoderConfig()
	dev := viper.GetBool("logger.dev")
	if dev {
		level = zapcore.DebugLevel
		ec = zap.NewDevelopmentEncoderConfig()
	}

	enc := zapcore.NewJSONEncoder(ec)
	if viper.GetString("logger.format") == "console" {
		enc = zapcore.NewConsoleEncoder(ec)
	}

	core := zapcore.NewCore(
		enc,
		zapcore.AddSync(w),
		level,
	)

	return zap.New(core, zap.WithCaller(
		viper.GetBool("verbose"),
	))
}

type Logger interface {
	Debugf(format string, args ...interface{})
	Debug(args ...interface{})
	Debugw(msg string, keysAndValues ...interface{})

	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Infow(msg string, keysAndValues ...interface{})

	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Warnw(msg string, keysAndValues ...interface{})

	Errorf(format string, args ...interface{})
	Error(args ...interface{})
	Errorw(msg string, keysAndValues ...interface{})

	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})

	// https://github.com/uber-go/zap/blob/85c4932ce3ea76b6babe3e0a3d79da10ef295b8d/FAQ.md#whats-dpanic
	DPanic(args ...interface{})
	DPanicf(format string, args ...interface{})
	DPanicw(msg string, keysAndValues ...interface{})
}

type NOOPLogger struct{}

var _ Logger = (*NOOPLogger)(nil)

func (n *NOOPLogger) Debugf(_ string, _ ...interface{}) {}
func (n *NOOPLogger) Debug(_ ...interface{})            {}
func (n *NOOPLogger) Debugw(_ string, _ ...interface{}) {
}

func (n *NOOPLogger) Info(_ ...interface{})            {}
func (n *NOOPLogger) Infof(_ string, _ ...interface{}) {}
func (n *NOOPLogger) Infow(_ string, _ ...interface{}) {
}

func (n *NOOPLogger) Warn(_ ...interface{})            {}
func (n *NOOPLogger) Warnf(_ string, _ ...interface{}) {}
func (n *NOOPLogger) Warnw(_ string, _ ...interface{}) {
}

func (n *NOOPLogger) Errorf(_ string, _ ...interface{}) {}
func (n *NOOPLogger) Error(_ ...interface{})            {}
func (n *NOOPLogger) Errorw(_ string, _ ...interface{}) {
}

func (n *NOOPLogger) Fatal(_ ...interface{})            {}
func (n *NOOPLogger) Fatalf(_ string, _ ...interface{}) {}
func (n *NOOPLogger) Fatalw(_ string, _ ...interface{}) {
}

func (n *NOOPLogger) DPanic(_ ...interface{})            {}
func (n *NOOPLogger) DPanicf(_ string, _ ...interface{}) {}
func (n *NOOPLogger) DPanicw(_ string, _ ...interface{}) {
}
