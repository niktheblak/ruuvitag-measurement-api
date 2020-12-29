#!/usr/bin/env bash

docker run \
  -d \
  --rm \
  --name influxdb \
  --network ruuvitag \
  -p 8086:8086 \
  -v "$PWD/test/influxdb/config/influxdb.conf:/etc/influxdb/influxdb.conf:ro" \
  -v "$PWD/test/influxdb/data:/var/lib/influxdb" \
  -e INFLUXDB_DB=ruuvitag \
  -e INFLUXDB_ADMIN_USER=admin -e INFLUXDB_ADMIN_PASSWORD=admin \
  -e INFLUXDB_USER=api -e INFLUXDB_USER_PASSWORD=api \
  influxdb:latest \
  -config /etc/influxdb/influxdb.conf
