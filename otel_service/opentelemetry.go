package otel_service

import (
	"context"
	"fmt"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"http-proxy/config"
	"http-proxy/proxy_connection"
	"log"
	"os"
)

const (
	instrumentationName = "http-proxy-tracer"
)

var Tracer = otel.Tracer(instrumentationName)

var Meter = otel.Meter("http-proxy-meter")

func initCounters() {
	_, err := Meter.Int64ObservableGauge("http.proxy.connections.new",
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			data, err := proxy_connection.MeasureEstablishedConnections()
			Info(ctx, Logger, "http.proxy.connections.new:", zap.Int("connections", data))
			if err != nil {
				return err
			}
			observer.Observe(int64(data))
			return nil
		}),
		metric.WithDescription("Current http proxy connections"))
	if err != nil {
		log.Fatal("error creating Int64ObservableGauge: ", err)
	}
}

func InitOpenTelemetry() {

	if _, exits := os.LookupEnv("UPTRACE_DSN"); !exits {
		fmt.Println("warn: UPTRACE_DSN not set")
		os.Exit(1)
	}
	environment := "development"
	if config.Production {
		environment = "production"
	}

	uptrace.ConfigureOpentelemetry(
		// copy your project DSN here or use UPTRACE_DSN env var
		uptrace.WithServiceName("http-proxy"),
		uptrace.WithServiceVersion("v1.0.0"),
		uptrace.WithDeploymentEnvironment(environment),
		uptrace.WithResourceDetectors(),
	)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		fmt.Println("otel - error: ", err)
	}))
	initCounters()
}
