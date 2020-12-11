package service_test

// Copyright (c) 2020 Dell Inc., or its subsidiaries. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dell/karavi-topology/internal/k8s"
	"github.com/dell/karavi-topology/internal/service"
	"github.com/dell/karavi-topology/internal/service/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type TestCtx struct {
	svc    *service.Service
	server *httptest.Server
}

func setup(volumeFinder service.VolumeInfoGetter) (*TestCtx, func()) {
	svc := &service.Service{
		VolumeFinder: volumeFinder,
	}
	ctx := &TestCtx{
		svc:    svc,
		server: httptest.NewServer(svc.Routes()),
	}
	return ctx, func() {
		ctx.server.Close()
	}
}

func TestRootHandler(t *testing.T) {
	type checkFn func(*testing.T, *http.Response, error)
	check := func(fns ...checkFn) []checkFn { return fns }

	hasExpectedStatusCode := func(expectedStatus int, expectedError error) func(t *testing.T, response *http.Response, err error) {
		return func(t *testing.T, response *http.Response, err error) {
			assert.Equal(t, expectedStatus, response.StatusCode)
			assert.Equal(t, expectedError, err)
		}
	}

	tests := map[string]func(t *testing.T) []checkFn{
		"success": func(*testing.T) []checkFn {
			return check(hasExpectedStatusCode(http.StatusOK, nil))
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			checkFns := tc(t)

			ctx, teardown := setup(nil)
			defer teardown()

			res, err := http.Get(ctx.server.URL + "/")
			for _, checkFn := range checkFns {
				checkFn(t, res, err)
			}
		})
	}
}

func TestQueryHandler(t *testing.T) {
	type checkFn func(*testing.T, *http.Response, error)
	check := func(fns ...checkFn) []checkFn { return fns }
	type marshalFn func(interface{}) ([]byte, error)

	hasExpectedStatusCode := func(expectedStatus int) func(t *testing.T, response *http.Response, err error) {
		return func(t *testing.T, response *http.Response, err error) {
			assert.NotNil(t, response)
			assert.Equal(t, expectedStatus, response.StatusCode)
		}
	}

	hasExpectedResponse := func(expectedType string, expectedColumns int, expectedRows int) func(t *testing.T, response *http.Response, err error) {
		return func(t *testing.T, response *http.Response, err error) {
			body, err := ioutil.ReadAll(response.Body)
			assert.Nil(t, err)

			var result []service.TableResponse
			err = json.Unmarshal(body, &result)
			assert.Nil(t, err)

			assert.Equal(t, 1, len(result))
			assert.Equal(t, expectedType, result[0].Type)
			assert.Equal(t, expectedColumns, len(result[0].Columns))
			assert.Equal(t, expectedRows, len(result[0].Rows))
		}
	}

	tests := map[string]func(t *testing.T) (service.VolumeInfoGetter, marshalFn, []checkFn){
		"success": func(*testing.T) (service.VolumeInfoGetter, marshalFn, []checkFn) {

			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)

			volumeInfo := []k8s.VolumeInfo{
				{
					Namespace: "ns-1",
				},
				{
					Namespace: "ns-2",
				},
			}

			volumeFinder.EXPECT().GetPersistentVolumes().Times(1).Return(volumeInfo, nil)

			expectedResponse := []service.TableResponse{
				{
					Type: "table",
				},
			}
			expectedResponse[0].Columns = []map[string]string{}

			expectedType := "table"
			expectedColumns := 10
			expectedRows := 2

			return volumeFinder, nil, check(hasExpectedStatusCode(http.StatusOK), hasExpectedResponse(expectedType, expectedColumns, expectedRows))
		},
		"error getting volume info": func(*testing.T) (service.VolumeInfoGetter, marshalFn, []checkFn) {

			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)

			volumeFinder.EXPECT().GetPersistentVolumes().Times(1).Return(nil, errors.New("error"))

			return volumeFinder, nil, check(hasExpectedStatusCode(http.StatusInternalServerError))
		},
		"error marshalling response": func(*testing.T) (service.VolumeInfoGetter, marshalFn, []checkFn) {

			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)
			volumeInfo := []k8s.VolumeInfo{
				{
					Namespace: "ns-1",
				},
				{
					Namespace: "ns-2",
				},
			}
			volumeFinder.EXPECT().GetPersistentVolumes().Times(1).Return(volumeInfo, nil)

			marshal := func(v interface{}) ([]byte, error) {
				return nil, errors.New("error")
			}

			return volumeFinder, marshal, check(hasExpectedStatusCode(http.StatusInternalServerError))
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			volumeFinder, marshalFn, checkFns := tc(t)

			if marshalFn != nil {
				oldMarshal := service.MarshalFn
				defer func() { service.MarshalFn = oldMarshal }()
				service.MarshalFn = marshalFn
			}
			ctx, teardown := setup(volumeFinder)
			defer teardown()

			res, err := http.Post(ctx.server.URL+"/query", "application/json", nil)

			for _, checkFn := range checkFns {
				checkFn(t, res, err)
			}
		})
	}
}

func TestHttpServerStartup(t *testing.T) {
	type checkFn func(*testing.T, bool, error)

	expectedError := func(t *testing.T, expectError bool, err error) {
		if expectError {
			assert.Error(t, err)
		} else {
			assert.Nil(t, err)
		}
	}

	tests := map[string]func(t *testing.T) (string, string, int, []checkFn, bool){
		"error no certs": func(*testing.T) (string, string, int, []checkFn, bool) {
			certFile := ""
			keyFile := ""

			return certFile, keyFile, 0, []checkFn{expectedError}, true
		},
		"invalid certs": func(*testing.T) (string, string, int, []checkFn, bool) {
			certFile := "/not-valid-certs/ca.crt"
			keyFile := "/not-valid-certs/key.file"

			return certFile, keyFile, 0, []checkFn{expectedError}, true
		},
		"port specified - invalid certs": func(*testing.T) (string, string, int, []checkFn, bool) {
			certFile := "/not-valid-certs/ca.crt"
			keyFile := "/not-valid-certs/key.file"

			return certFile, keyFile, 8443, []checkFn{expectedError}, true
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			certFile, keyFile, port, checkFns, expectError := tc(t)

			ctx, teardown := setup(nil)
			defer teardown()
			ctx.svc.CertFile = certFile
			ctx.svc.KeyFile = keyFile
			ctx.svc.Port = port

			err := ctx.svc.Run()

			for _, checkFn := range checkFns {
				checkFn(t, expectError, err)
			}
		})
	}
}
