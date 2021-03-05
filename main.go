package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/niktheblak/temperature-api/internal/server"
	"github.com/niktheblak/temperature-api/pkg/measurement"
)

func main() {
	addr := os.Getenv("INFLUXDB_ADDR")
	if addr == "" {
		addr = "http://127.0.0.1:8086"
	}
	org := os.Getenv("INFLUXDB_ORG")
	token := os.Getenv("INFLUXDB_TOKEN")
	bucket := os.Getenv("INFLUXDB_BUCKET")
	meas := os.Getenv("INFLUXDB_MEASUREMENT")
	port, _ := strconv.Atoi(os.Getenv("HTTP_PORT"))
	if port <= 0 || port > 65536 {
		port = 8080
	}
	cfg := measurement.Config{
		Addr:        addr,
		Org:         org,
		Token:       token,
		Bucket:      bucket,
		Measurement: meas,
		Timeout:     10 * time.Second,
	}
	svc, err := measurement.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := svc.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	cancel()
	defer svc.Close()
	srv := server.New(svc)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), srv))
}
