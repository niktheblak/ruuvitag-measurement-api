#!/usr/bin/env bash

docker run \
  -it \
  --rm \
  -e INFLUXDB_DB=ruuvitag \
  -e INFLUXDB_ADMIN_USER=admin -e INFLUXDB_ADMIN_PASSWORD=admin \
  -e INFLUXDB_USER=api -e INFLUXDB_USER_PASSWORD=api \
  -v "$PWD/influxdb/data:/var/lib/influxdb" \
  influxdb /init-influxdb.sh
