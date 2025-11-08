package telemetry

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	tracer            trace.Tracer
	meter             otelmetric.Meter
	shutdownFuncs    []func(context.Context) error
	prometheusExporter *otelprometheus.Exporter
	prometheusRegistry *prometheus.Registry
)

const (
	serviceName = "syncer"
	version     = "0.1.0"
)

// Init initializes OpenTelemetry with support for multiple exporters
func Init(ctx context.Context) error {
	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", getEnv("OTEL_SERVICE_NAME", serviceName)),
			attribute.String("service.version", version),
		),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize tracer
	if err := initTracer(ctx, res); err != nil {
		return fmt.Errorf("failed to initialize tracer: %w", err)
	}

	// Initialize meter
	if err := initMeter(ctx, res); err != nil {
		return fmt.Errorf("failed to initialize meter: %w", err)
	}

	// Start runtime instrumentation
	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second)); err != nil {
		return fmt.Errorf("failed to start runtime instrumentation: %w", err)
	}

	return nil
}

func initTracer(ctx context.Context, res *resource.Resource) error {
	exporterType := getEnv("OTEL_TRACES_EXPORTER", "otlp")
	if exporterType == "none" {
		// Use noop tracer
		tracer = otel.Tracer(serviceName)
		return nil
	}

	var exporter sdktrace.SpanExporter
	var err error

	if exporterType == "otlp" || exporterType == "" {
		endpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
		protocol := getEnv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
		
		// Strip protocol prefix for gRPC (it expects just host:port)
		endpoint = stripProtocol(endpoint)
		
		if protocol == "grpc" {
			exporter, err = otlptracegrpc.New(ctx,
				otlptracegrpc.WithEndpoint(endpoint),
				otlptracegrpc.WithInsecure(),
			)
		} else {
			// HTTP protocol would go here if needed
			exporter, err = otlptracegrpc.New(ctx,
				otlptracegrpc.WithEndpoint(endpoint),
				otlptracegrpc.WithInsecure(),
			)
		}
		if err != nil {
			return fmt.Errorf("failed to create OTLP trace exporter: %w", err)
		}
	} else {
		// Default to noop if unknown exporter
		tracer = otel.Tracer(serviceName)
		return nil
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer = tp.Tracer(serviceName)

	shutdownFuncs = append(shutdownFuncs, func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	})

	return nil
}

func initMeter(ctx context.Context, res *resource.Resource) error {
	exporterType := getEnv("OTEL_METRICS_EXPORTER", "prometheus")
	
	var reader sdkmetric.Reader
	var err error

	if exporterType == "prometheus" {
		// Create Prometheus registry
		prometheusRegistry = prometheus.NewRegistry()
		
		// Create Prometheus exporter with the registry
		prometheusExporter, err = otelprometheus.New(
			otelprometheus.WithRegisterer(prometheusRegistry),
		)
		if err != nil {
			return fmt.Errorf("failed to create Prometheus exporter: %w", err)
		}
		reader = prometheusExporter
	} else if exporterType == "otlp" {
		endpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
		// Strip protocol prefix for gRPC (it expects just host:port)
		endpoint = stripProtocol(endpoint)
		exporter, err := otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(endpoint),
			otlpmetricgrpc.WithInsecure(),
		)
		if err != nil {
			return fmt.Errorf("failed to create OTLP metric exporter: %w", err)
		}
		reader = sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(10*time.Second))
	} else if exporterType == "none" {
		// Use noop meter
		meter = otel.Meter(serviceName)
		return nil
	} else {
		// Default to Prometheus
		prometheusRegistry = prometheus.NewRegistry()
		prometheusExporter, err = otelprometheus.New(
			otelprometheus.WithRegisterer(prometheusRegistry),
		)
		if err != nil {
			return fmt.Errorf("failed to create Prometheus exporter: %w", err)
		}
		reader = prometheusExporter
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)

	otel.SetMeterProvider(mp)
	meter = mp.Meter(serviceName)

	shutdownFuncs = append(shutdownFuncs, func(ctx context.Context) error {
		return mp.Shutdown(ctx)
	})

	return nil
}

// Shutdown gracefully shuts down all telemetry exporters
func Shutdown(ctx context.Context) error {
	var errs []error
	for _, fn := range shutdownFuncs {
		if err := fn(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}
	return nil
}

// Tracer returns the global tracer
func Tracer() trace.Tracer {
	if tracer == nil {
		return otel.Tracer(serviceName)
	}
	return tracer
}

// Meter returns the global meter
func Meter() otelmetric.Meter {
	if meter == nil {
		return otel.Meter(serviceName)
	}
	return meter
}

// GetPrometheusExporter returns the Prometheus exporter if it was initialized
func GetPrometheusExporter() *otelprometheus.Exporter {
	return prometheusExporter
}

// GetPrometheusRegistry returns the Prometheus registry if it was initialized
func GetPrometheusRegistry() *prometheus.Registry {
	return prometheusRegistry
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// stripProtocol removes http:// or https:// prefix from endpoint
// gRPC clients expect just host:port format
func stripProtocol(endpoint string) string {
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	return endpoint
}

// Helper function to create insecure gRPC connection
func insecureGRPCConn(endpoint string) (*grpc.ClientConn, error) {
	return grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

