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
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/dell/karavi-topology/internal/entrypoint"
	"github.com/dell/karavi-topology/internal/k8s"
	"github.com/dell/karavi-topology/internal/service"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const (
	port              = "443"
	defaultCertFile   = "/certs/localhost.crt"
	defaultKeyFile    = "/certs/localhost.key"
	defaultConfigFile = "/etc/config/karavi-topology.yaml"
)

func main() {

	// enable viper to get properties from environment variables or default configuration file
	viper.AutomaticEnv()
	viper.SetConfigFile(defaultConfigFile)

	err := viper.ReadInConfig()
	// if unable to read configuration file, proceed in case we use environment variables
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to read config file: %v", err)
	}

	volumeFinder := &k8s.VolumeFinder{
		API: &k8s.API{},
	}

	updateDriverNames := func(volumeFinder *k8s.VolumeFinder) {
		provisionerNamesValue := viper.GetString("PROVISIONER_NAMES")
		provisionerNames := strings.Split(provisionerNamesValue, ",")
		volumeFinder.DriverNames = provisionerNames
		log.Printf("Set DriverNames to %s", provisionerNames)
	}

	updateDriverNames(volumeFinder)

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("Config file changed: %v", e)
		updateDriverNames(volumeFinder)
	})

	// TLS_CERT_PATH is only read as an environment variable
	certFile := viper.GetString("TLS_CERT_PATH")
	if len(strings.TrimSpace(certFile)) < 1 {
		certFile = defaultCertFile
	}

	// TLS_KEY_PATH is only read as an environment variable
	keyFile := viper.GetString("TLS_KEY_PATH")
	if len(strings.TrimSpace(keyFile)) < 1 {
		keyFile = defaultKeyFile
	}

	var bindPort int
	// PORT is only read as an environment variable
	portEnv := viper.GetString("PORT")
	if portEnv != "" {
		var err error
		if bindPort, err = strconv.Atoi(portEnv); err != nil {
			fmt.Fprintf(os.Stderr, "PORT value is invalid: '%s'", portEnv)
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
