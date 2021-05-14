// Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0

package tracer

import (
	"context"
	"errors"
	"io/ioutil"
	"strings"

	traceAPI "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/trace/zipkin"

	"go.opentelemetry.io/otel/api/global"

	"go.opentelemetry.io/otel/sdk/trace"

	stdLog "log"
)

// InitTracing initializes a trace provider
func InitTracing(uri, name string, prob float64) (*trace.Provider, error) {
	if len(strings.TrimSpace(uri)) == 0 {
		return nil, errors.New("zipkin uri is empty")
	}
	exporter, err := zipkin.NewExporter(
		uri,
		name,
		zipkin.WithLogger(stdLog.New(ioutil.Discard, "", stdLog.LstdFlags)),
	)
	if err != nil {
		return nil, err
	}
	tp, err := trace.NewProvider(
		trace.WithConfig(trace.Config{DefaultSampler: trace.ProbabilitySampler(prob)}),
		trace.WithBatcher(
			exporter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultBatchTimeout),
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
		),
	)
	if err != nil {
		return nil, err
	}
	return tp, nil
}

// GetTracer returns the generic tracer for the application
func GetTracer(ctx context.Context, spanName string) (context.Context, traceAPI.Span) {
	tr := global.TraceProvider().Tracer("karavi-topology")
	return tr.Start(ctx, spanName)
}
