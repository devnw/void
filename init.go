package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version string
	root    = &cobra.Command{
		Use:     "void [flags]",
		Short:   "void is a simple cluster based dns provider/sink",
		Version: version,
		Run:     exec,
	}
)

// TODO: Make DNS sink a subcommand
// TODO: Make Proxy a subcommand (future)
// TODO: Root command should base function on config file

func init() {
	root.PersistentFlags().BoolP(
		"verbose",
		"v",
		false,
		"verbose output",
	)
	viper.BindPFlag("verbose", root.PersistentFlags().Lookup("verbose"))

	root.PersistentFlags().String(
		"config",
		"",
		"config file location",
	)

	// Fix this with: https://umarcor.github.io/cobra/#getting-started
	viper.BindPFlag("config", root.PersistentFlags().Lookup("config"))
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

	viper.BindPFlag("DNS.Port", root.PersistentFlags().Lookup("port"))
	viper.BindPFlag("DNS.Upstream", root.PersistentFlags().Lookup("upstream"))
	viper.BindPFlag("DNS.Peers", root.PersistentFlags().Lookup("peers"))

	viper.AutomaticEnv()
	viper.SetConfigName("void")

	viper.AddConfigPath("/etc/void/")

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

	err = viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
}
