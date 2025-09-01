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

	"github.com/dell/karavi-topology/internal/entrypoint"
	"github.com/dell/karavi-topology/internal/k8s"
	"github.com/dell/karavi-topology/internal/service"
	tracer "github.com/dell/karavi-topology/internal/tracers"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
)

const (
	defaultConfigFile = "/etc/config/karavi-topology.yaml"
	defaultCertFile   = "/certs/localhost.crt"
	defaultKeyFile    = "/certs/localhost.key"
)

type ServiceConfig struct {
	CertFile     string
	KeyFile      string
	Port         int
	EnableDebug  bool
	VolumeFinder *k8s.VolumeFinder
}

func main() {
	mainWithEntrypoint(entrypoint.Run)
}

func mainWithEntrypoint(entrypointRun func(ctx context.Context, service entrypoint.ServiceRunner) error) {
	logger := configureLogger()
	setupViper(logger)
	config := initializeServiceConfig(logger)
	setupConfigWatchers(logger, config)
	initializeTracing(logger)

	if err := entrypointRun(context.Background(), createService(config, logger)); err != nil {
		logger.WithError(err).Fatal("Service startup failed")
	}
}

func configureLogger() *logrus.Logger {
	logger := logrus.New()
	updateLogSettings(logger)
	return logger
}

func updateLogSettings(logger *logrus.Logger) {
	if strings.EqualFold(viper.GetString("LOG_FORMAT"), "json") {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{})
	}

	if level, err := logrus.ParseLevel(viper.GetString("LOG_LEVEL")); err == nil {
		logger.SetLevel(level)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
}

func setupViper(logger *logrus.Logger) {
	viper.AutomaticEnv()
	viper.SetConfigFile(defaultConfigFile)
	if err := viper.ReadInConfig(); err != nil {
		logger.WithError(err).Warn("Config file not found; using environment variables only")
	}
}

func initializeServiceConfig(logger *logrus.Logger) *ServiceConfig {
	return &ServiceConfig{
		CertFile:     getEnvWithDefault("TLS_CERT_PATH", defaultCertFile),
		KeyFile:      getEnvWithDefault("TLS_KEY_PATH", defaultKeyFile),
		Port:         parsePort(logger),
		EnableDebug:  parseDebugFlag(logger),
		VolumeFinder: createVolumeFinder(logger),
	}
}

func createVolumeFinder(logger *logrus.Logger) *k8s.VolumeFinder {
	vf := &k8s.VolumeFinder{
		API:    &k8s.API{},
		Logger: logger,
	}
	vf.DriverNames = parseDriverNames(logger)
	return vf
}

func parseDriverNames(logger *logrus.Logger) []string {
	names := strings.TrimSpace(viper.GetString("PROVISIONER_NAMES"))
	if names == "" {
		logger.Warn("PROVISIONER_NAMES is empty; no provisioners will be used")
		return nil
	}
	return strings.Split(names, ",")
}

func setupConfigWatchers(logger *logrus.Logger, config *ServiceConfig) {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		handleConfigChange(e, logger, config)
	})
}

func handleConfigChange(e fsnotify.Event, logger *logrus.Logger, config *ServiceConfig) {
	logger.WithField("file", e.Name).Info("Configuration updated")
	updateLogSettings(logger)
	config.VolumeFinder.DriverNames = parseDriverNames(logger)
	initializeTracing(logger)
}

func initializeTracing(logger *logrus.Logger) {
	zipkinConfig := struct {
		URI         string
		ServiceName string
		Probability float64
	}{
		URI:         viper.GetString("ZIPKIN_URI"),
		ServiceName: viper.GetString("ZIPKIN_SERVICE_NAME"),
		Probability: viper.GetFloat64("ZIPKIN_PROBABILITY"),
	}

	tp, err := tracer.InitTracing(zipkinConfig.URI, zipkinConfig.Probability)
	if err != nil {
		logger.WithError(err).Error("Tracing initialization failed")
		return
	}

	logger.WithFields(logrus.Fields{
		"uri":         zipkinConfig.URI,
		"service":     zipkinConfig.ServiceName,
		"probability": zipkinConfig.Probability,
	}).Info("Configured tracing")
	otel.SetTracerProvider(tp)
}

func createService(config *ServiceConfig, logger *logrus.Logger) *service.Service {
	return &service.Service{
		VolumeFinder: config.VolumeFinder,
		CertFile:     config.CertFile,
		KeyFile:      config.KeyFile,
		Port:         config.Port,
		Logger:       logger,
		EnableDebug:  config.EnableDebug,
	}
}

func getEnvWithDefault(envVar, defaultValue string) string {
	if value := strings.TrimSpace(viper.GetString(envVar)); value != "" {
		return value
	}
	return defaultValue
}

func parsePort(logger *logrus.Logger) int {
	if port, err := strconv.Atoi(viper.GetString("PORT")); err == nil {
		return port
	}
	logger.Warn("Using default port 443")
	return 443
}

func parseDebugFlag(logger *logrus.Logger) bool {
	debug, err := strconv.ParseBool(viper.GetString("DEBUG"))
	if err != nil {
		logger.WithError(err).Warn("Invalid DEBUG value; defaulting to false")
		return false
	}
	return debug
}
