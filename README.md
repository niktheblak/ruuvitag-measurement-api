# temperature-api
API for reading current RuuviTag temperatures from InfluxDB

## Usage

Generate an access token for InfluxDB 2.x using the InfluxDB tools.
Provide InfluxDB address and credentials via environemnt variables and run
the `main.go` executable.

```shell
export INFLUXDB_ADDR=http://influxdb:8086 # this can be a remote server address as well
export INFLUXDB_ORG=myorg
export INFLUXDB_TOKEN=token_from_influxdb
export INFLUXDB_BUCKET=mybucket
export INFLUXDB_MEASUREMENT=mymeasurement
export HTTP_PORT=8080

go run main.go
```
