package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version string = "dev"

	//nolint:gochecknoglobals // necessary for cobra init
	cfgPath string
)

const (
	defaultConfigDir  = "/etc/void"
	defaultConfigName = "config"
)

//nolint:gochecknoglobals // necessary for cobra root command
var root = &cobra.Command{
	Use:     "void [flags]",
	Short:   "void is a simple cluster based dns provider/sink",
	Version: version,
	Run:     exec,
}

// TODO: Make DNS sink a subcommand
// TODO: Make Proxy a subcommand (future)
// TODO: Root command should base function on config file

func init() {
	var err error
	defer func() {
		if err != nil {
			log.Fatal(err)
		}
	}()

	cobra.OnInitialize(initConfig)

	root.PersistentFlags().StringVar(&cfgPath, "config", "", "config path")

	root.PersistentFlags().BoolP(
		"verbose",
		"v",
		false,
		"verbose output",
	)

	err = viper.BindPFlag("verbose", root.PersistentFlags().Lookup("verbose"))
	if err != nil {
		return
	}

	root.PersistentFlags().String(
		"cache",
		"/etc/void/cache",
		"cache folder for remote sources",
	)

	root.PersistentFlags().String(
		"logs",
		"/var/log/void",
		"directory where logs will be stored, or stdout|stderr if empty",
	)

	// Fix this with: https://umarcor.github.io/cobra/#getting-started
	err = viper.BindPFlag("config", root.PersistentFlags().Lookup("config"))
	if err != nil {
		return
	}

	if viper.GetString("config") != "" {
		viper.SetConfigFile(viper.GetString("config"))
	}

	root.PersistentFlags().Uint16P(
		"port",
		"p",
		53,
		"DNS listening port",
	)

	root.PersistentFlags().StringSliceP(
		"upstream",
		"u",
		[]string{
			"tcp-tls://1.1.1.1:853",
			"tcp-tls://1.0.0.1:853",
		},
		"Upstream DNS Servers",
	)

	root.PersistentFlags().StringSlice(
		"peers",
		[]string{},
		"DNS cluster peers (example: tcp://192.168.0.10, tcp-tls://, quic://)",
	)

	err = viper.BindPFlag("dns.port", root.PersistentFlags().Lookup("port"))
	if err != nil {
		return
	}

	err = viper.BindPFlag("dns.upstream", root.PersistentFlags().Lookup("upstream"))
	if err != nil {
		return
	}

	err = viper.BindPFlag("dns.cache", root.PersistentFlags().Lookup("cache"))
	if err != nil {
		return
	}

	err = viper.BindPFlag("dns.logs", root.PersistentFlags().Lookup("logs"))
	if err != nil {
		return
	}

	err = viper.BindPFlag("dns.peers", root.PersistentFlags().Lookup("peers"))
	if err != nil {
		return
	}
}

func initConfig() {
	if cfgPath != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgPath)
	} else {
		viper.SetConfigName(defaultConfigName)
		viper.AddConfigPath(defaultConfigDir)

		// Check working directory for config
		wd, err := os.Getwd()
		if err == nil {
			viper.AddConfigPath(wd)
		}
	}

	// Check home directory/.void for config
	home, err := os.UserHomeDir()
	if err == nil {
		viper.AddConfigPath(filepath.Join(home, ".void"))
	}

	// Check working directory for config
	wd, err := os.Getwd()
	if err == nil {
		viper.AddConfigPath(wd)
	}

	// Use VOID as a prefix for all environment variables that
	// map to configuration values
	viper.SetEnvPrefix("VOID")
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
}
