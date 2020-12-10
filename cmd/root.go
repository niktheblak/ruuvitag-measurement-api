package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/niktheblak/temperature-api/internal/server"
	"github.com/niktheblak/temperature-api/pkg/measurement"
)

var rootCmd = &cobra.Command{
	Use:          "temperature-api",
	Short:        "REST API for getting current temperatures",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := measurement.Config{
			Addr:        viper.GetString("influxdb.addr"),
			Username:    viper.GetString("influxdb.username"),
			Password:    viper.GetString("influxdb.password"),
			Database:    viper.GetString("influxdb.database"),
			Measurement: viper.GetString("influxdb.measurement"),
			Timeout:     10 * time.Second,
		}
		svc, err := measurement.New(cfg)
		if err != nil {
			return err
		}
		if err := svc.Ping(); err != nil {
			return err
		}
		defer svc.Close()
		srv := &server.Server{
			Service: svc,
		}
		srv.Routes()
		port := viper.GetInt("http.port")
		return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("influxdb.addr", "http://127.0.0.1:8086", "InfluxDB address")
	rootCmd.PersistentFlags().String("influxdb.username", "", "InfluxDB username")
	rootCmd.PersistentFlags().String("influxdb.password", "", "InfluxDB password")
	rootCmd.PersistentFlags().String("influxdb.database", "ruuvitag", "InfluxDB database")
	rootCmd.PersistentFlags().String("influxdb.measurement", "ruuvitag", "InfluxDB measurement")
	rootCmd.PersistentFlags().StringSlice("ruuvitags", nil, "RuuviTag names")
	rootCmd.PersistentFlags().Int("http.port", 8080, "HTTP server port")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}
}

func initConfig() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}
