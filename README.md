# ruuvitag-measurement-api
API for reading current RuuviTag temperatures from PostgreSQL (or TimescaleDB).

## Usage

Create a config file (TOML or YAML) with your PostgreSQL credentials:

```toml
[postgres]
host = "my-postgres-instance.cloud"
port = 5432
database = "ruuvitag"
username = "measurement_api"
password = "..."
table = "ruuvitag"
name_table = "ruuvitag_names"

[server]
port = 8180
token = ["..."]
```

Then run the server:

```shell
go run main.go server --config config.toml
```
