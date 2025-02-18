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
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/dell/karavi-topology/internal/entrypoint"
	"github.com/dell/karavi-topology/internal/k8s"
	"github.com/dell/karavi-topology/internal/service"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMainFunction(t *testing.T) {
	tests := []struct {
		name         string
		setup        func()
		expectError  bool
		expectedPort int
		expectedCert string
		expectedKey  string
	}{
		{
			name: "Successful service startup with defaults",
			setup: func() {
				os.Setenv("PORT", "8443")
				os.Setenv("TLS_CERT_PATH", "/certs/cert.pem")
				os.Setenv("TLS_KEY_PATH", "/certs/key.pem")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			mockEntrypointRun := func(_ context.Context, svc entrypoint.ServiceRunner) error {
				// Perform a type assertion
				s, ok := svc.(*service.Service)
				if !ok {
					t.Fatal("Expected service.Service type, but got different implementation")
				}

				if s.Port != 8443 {
					t.Errorf("Expected port 8443, got %d", s.Port)
				}
				if s.CertFile != "/certs/cert.pem" || s.KeyFile != "/certs/key.pem" {
					t.Error("Incorrect TLS paths")
				}
				return nil
			}

			// Capture log output to avoid cluttering test results
			logOutput := bytes.NewBuffer(nil)
			logrus.SetOutput(logOutput)
			defer logrus.SetOutput(os.Stderr)

			// Run mainWithEntrypoint instead of main()
			mainWithEntrypoint(mockEntrypointRun)
		})
	}
}

func TestConfigureLogger(t *testing.T) {
	logger := configureLogger()
	assert.NotNil(t, logger)
	assert.IsType(t, &logrus.Logger{}, logger)
}

func TestSetupViper(t *testing.T) {
	logger := logrus.New()
	setupViper(logger)
	assert.Equal(t, "/etc/config/karavi-topology.yaml", viper.ConfigFileUsed())
}

func TestInitializeServiceConfig(t *testing.T) {
	logger := logrus.New()
	viper.Set("TLS_CERT_PATH", "/test/cert")
	viper.Set("TLS_KEY_PATH", "/test/key")
	viper.Set("PORT", "9090")
	viper.Set("DEBUG", "true")
	viper.Set("PROVISIONER_NAMES", "driver1,driver2")

	config := initializeServiceConfig(logger)

	assert.Equal(t, "/test/cert", config.CertFile)
	assert.Equal(t, "/test/key", config.KeyFile)
	assert.Equal(t, 9090, config.Port)
	assert.True(t, config.EnableDebug)
	assert.NotNil(t, config.VolumeFinder)
	assert.Equal(t, []string{"driver1", "driver2"}, config.VolumeFinder.DriverNames)
}

func TestCreateVolumeFinder(t *testing.T) {
	logger := logrus.New()
	viper.Set("PROVISIONER_NAMES", "driver1,driver2")

	vf := createVolumeFinder(logger)
	assert.NotNil(t, vf)
	assert.Equal(t, []string{"driver1", "driver2"}, vf.DriverNames)
	assert.IsType(t, &k8s.API{}, vf.API)
}

func TestParseDriverNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"Valid drivers", "driver1,driver2", []string{"driver1", "driver2"}},
		{"Single driver", "driver1", []string{"driver1"}},
		{"Empty input", "", nil},
		{"Whitespace input", "  ", nil},
	}

	logger := logrus.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set("PROVISIONER_NAMES", tt.input)
			result := parseDriverNames(logger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleConfigChange(t *testing.T) {
	tests := []struct {
		name                string
		event               fsnotify.Event
		setupConfig         func(*ServiceConfig)
		expectedDriverNames []string
	}{
		{
			name: "Valid config file update",
			event: fsnotify.Event{
				Name: "/etc/config/karavi-topology.yaml",
				Op:   fsnotify.Write,
			},
			setupConfig: func(_ *ServiceConfig) {
				viper.Set("PROVISIONER_NAMES", "csi-driver-1,csi-driver-2")
			},
			expectedDriverNames: []string{"csi-driver-1", "csi-driver-2"},
		},
		{
			name: "Empty provisioner names",
			event: fsnotify.Event{
				Name: "/etc/config/karavi-topology.yaml",
				Op:   fsnotify.Write,
			},
			setupConfig: func(_ *ServiceConfig) {
				viper.Set("PROVISIONER_NAMES", "")
			},
			expectedDriverNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			config := &ServiceConfig{
				VolumeFinder: &k8s.VolumeFinder{},
			}

			// Setup test configuration
			tt.setupConfig(config)

			// Call function
			handleConfigChange(tt.event, logger, config)

			// Validate changes
			if len(config.VolumeFinder.DriverNames) != len(tt.expectedDriverNames) {
				t.Errorf("Expected driver names %v, got %v", tt.expectedDriverNames, config.VolumeFinder.DriverNames)
			}
		})
	}
}

func TestInitializeTracing(_ *testing.T) {
	logger := logrus.New()
	viper.Set("ZIPKIN_URI", "http://localhost:9411")
	viper.Set("ZIPKIN_SERVICE_NAME", "test-service")
	viper.Set("ZIPKIN_PROBABILITY", "1.0")

	initializeTracing(logger)
}

func TestCreateService(t *testing.T) {
	logger := logrus.New()
	config := &ServiceConfig{
		CertFile:     "/test/cert",
		KeyFile:      "/test/key",
		Port:         9090,
		EnableDebug:  true,
		VolumeFinder: &k8s.VolumeFinder{},
	}

	service := createService(config, logger)
	assert.NotNil(t, service)
	assert.Equal(t, config.VolumeFinder, service.VolumeFinder)
	assert.Equal(t, config.CertFile, service.CertFile)
	assert.Equal(t, config.KeyFile, service.KeyFile)
	assert.Equal(t, config.Port, service.Port)
	assert.Equal(t, config.EnableDebug, service.EnableDebug)
}

func TestGetEnvWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		envVar       string
		envValue     string
		defaultValue string
		expected     string
	}{
		{"Use env variable", "TEST_ENV", "custom_value", "default_value", "custom_value"},
		{"Use default value", "TEST_ENV", "", "default_value", "default_value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set(tt.envVar, tt.envValue)
			result := getEnvWithDefault(tt.envVar, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		name     string
		port     string
		expected int
	}{
		{"Valid port", "8080", 8080},
		{"Invalid port", "invalid", 443},
		{"Empty port", "", 443},
	}

	logger := logrus.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set("PORT", tt.port)
			result := parsePort(logger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseDebugFlag(t *testing.T) {
	tests := []struct {
		name     string
		debug    string
		expected bool
	}{
		{"Enable debug", "true", true},
		{"Disable debug", "false", false},
		{"Invalid debug", "invalid", false},
	}

	logger := logrus.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set("DEBUG", tt.debug)
			result := parseDebugFlag(logger)
			assert.Equal(t, tt.expected, result)
		})
	}
}
