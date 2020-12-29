#!/usr/bin/env bash

docker run \
  -it \
  --rm \
  --network ruuvitag \
  influxdb:latest \
  influx \
  -host influxdb \
  -database ruuvitag \
  -username admin \
  -password admin
