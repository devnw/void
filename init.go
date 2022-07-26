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

func init() {
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

	viper.BindPFlag("port", root.PersistentFlags().Lookup("port"))
	viper.BindPFlag("upstream", root.PersistentFlags().Lookup("upstream"))

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
