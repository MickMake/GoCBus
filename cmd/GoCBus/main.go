package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	gocbusruntime "github.com/MickMake/GoCBus/internal/runtime"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := gocbusruntime.RunContext(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
