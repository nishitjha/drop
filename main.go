package main

import (
	"fmt"
	"os"

	"github.com/nishitjha/drop/cli"
	"github.com/nishitjha/drop/discovery"
	"github.com/nishitjha/drop/internal"
	"github.com/nishitjha/drop/internal/config"
)


func main() {
	err := config.Launch()
	if err != nil {
		fmt.Printf("%s Error reading config file: %v\n", internal.Icons.Negative, err)
		os.Exit(1)
	}
	discovery.Initialize()
	cli.Execute()
}
