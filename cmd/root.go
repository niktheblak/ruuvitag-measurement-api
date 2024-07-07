package cmd

import (
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	logLevel string
	logger   *slog.Logger
)

var rootCmd = &cobra.Command{
	Use:          "temperature-api",
	Short:        "API for reading current environment measurements from InfluxDB",
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var level = new(slog.LevelVar)
		if err := level.UnmarshalText([]byte(viper.GetString("loglevel"))); err != nil {
			return err
		}
		h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
		logger = slog.New(h)
		if viper.ConfigFileUsed() != "" {
			logger.LogAttrs(nil, slog.LevelInfo, "Using config file", slog.String("config", viper.ConfigFileUsed()))
		}
		logger.Info("Using log level", "level", level)
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().StringVar(&logLevel, "loglevel", "", "log level")

	cobra.CheckErr(viper.BindPFlags(rootCmd.PersistentFlags()))

	viper.SetDefault("loglevel", "info")
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.temperature-api")
		viper.AddConfigPath("/etc/temperature-api")
		viper.SetConfigName("config")
	}
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := viper.ReadInConfig(); err != nil {
		// use only command line options
	}
}
