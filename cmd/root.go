package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/niktheblak/ruuvitag-common/pkg/sensor"
	"github.com/niktheblak/ruuvitag-measurement-api/internal/server"
	"github.com/niktheblak/ruuvitag-measurement-api/pkg/ruuvitag"
	"github.com/niktheblak/web-common/pkg/auth"
	"github.com/niktheblak/web-common/pkg/graceful"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var DefaultColumns = sensor.DefaultColumnMap

const (
	postgresPortConfigKey = "postgres.port"
	serverPortConfigKey   = "server.port"
)

var (
	cfgFile  string
	logLevel string
	logger   *slog.Logger
)

var rootCmd = &cobra.Command{
	Use:               "ruuvitag-measurement-api",
	Short:             "REST API for reading current RuuviTag measurement values from PostgreSQL",
	SilenceUsage:      true,
	PersistentPreRunE: preRun,
	RunE:              run,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "config file path")
	rootCmd.Flags().StringVar(&logLevel, "loglevel", "", "log level")

	rootCmd.Flags().String("postgres.host", "", "host")
	rootCmd.Flags().Int(postgresPortConfigKey, 0, "port")
	rootCmd.Flags().String("postgres.username", "", "username")
	rootCmd.Flags().String("postgres.password", "", "username")
	rootCmd.Flags().String("postgres.database", "", "database name")
	rootCmd.Flags().String("postgres.table", "", "table name")
	rootCmd.Flags().String("postgres.name_table", "", "RuuviTag name table name")
	rootCmd.Flags().Int(serverPortConfigKey, 0, "Server port")
	rootCmd.Flags().StringSlice("server.token", nil, "Allowed API access tokens")
	rootCmd.Flags().StringToString("columns", nil, "columns to use")

	cobra.CheckErr(viper.BindPFlags(rootCmd.Flags()))

	viper.SetDefault("loglevel", "info")
	viper.SetDefault(postgresPortConfigKey, "5432")
	viper.SetDefault(serverPortConfigKey, 8080)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.ruuvitag-measurement-api")
		viper.AddConfigPath("/etc/ruuvitag-measurement-api")
		viper.SetConfigName("config")
	}
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Config file not found, using config from environment variables and arguments")
	}
}

func preRun(_ *cobra.Command, _ []string) error {
	level := new(slog.LevelVar)
	if err := level.UnmarshalText([]byte(viper.GetString("loglevel"))); err != nil {
		return err
	}
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	logger = slog.New(h)
	if viper.ConfigFileUsed() != "" {
		logger.LogAttrs(context.TODO(), slog.LevelInfo, "Using config file", slog.String("config", viper.ConfigFileUsed()))
	}
	logger.Info("Using log level", "level", level)
	return nil
}

func run(_ *cobra.Command, _ []string) error {
	var (
		accessToken   = viper.GetStringSlice("server.token")
		psqlHost      = viper.GetString("postgres.host")
		psqlPort      = viper.GetInt(postgresPortConfigKey)
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
		context.TODO(),
		slog.LevelInfo,
		"Connecting to PostgreSQL",
		slog.String("host", psqlHost),
		slog.Int("port", psqlPort),
		slog.String("database", psqlDatabase),
		slog.String("table", psqlTable),
		slog.String("name_table", psqlNameTable),
		slog.Any("columns", columns),
	)
	ctx := context.Background()
	svc, err := ruuvitag.New(ctx, ruuvitag.Config{
		ConnString: psqlInfo,
		Table:      psqlTable,
		NameTable:  psqlNameTable,
		Columns:    columns,
		Logger:     logger,
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
	httpServer := graceful.Shutdown{
		Server: &http.Server{
			Addr:    fmt.Sprintf(":%d", viper.GetInt(serverPortConfigKey)),
			Handler: server.New(svc, columns, authenticator, logger),
		},
		ShutdownTimeout: 5 * time.Second,
		Signals:         []os.Signal{os.Interrupt},
	}
	return errors.Join(httpServer.Serve(ctx), svc.Close())
}
