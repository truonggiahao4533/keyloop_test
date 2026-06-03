package port

import "context"

// SpanAttr is a typed key-value pair attached to a span at creation time.
type SpanAttr struct {
	Key   string
	Value any
}

func StringAttr(key, val string) SpanAttr  { return SpanAttr{key, val} }
func IntAttr(key string, val int) SpanAttr { return SpanAttr{key, val} }

// Span represents an active trace span.
type Span interface {
	End()
	RecordError(err error)
	SetErrorStatus(msg string)
}

// Tracer creates spans. Implementations live in infrastructure/telemetry.
type Tracer interface {
	Start(ctx context.Context, spanName string, attrs ...SpanAttr) (context.Context, Span)
}
