/*
 Copyright (c) 2021-2023 Dell Inc. or its subsidiaries. All Rights Reserved.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package tracer

import (
	"context"
	"errors"
	"io"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	stdLog "log"
)

// InitTracing initializes a trace provider
func InitTracing(uri string, prob float64) (trace.TracerProvider, error) {
	if len(strings.TrimSpace(uri)) == 0 {
		return nil, errors.New("zipkin uri is empty")
	}
	exporter, err := zipkin.New(
		uri,
		zipkin.WithLogger(stdLog.New(io.Discard, "", stdLog.LstdFlags)),
	)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(prob)),
		sdktrace.WithBatcher(
			exporter,
			sdktrace.WithMaxExportBatchSize(sdktrace.DefaultMaxExportBatchSize),
			sdktrace.WithBatchTimeout(sdktrace.DefaultExportTimeout),
			sdktrace.WithMaxExportBatchSize(sdktrace.DefaultMaxExportBatchSize),
		),
	)
	if err != nil {
		return nil, err
	}
	return tp, nil
}

// GetTracer returns the generic tracer for the application
func GetTracer(ctx context.Context, spanName string) (context.Context, trace.Span) {
	tr := otel.GetTracerProvider()
	return tr.Tracer("karavi-topology").Start(ctx, spanName)
}
