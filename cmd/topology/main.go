/*
 Copyright (c) 2020-2022 Dell Inc. or its subsidiaries. All Rights Reserved.

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

package main

import (
	"context"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	"github.com/dell/karavi-topology/internal/entrypoint"
	"github.com/dell/karavi-topology/internal/k8s"
	"github.com/dell/karavi-topology/internal/service"
	tracer "github.com/dell/karavi-topology/internal/tracers"
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
	logger := logrus.New()

	// enable viper to get properties from environment variables or default configuration file
	viper.AutomaticEnv()
	viper.SetConfigFile(defaultConfigFile)

	err := viper.ReadInConfig()
	// if unable to read configuration file, proceed in case we use environment variables
	if err != nil {
		logger.WithError(err).Error("unable to read config file")
	}

	volumeFinder := &k8s.VolumeFinder{
		API:    &k8s.API{},
		Logger: logger,
	}

	updateDriverNames := func(volumeFinder *k8s.VolumeFinder) {
		provisionerNamesValue := viper.GetString("PROVISIONER_NAMES")
		provisionerNames := strings.Split(provisionerNamesValue, ",")
		volumeFinder.DriverNames = provisionerNames
		logger.WithField("driver_names", provisionerNames).Info("setting driver names")
	}

	updateLoggingSettings := func(logger *logrus.Logger) {
		logFormat := viper.GetString("LOG_FORMAT")
		if strings.EqualFold(logFormat, "json") {
			logger.SetFormatter(&logrus.JSONFormatter{})
		} else {
			// use text formatter by default
			logger.SetFormatter(&logrus.TextFormatter{})
		}
		logLevel := viper.GetString("LOG_LEVEL")
		level, err := logrus.ParseLevel(logLevel)
		if err != nil {
			// use INFO level by default
			level = logrus.InfoLevel
		}
		logger.SetLevel(level)
	}

	updateLoggingSettings(logger)
	updateDriverNames(volumeFinder)
	updateTracing(logger)

	viper.WatchConfig()
	viper.OnConfigChange(func(_ fsnotify.Event) {
		logger.WithField("file", defaultConfigFile).Info("configuration file changed")
		updateDriverNames(volumeFinder)
		updateLoggingSettings(logger)
		updateTracing(logger)
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
			logger.WithError(err).WithField("port", portEnv).Fatal("port value is invalid")
		}
	}

	var enableDebug bool
	// DEBUG is only read as an environment variable
	debugEnv := viper.GetString("DEBUG")
	if debugEnv != "" {
		var err error
		if enableDebug, err = strconv.ParseBool(debugEnv); err != nil {
			logger.WithError(err).WithField("debug", debugEnv).Fatal("debug value is invalid")
		}
	}

	svc := &service.Service{
		VolumeFinder: volumeFinder,
		CertFile:     certFile,
		KeyFile:      keyFile,
		Port:         bindPort,
		Logger:       logger,
		EnableDebug:  enableDebug,
	}

	if err := entrypoint.Run(context.Background(), svc); err != nil {
		logger.WithError(err).Fatal("running service")
	}
}

func updateTracing(logger *logrus.Logger) {
	zipkinURI := viper.GetString("ZIPKIN_URI")
	zipkinServiceName := viper.GetString("ZIPKIN_SERVICE_NAME")
	zipkinProbability := viper.GetFloat64("ZIPKIN_PROBABILITY")

	tp, err := tracer.InitTracing(zipkinURI, zipkinProbability)
	if err != nil {
		logger.WithError(err).Error("initializing tracer")
		return
	}

	logger.WithFields(logrus.Fields{
		"uri":          zipkinURI,
		"service_name": zipkinServiceName,
		"probablity":   zipkinProbability,
	}).Infof("setting zipkin tracing")
	otel.SetTracerProvider(tp)
}
