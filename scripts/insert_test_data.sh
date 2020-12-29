#!/usr/bin/env bash

set -e

TS=$(date +%s)

curl 'http://localhost:8086/api/v2/write?bucket=ruuvitag&precision=s' \
  --header 'Authorization: Token api:api' \
  --data-binary "ruuvi,name=Mancave,mac=CC:CA:7E:52:CC:34 temperature=23.1 $TS
ruuvi,name=Mancave,mac=CC:CA:7E:52:CC:34 humidity=45.0 $TS
ruuvi,name=Mancave,mac=CC:CA:7E:52:CC:34 pressure=999.0 $TS
ruuvi,name=Garage,mac=FB:E1:B7:04:95:EE temperature=12.5 $TS
ruuvi,name=Garage,mac=FB:E1:B7:04:95:EE humidity=61.0 $TS
ruuvi,name=Garage,mac=FB:E1:B7:04:95:EE pressure=998.0 $TS"
