/*
 *
 * Copyright © 2021-2024 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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
