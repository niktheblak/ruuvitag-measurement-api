package main

import (
	"os"

	"github.com/niktheblak/ruuvitag-measurement-api/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
