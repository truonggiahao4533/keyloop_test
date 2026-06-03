package telemetry

import (
	"context"

	"keyloop-test/internal/port"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type otelTracer struct {
	t oteltrace.Tracer
}

// NewTracer returns a port.Tracer backed by the global OTel tracer provider.
func NewTracer(instrumentationName string) port.Tracer {
	return &otelTracer{t: otel.Tracer(instrumentationName)}
}

func (tr *otelTracer) Start(ctx context.Context, spanName string, attrs ...port.SpanAttr) (context.Context, port.Span) {
	kvs := make([]attribute.KeyValue, 0, len(attrs))
	for _, a := range attrs {
		switch v := a.Value.(type) {
		case string:
			kvs = append(kvs, attribute.String(a.Key, v))
		case int:
			kvs = append(kvs, attribute.Int(a.Key, v))
		case bool:
			kvs = append(kvs, attribute.Bool(a.Key, v))
		case float64:
			kvs = append(kvs, attribute.Float64(a.Key, v))
		}
	}
	ctx, span := tr.t.Start(ctx, spanName, oteltrace.WithAttributes(kvs...))
	return ctx, &otelSpan{s: span}
}

type otelSpan struct {
	s oteltrace.Span
}

func (s *otelSpan) End()                      { s.s.End() }
func (s *otelSpan) RecordError(err error)      { s.s.RecordError(err) }
func (s *otelSpan) SetErrorStatus(msg string)  { s.s.SetStatus(codes.Error, msg) }
