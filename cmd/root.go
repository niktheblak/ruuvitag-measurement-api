package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/niktheblak/temperature-api/internal/server"
	"github.com/niktheblak/temperature-api/pkg/measurement"
)

var (
	client influxdb.Client
	svc    measurement.Service
)

var rootCmd = &cobra.Command{
	Use:          "temperature-api",
	Short:        "REST API for getting current temperatures",
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		svc, err = measurement.New(measurement.Config{
			Addr:        viper.GetString("influxdb.addr"),
			Username:    viper.GetString("influxdb.username"),
			Password:    viper.GetString("influxdb.password"),
			Database:    viper.GetString("influxdb.database"),
			Measurement: viper.GetString("influxdb.measurement"),
			Timeout:     10 * time.Second,
		})
		if err != nil {
			return
		}
		err = svc.Ping()
		return
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if client != nil {
			client.Close()
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		srv := server.New(svc)
		router := httprouter.New()
		router.GET("/", srv.Current)
		port := viper.GetInt("http.port")
		return http.ListenAndServe(fmt.Sprintf(":%d", port), router)
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
