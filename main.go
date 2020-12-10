package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/niktheblak/temperature-api/internal/server"
	"github.com/niktheblak/temperature-api/pkg/measurement"
)

func main() {
	addr := flag.String("addr", "http://127.0.0.1:8086", "InfluxDB address")
	username := flag.String("username", "", "InfluxDB username")
	password := flag.String("password", "", "InfluxDB password")
	db := flag.String("database", "ruuvitag", "InfluxDB database")
	meas := flag.String("measurement", "ruuvitag", "InfluxDB measurement")
	port := flag.Int("port", 8080, "HTTP server port")
	flag.Parse()
	cfg := measurement.Config{
		Addr:        *addr,
		Username:    *username,
		Password:    *password,
		Database:    *db,
		Measurement: *meas,
		Timeout:     10 * time.Second,
	}
	svc, err := measurement.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := svc.Ping(); err != nil {
		log.Fatal(err)
	}
	defer svc.Close()
	srv := &server.Server{
		Service: svc,
	}
	srv.Routes()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
