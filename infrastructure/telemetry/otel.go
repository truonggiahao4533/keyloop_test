package telemetry

import (
	"context"
	"errors"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otlploghttp "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	prometheusexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func SetupOTel(ctx context.Context) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
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

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName("keyloop-api")))
	if err != nil {
		handleErr(err)
		return
	}

	// Export traces to console for debugging
	// traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	// if err != nil {
	// 	handleErr(err)
	// 	return
	// }

	// Export traces to OTLP collector
	// For production, spin up an OTLP collector (Jaeger, Tempo, etc.)
	traceExporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		log.Printf("⚠️  tracing disabled: %v", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExporter), sdktrace.WithResource(res))
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	metricExporter, err := prometheusexporter.New()
	if err != nil {
		handleErr(err)
		return
	}
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(metricExporter), sdkmetric.WithResource(res))
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	// OTEL_EXPORTER_OTLP_LOGS_ENDPOINT env var is read automatically (e.g. http://loki:3100/otlp/v1/logs)
	logExporter, err := otlploghttp.New(ctx)
	if err != nil {
		log.Printf("⚠️  log export disabled: %v", err)
	}
	var loggerProvider *sdklog.LoggerProvider
	if logExporter != nil {
		loggerProvider = sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)), sdklog.WithResource(res))
	} else {
		loggerProvider = sdklog.NewLoggerProvider(sdklog.WithResource(res))
	}
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)

	return
}
