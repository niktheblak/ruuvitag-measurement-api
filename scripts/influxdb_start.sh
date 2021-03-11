#!/usr/bin/env bash

docker run \
  --rm \
  --name influxdb \
  --network ruuvitag \
  -p 8086:8086 \
  -v "$PWD/test/influxdb/data:/var/lib/influxdb2" \
  -e DOCKER_INFLUXDB_INIT_MODE=setup \
  -e DOCKER_INFLUXDB_INIT_USERNAME=admin \
  -e DOCKER_INFLUXDB_INIT_PASSWORD=DockerInfluxDBAdminPassword \
  -e DOCKER_INFLUXDB_INIT_ORG=ruuvitag \
  -e DOCKER_INFLUXDB_INIT_BUCKET=ruuvitag \
  -e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=DockerInfluxDBAdminToken \
  influxdb:2.0
