package cmd

import (
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

var rootCmd = &cobra.Command{
	Use:          "temperature-api",
	Short:        "API for reading current environment measurements from InfluxDB",
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.temperature-api.toml)")
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("/etc/temperature-api")
		viper.AddConfigPath("$HOME/.temperature-api")
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := viper.ReadInConfig(); err == nil {
		slog.Info("Using config file", "config", viper.ConfigFileUsed())
	}
}
