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
	"strconv"
	"strings"

	"github.com/dell/karavi-topology/internal/entrypoint"
	"github.com/dell/karavi-topology/internal/k8s"
	"github.com/dell/karavi-topology/internal/service"
)

const (
	port            = "443"
	defaultCertFile = "/certs/localhost.crt"
	defaultKeyFile  = "/certs/localhost.key"
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

	certFile := os.Getenv("TLS_CERT_PATH")
	if len(strings.TrimSpace(certFile)) < 1 {
		certFile = defaultCertFile
	}

	keyFile := os.Getenv("TLS_KEY_PATH")
	if len(strings.TrimSpace(keyFile)) < 1 {
		keyFile = defaultKeyFile
	}

	var bindPort int
	portEnv := os.Getenv("PORT")
	if portEnv != "" {
		var err error
		if bindPort, err = strconv.Atoi(portEnv); err != nil {
			fmt.Fprintf(os.Stderr, "PORT value is valid '%s'", portEnv)
			os.Exit(1)
		}
	}

	svc := &service.Service{
		VolumeFinder: volumeFinder,
		CertFile:     certFile,
		KeyFile:      keyFile,
		Port:         bindPort,
	}

	if err := entrypoint.Run(context.Background(), svc); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
