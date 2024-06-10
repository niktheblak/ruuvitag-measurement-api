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
		psqlInfo := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			viper.GetString("timescaledb.host"),
			viper.GetInt("timescaledb.port"),
			viper.GetString("timescaledb.username"),
			viper.GetString("timescaledb.password"),
			viper.GetString("timescaledb.database"),
		)
		accessToken := viper.GetStringSlice("server.token")
		logger.LogAttrs(
			nil,
			slog.LevelInfo,
			"Connecting to TimescaleDB",
			slog.String("host", viper.GetString("timescaledb.host")),
			slog.Int("port", viper.GetInt("timescaledb.port")),
			slog.String("database", viper.GetString("timescaledb.database")),
			slog.String("table", viper.GetString("timescaledb.table")),
		)
		svc, err := measurement.New(psqlInfo, viper.GetString("timescaledb.table"))
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
			Addr:    fmt.Sprintf(":%d", viper.GetInt("server.port")),
			Handler: server.New(svc, authenticator, logger),
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		go func() {
			logger.LogAttrs(nil, slog.LevelInfo, "Starting server", slog.Int("port", viper.GetInt("server.port")))
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
	serverCmd.Flags().Bool("timescaledb.enabled", false, "Store measurements to TimescaleDB")
	serverCmd.Flags().String("timescaledb.host", "", "TimescaleDB host")
	serverCmd.Flags().Int("timescaledb.port", 0, "TimescaleDB port")
	serverCmd.Flags().String("timescaledb.username", "", "TimescaleDB username")
	serverCmd.Flags().String("timescaledb.password", "", "TimescaleDB username")
	serverCmd.Flags().String("timescaledb.database", "", "TimescaleDB database")
	serverCmd.Flags().String("timescaledb.table", "", "TimescaleDB table")
	serverCmd.Flags().Int("server.port", 0, "Server port")
	serverCmd.Flags().StringSlice("server.token", nil, "Allowed API access tokens")

	cobra.CheckErr(viper.BindPFlags(serverCmd.Flags()))

	viper.SetDefault("timescaledb.port", "5432")
	viper.SetDefault("server.port", 8080)

	rootCmd.AddCommand(serverCmd)
}
