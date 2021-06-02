package service_test

// Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dell/karavi-topology/internal/k8s"
	"github.com/dell/karavi-topology/internal/service"
	"github.com/dell/karavi-topology/internal/service/mocks"
	"github.com/sirupsen/logrus"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type TestCtx struct {
	svc    *service.Service
	server *httptest.Server
}

type monkeyPatch struct {
	marshalFn    func(interface{}) ([]byte, error)
	decodeBodyFn func(body io.Reader, v interface{}) error
	unMarshalFn  func(data []byte, v interface{}) error
	httpWrite    func(w *http.ResponseWriter, data []byte) (int, error)
}

func setup(volumeFinder service.VolumeInfoGetter) (*TestCtx, func()) {
	svc := &service.Service{
		VolumeFinder: volumeFinder,
		Logger:       logrus.New(),
		EnableDebug:  true,
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

func TestSearchHandler(t *testing.T) {
	type checkFn func(*testing.T, *http.Response, error)
	check := func(fns ...checkFn) []checkFn { return fns }

	hasExpectedStatusCode := func(expectedStatus int) func(t *testing.T, response *http.Response, err error) {
		return func(t *testing.T, response *http.Response, err error) {
			assert.NotNil(t, response)
			assert.Equal(t, expectedStatus, response.StatusCode)
		}
	}

	hasExpectedResponse := func(expectedList []string) func(t *testing.T, response *http.Response, err error) {
		return func(t *testing.T, response *http.Response, err error) {
			assert.Nil(t, err)

			body, err := ioutil.ReadAll(response.Body)
			assert.Nil(t, err)

			var result []string
			err = json.Unmarshal(body, &result)
			assert.Nil(t, err)

			contains := func(slice []string, val string) bool {
				for _, item := range slice {
					if item == val {
						return true
					}
				}
				return false
			}

			assert.Equal(t, len(result), len(expectedList))
			for _, v := range result {
				if !contains(expectedList, v) {
					t.Errorf("could not find %s in expectedList", v)
				}
			}
		}
	}

	tests := map[string]func(t *testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader){
		"success with body": func(*testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader) {
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
			expectedList := []string{"ns-1", "ns-2"}

			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)

			var jsonStr = []byte(`{"target":"Namespace"}`)
			return volumeFinder, monkeyPatch{}, check(hasExpectedStatusCode(http.StatusOK), hasExpectedResponse(expectedList)), bytes.NewBuffer(jsonStr)
		},
		"success without body": func(*testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader) {
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
			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)
			expectedList := []string{}
			return volumeFinder, monkeyPatch{}, check(hasExpectedStatusCode(http.StatusOK), hasExpectedResponse(expectedList)), http.NoBody
		},
		"error getting volume info": func(*testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader) {

			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)

			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(nil, errors.New("error"))

			return volumeFinder, monkeyPatch{}, check(hasExpectedStatusCode(http.StatusInternalServerError)), bytes.NewBuffer([]byte(`{"target":"Namespace"}`))
		},
		"error marshalling response": func(*testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader) {
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
			patch := monkeyPatch{
				marshalFn: func(v interface{}) ([]byte, error) {
					return nil, errors.New("error")
				},
			}

			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)

			return volumeFinder, patch, check(hasExpectedStatusCode(http.StatusInternalServerError)), bytes.NewBuffer([]byte(`{"target":"Namespace"}`))
		},
		"error decoding body": func(*testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader) {
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
			patch := monkeyPatch{
				decodeBodyFn: func(body io.Reader, v interface{}) error {
					return errors.New("error")
				},
			}

			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)

			return volumeFinder, patch, check(hasExpectedStatusCode(http.StatusInternalServerError)), bytes.NewBuffer([]byte(`{"target":"Namespace"}`))
		},
		"error http write": func(*testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader) {
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
			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)

			patch := monkeyPatch{
				httpWrite: func(w *http.ResponseWriter, data []byte) (int, error) {
					return 0, errors.New("error")
				},
			}

			return volumeFinder, patch, check(hasExpectedStatusCode(http.StatusInternalServerError)), http.NoBody
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			volumeFinder, patch, checkFns, body := tc(t)

			if patch.marshalFn != nil {
				oldMarshal := service.MarshalFn
				defer func() { service.MarshalFn = oldMarshal }()
				service.MarshalFn = patch.marshalFn
			}
			if patch.decodeBodyFn != nil {
				oldDecodeBodyFn := service.DecodeBodyFn
				defer func() { service.DecodeBodyFn = oldDecodeBodyFn }()
				service.DecodeBodyFn = patch.decodeBodyFn
			}
			if patch.httpWrite != nil {
				oldhttpWrite := service.HTTPWrite
				defer func() { service.HTTPWrite = oldhttpWrite }()
				service.HTTPWrite = patch.httpWrite
			}

			ctx, teardown := setup(volumeFinder)
			defer teardown()

			res, err := http.Post(ctx.server.URL+"/search", "application/json", body)

			for _, checkFn := range checkFns {
				checkFn(t, res, err)
			}
		})
	}
}

