package tracer_test

import (
	"context"
	"testing"

	tracer "github.com/dell/karavi-topology/internal/tracers"
)

func TestInitTracing(t *testing.T) {
	// Test case: Empty URI
	_, err := tracer.InitTracing("", 0.5)
	if err == nil {
		t.Errorf("Expected error for empty URI, got nil")
	}

	// Test case: Invalid URI
	_, err = tracer.InitTracing("invalid_uri", 0.5)
	if err == nil {
		t.Errorf("Expected error for invalid URI, got nil")
	}

	// Test case: Valid URI
	uri := "http://localhost:9411/api/v2/spans"
	tp, err := tracer.InitTracing(uri, 0.5)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if tp == nil {
		t.Errorf("Expected non-nil TracerProvider, got nil")
	}
}

func TestGetTracer(t *testing.T) {
	// Given a context and a span name
	ctx := context.Background()
	spanName := "test-span"

	// When calling GetTracer
	ctx, span := tracer.GetTracer(ctx, spanName)

	// Then the returned context should not be nil
	if ctx == nil {
		t.Errorf("Expected non-nil context, got nil")
	}

	// And the returned span should not be nil
	if span == nil {
		t.Errorf("Expected non-nil span, got nil")
	}

}
