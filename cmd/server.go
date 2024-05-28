package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/niktheblak/temperature-api/internal/server"
	"github.com/niktheblak/temperature-api/pkg/auth"
	"github.com/niktheblak/temperature-api/pkg/measurement"
)

var serverCmd = &cobra.Command{
	Use:          "server",
	Short:        "Start temperature API server",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			addr        = viper.GetString("influxdb.addr")
			org         = viper.GetString("influxdb.org")
			token       = viper.GetString("influxdb.token")
			bucket      = viper.GetString("influxdb.bucket")
			meas        = viper.GetString("influxdb.measurement")
			port        = viper.GetInt("server.port")
			accessToken = viper.GetStringSlice("server.token")
		)
		cfg := measurement.Config{
			Addr:        addr,
			Org:         org,
			Token:       token,
			Bucket:      bucket,
			Measurement: meas,
			Timeout:     10 * time.Second,
		}
		logger.LogAttrs(nil, slog.LevelInfo, "Connecting to InfluxDB", slog.String("addr", addr), slog.String("bucket", bucket), slog.String("org", org))
		svc, err := measurement.New(cfg)
		if err != nil {
			return err
		}
		var authenticator auth.Authenticator
		if len(accessToken) > 0 {
			logger.Info("Using authentication", "tokens", len(accessToken))
			authenticator = auth.Static(accessToken...)
		} else {
			logger.Info("Not using authentication")
			authenticator = auth.AlwaysAllow()
		}
		httpServer := &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: server.New(svc, authenticator, logger),
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		go func() {
			logger.LogAttrs(nil, slog.LevelInfo, "Starting server", slog.Int("port", port))
			if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("Failed to start HTTP server", "err", err)
			}
		}()
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ctx.Done()
			logger.Info("Shutting down service")
			shutdownCtx := context.Background()
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				logger.Error("Failed to shut down HTTP server", "err", err)
			}
			if err := svc.Close(); err != nil {
				logger.Error("Failed to shut down service", "err", err)
			}
		}()
		wg.Wait()
		return nil
	},
}

func init() {
	serverCmd.Flags().String("influxdb.addr", "", "InfluxDB server address")
	serverCmd.Flags().String("influxdb.org", "", "InfluxDB organization")
	serverCmd.Flags().String("influxdb.token", "", "InfluxDB token")
	serverCmd.Flags().String("influxdb.bucket", "", "InfluxDB bucket")
	serverCmd.Flags().String("influxdb.measurement", "", "InfluxDB measurement")
	serverCmd.Flags().Int("server.port", 0, "Server port")
	serverCmd.Flags().StringSlice("server.token", nil, "Allowed API access tokens")

	cobra.CheckErr(viper.BindPFlags(serverCmd.Flags()))

	viper.SetDefault("influxdb.addr", "http://127.0.0.1:8086")
	viper.SetDefault("influxdb.bucket", "RuuviTag")
	viper.SetDefault("influxdb.measurement", "ruuvitag")
	viper.SetDefault("server.port", 8080)

	rootCmd.AddCommand(serverCmd)
}
