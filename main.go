package main

import (
	"context"
	"github.com/uptrace/uptrace-go/uptrace"
	"http-proxy/config"
	"http-proxy/otel_service"
	"http-proxy/proxy"
)

func main() {
	ctx := context.Background()
	config.InitConfig()

	otel_service.InitLogger()

	otel_service.InitOpenTelemetry()
	defer uptrace.Shutdown(ctx)
	defer otel_service.Logger.Sync()

	proxy.SetupProxy(ctx)
}
