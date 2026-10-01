package main

import (
	"os"

	gocbusruntime "github.com/MickMake/GoCBus/internal/runtime"
)

func main() {
	os.Exit(gocbusruntime.Run(os.Args[1:], os.Stdout, os.Stderr))
}