func TestQueryHandler(t *testing.T) {
	type checkFn func(*testing.T, []byte, int, error)
	check := func(fns ...checkFn) []checkFn { return fns }

	hasExpectedStatusCode := func(expectedStatus int) func(t *testing.T, body []byte, statusCode int, err error) {
		return func(t *testing.T, body []byte, statusCode int, err error) {
			assert.Equal(t, expectedStatus, statusCode)
		}
	}

	hasExpectedResponse := func(expectedType string, expectedColumns int, expectedRows int) func(t *testing.T, body []byte, statusCode int, err error) {
		return func(t *testing.T, body []byte, _ int, err error) {
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

	hasExpectedNamespacesDriver := func(ns, driver, status string) func(t *testing.T, body []byte, _ int, err error) {
		return func(t *testing.T, body []byte, _ int, err error) {
			assert.Nil(t, err)

			var result []service.TableResponse
			err = json.Unmarshal(body, &result)
			assert.Nil(t, err)

			for _, v := range result[0].Rows {
				assert.Equal(t, ns, v[0])                      // Namespace index is 0
				assert.Equal(t, driver, v[4])                  // CSI Driver index is 4
				assert.True(t, strings.Contains(status, v[2])) // Status index is 2
			}

		}
	}
	var testJSON = []byte(`
		{
			"app":"dashboard",
			"requestId":"Q107",
			"timezone":"browser",
			"targets": [
		       {
			     "target": "{\n   \"Namespace\":\"ns-1\",\n   \"CSI Driver\":\"powerstore\",\n   \"Status\":\"(Bound|Pending)\"\n}",
			     "refId": "A",
			     "hide": false,
			     "type": "table"
		       }
		    ]
		}`)

	tests := map[string]func(t *testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader){
		"success": func(*testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader) {

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

			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)

			expectedType := "table"
			expectedColumns := 12
			expectedRows := 2

			return volumeFinder, monkeyPatch{}, check(hasExpectedStatusCode(http.StatusOK), hasExpectedResponse(expectedType, expectedColumns, expectedRows)), http.NoBody
		},
		"success with target filter": func(*testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader) {
			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)

			volumeInfo := []k8s.VolumeInfo{
				{Namespace: "ns-1", Driver: "powerstore", PersistentVolumeStatus: "Bound"},
				{Namespace: "ns-1", Driver: "powerstore", PersistentVolumeStatus: "Pending"},
				{Namespace: "ns-1", Driver: "powermax", PersistentVolumeStatus: "Bound"},
				{Namespace: "ns-2", Driver: "powerstore", PersistentVolumeStatus: "Pending"},
				{Namespace: "ns-2", Driver: "powermax", PersistentVolumeStatus: "Terminating"},
			}

			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)

			expectedType := "table"
			expectedColumns := 12
			expectedRows := 2

			return volumeFinder, monkeyPatch{}, check(hasExpectedStatusCode(http.StatusOK), hasExpectedResponse(expectedType, expectedColumns, expectedRows), hasExpectedNamespacesDriver("ns-1", "powerstore", "(Bound|Pending)")), bytes.NewBuffer(testJSON)
		},
		"error getting volume info": func(*testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader) {

			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)

			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(nil, errors.New("error"))

			return volumeFinder, monkeyPatch{}, check(hasExpectedStatusCode(http.StatusInternalServerError)), nil
		},
		"error marshalling response": func(*testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader) {

			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)
			volumeInfo := []k8s.VolumeInfo{
				{
					Namespace: "ns-1",
				},
			}
			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)

			patch := monkeyPatch{
				marshalFn: func(v interface{}) ([]byte, error) {
					return nil, errors.New("error")
				},
			}

			return volumeFinder, patch, check(hasExpectedStatusCode(http.StatusInternalServerError)), nil
		},
		"error decoding body": func(*testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader) {
			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)

			volumeInfo := []k8s.VolumeInfo{
				{
					Namespace: "ns-1",
				},
			}
			patch := monkeyPatch{
				decodeBodyFn: func(body io.Reader, v interface{}) error {
					return errors.New("error")
				},
			}
			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)

			return volumeFinder, patch, check(hasExpectedStatusCode(http.StatusInternalServerError)), bytes.NewBuffer([]byte(testJSON))
		},
		"error unmashalling": func(*testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader) {
			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)

			volumeInfo := []k8s.VolumeInfo{
				{
					Namespace: "ns-1",
				},
			}
			patch := monkeyPatch{
				unMarshalFn: func(_ []byte, v interface{}) error {
					return errors.New("error")
				},
			}

			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)

			return volumeFinder, patch, check(hasExpectedStatusCode(http.StatusInternalServerError)), bytes.NewBuffer([]byte(testJSON))
		},
		"error writing http": func(*testing.T) (service.VolumeInfoGetter, monkeyPatch, []checkFn, io.Reader) {
			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)

			volumeInfo := []k8s.VolumeInfo{
				{
					Namespace: "ns-1",
				},
			}
			patch := monkeyPatch{
				httpWrite: func(w *http.ResponseWriter, data []byte) (int, error) {
					return 0, errors.New("error")
				},
			}

			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)

			return volumeFinder, patch, check(hasExpectedStatusCode(http.StatusInternalServerError)), bytes.NewBuffer([]byte(testJSON))
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			volumeFinder, patch, checkFns, body := tc(t)

			if patch.marshalFn != nil {
				oldMarshal := service.MarshalFn
				defer func() { service.MarshalFn = oldMarshal }()
				service.MarshalFn = patch.marshalFn
			}
			if patch.decodeBodyFn != nil {
				oldDecodeBodyFn := service.DecodeBodyFn
				defer func() { service.DecodeBodyFn = oldDecodeBodyFn }()
				service.DecodeBodyFn = patch.decodeBodyFn
			}
			if patch.httpWrite != nil {
				oldhttpWrite := service.HTTPWrite
				defer func() { service.HTTPWrite = oldhttpWrite }()
				service.HTTPWrite = patch.httpWrite
			}
			if patch.unMarshalFn != nil {
				oldunMarshalFn := service.UnMarshalFn
				defer func() { service.UnMarshalFn = oldunMarshalFn }()
				service.UnMarshalFn = patch.unMarshalFn
			}

			ctx, teardown := setup(volumeFinder)
			defer teardown()

			res, err := http.Post(ctx.server.URL+"/query", "application/json", body)
			resBody, errB := ioutil.ReadAll(res.Body)
			assert.Nil(t, errB)

			for _, checkFn := range checkFns {
				checkFn(t, resBody, res.StatusCode, err)
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
