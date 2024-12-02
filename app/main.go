package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const name = "github.com/taxintt/opentelemetry-mackerel-playgrounds"

var (
	meter  = otel.Meter(name)
	tracer = otel.Tracer(name)

	apiCallCounter metric.Int64Counter

	shutdownFuncs []func(context.Context) error
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func init() {
	var err error
	apiCallCounter, err = meter.Int64Counter(
		"api.call.counter",
		metric.WithDescription("Number of API calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		panic(err)
	}
}

func run() (err error) {
	// create echo instance
	e := echo.New()
	ctx := context.Background()

	// error handler
	shutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	defer func() {
		err = errors.Join(err, shutdown(context.Background()))
	}()

	// create counter
	err = newMetricProvider(ctx, handleErr)
	if err != nil {
		handleErr(err)
		return
	}

	// create tracer
	err = newTraceProvider(ctx, handleErr)
	if err != nil {
		handleErr(err)
		return
	}

	// create middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/hello", func(c echo.Context) error {
		// create span
		ctx, span := tracer.Start(context.Background(), "hello")
		defer span.End()

		time.Sleep(1000 * time.Millisecond)

		// increment counter
		apiCallCounter.Add(ctx, 1)
		return c.String(http.StatusOK, "Hello, World!")
	})

	// start server
	e.Logger.Fatal(e.Start(":" + os.Getenv("ENV_PORT")))

	// graceful shutdown
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt)
	<-ctx.Done()

	return
}

func newMetricProvider(ctx context.Context, handleErr func(err error)) error {
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint("localhost:4317"),
	)
	if err != nil {
		handleErr(err)
		return err
	}

	serviceName := getenv("SERVICE_NAME", "sample-app")
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		handleErr(err)
		return err
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(time.Second)),
		),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(provider)
	shutdownFuncs = append(shutdownFuncs, provider.Shutdown)

	return nil
}

func newTraceProvider(ctx context.Context, handleErr func(err error)) error {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint("localhost:4317"),
	)
	if err != nil {
		handleErr(err)
		return err
	}

	serviceName := getenv("SERVICE_NAME", "sample-app")
	res, err := resource.New(ctx,
		// Use the GCP resource detector to detect information about the GCP platform
		resource.WithDetectors(gcp.NewDetector()),
		// Keep the default detectors
		resource.WithTelemetrySDK(),
		// Add your own custom attributes to identify your application
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		handleErr(err)
		return err
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	shutdownFuncs = append(shutdownFuncs, provider.Shutdown)

	return nil
}

func getenv(key, fallback string) string {
    value := os.Getenv(key)
    if len(value) == 0 {
        return fallback
    }
    return value
}