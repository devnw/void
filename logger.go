package main

import (
	"io"
	"log"
	"os"

	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
	"gopkg.in/natefinch/lumberjack.v2"
)

func configLogger() *slog.Logger {
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

	return slog.New(slog.NewTextHandler(w, nil))

	//level := zapcore.ErrorLevel
	//if viper.IsSet("logger.level") {
	//	l, err := zap.ParseAtomicLevel(viper.GetString("logger.level"))
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	level = l.Level()
	//}

	//ec := zap.NewProductionEncoderConfig()
	//dev := viper.GetBool("logger.dev")
	//if dev {
	//	level = zapcore.DebugLevel
	//	ec = zap.NewDevelopmentEncoderConfig()
	//}

	//enc := zapcore.NewJSONEncoder(ec)
	//if viper.GetString("logger.format") == "console" {
	//	enc = zapcore.NewConsoleEncoder(ec)
	//}

	//core := zapcore.NewCore(
	//	enc,
	//	zapcore.AddSync(w),
	//	level,
	//)

	//return zap.New(core, zap.WithCaller(
	//	viper.GetBool("verbose"),
	//))
}

//type Logger = slog.Logger

//type Logger interface {
//	Debugf(format string, args ...interface{})
//	Debug(args ...interface{})
//	Debugw(msg string, keysAndValues ...interface{})
//
//	Info(args ...interface{})
//	Infof(format string, args ...interface{})
//	Infow(msg string, keysAndValues ...interface{})
//
//	Warn(args ...interface{})
//	Warnf(format string, args ...interface{})
//	Warnw(msg string, keysAndValues ...interface{})
//
//	Errorf(format string, args ...interface{})
//	Error(args ...interface{})
//	Errorw(msg string, keysAndValues ...interface{})
//
//	Fatal(args ...interface{})
//	Fatalf(format string, args ...interface{})
//	Fatalw(msg string, keysAndValues ...interface{})
//
//	// https://github.com/uber-go/zap/blob/85c4932ce3ea76b6babe3e0a3d79da10ef295b8d/FAQ.md#whats-dpanic
//	DPanic(args ...interface{})
//	DPanicf(format string, args ...interface{})
//	DPanicw(msg string, keysAndValues ...interface{})
//}
