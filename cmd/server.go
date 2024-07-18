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

	"github.com/niktheblak/ruuvitag-common/pkg/sensor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/niktheblak/web-common/pkg/auth"

	"github.com/niktheblak/ruuvitag-measurement-api/internal/server"
	"github.com/niktheblak/ruuvitag-measurement-api/pkg/measurement"
)

var DefaultColumns = sensor.DefaultColumnMap

var serverCmd = &cobra.Command{
	Use:          "server",
	Short:        "Start temperature API server",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			accessToken   = viper.GetStringSlice("server.token")
			psqlHost      = viper.GetString("postgres.host")
			psqlPort      = viper.GetInt("postgres.port")
			psqlUsername  = viper.GetString("postgres.username")
			psqlPassword  = viper.GetString("postgres.password")
			psqlDatabase  = viper.GetString("postgres.database")
			psqlTable     = viper.GetString("postgres.table")
			psqlNameTable = viper.GetString("postgres.name_table")
			columns       = viper.GetStringMapString("columns")
		)
		if len(columns) == 0 {
			columns = DefaultColumns
		}
		psqlInfo := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			psqlHost,
			psqlPort,
			psqlUsername,
			psqlPassword,
			psqlDatabase,
		)
		logger.LogAttrs(
			nil,
			slog.LevelInfo,
			"Connecting to TimescaleDB",
			slog.String("host", psqlHost),
			slog.Int("port", psqlPort),
			slog.String("database", psqlDatabase),
			slog.String("table", psqlTable),
			slog.String("name_table", psqlNameTable),
			slog.Any("columns", columns),
		)
		svc, err := measurement.New(measurement.Config{
			PsqlInfo:  psqlInfo,
			Table:     psqlTable,
			NameTable: psqlNameTable,
			Columns:   columns,
			Logger:    logger,
		})
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
			Handler: server.New(svc, columns, authenticator, logger),
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
	serverCmd.Flags().String("postgres.host", "", "host")
	serverCmd.Flags().Int("postgres.port", 0, "port")
	serverCmd.Flags().String("postgres.username", "", "username")
	serverCmd.Flags().String("postgres.password", "", "username")
	serverCmd.Flags().String("postgres.database", "", "database name")
	serverCmd.Flags().String("postgres.table", "", "table name")
	serverCmd.Flags().String("postgres.name_table", "", "RuuviTag name table name")
	serverCmd.Flags().Int("server.port", 0, "Server port")
	serverCmd.Flags().StringSlice("server.token", nil, "Allowed API access tokens")
	serverCmd.Flags().StringToString("columns", nil, "columns to use")

	cobra.CheckErr(viper.BindPFlags(serverCmd.Flags()))

	viper.SetDefault("postgres.port", "5432")
	viper.SetDefault("server.port", 8080)

	rootCmd.AddCommand(serverCmd)
}
