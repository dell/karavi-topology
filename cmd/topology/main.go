package main

// Copyright (c) 2020 Dell Inc., or its subsidiaries. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dell/karavi-topology/internal/entrypoint"
	"github.com/dell/karavi-topology/internal/k8s"
	"github.com/dell/karavi-topology/internal/service"
)

const (
	port = "8080"
)

func main() {

	provisionerNamesValue := os.Getenv("PROVISIONER_NAMES")
	if provisionerNamesValue == "" {
		fmt.Fprintf(os.Stderr, "PROVISIONER_NAMES is required")
		os.Exit(1)
	}

	provisionerNames := strings.Split(provisionerNamesValue, ",")

	volumeFinder := k8s.VolumeFinder{
		API:         &k8s.API{},
		DriverNames: provisionerNames,
	}

	svc := &service.Service{
		VolumeFinder: volumeFinder,
	}

	if err := entrypoint.Run(context.Background(), svc); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
