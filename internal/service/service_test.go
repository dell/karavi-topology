/*
 Copyright (c) 2020-2023 Dell Inc. or its subsidiaries. All Rights Reserved.

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

package service_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

type testOverrides struct {
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

func TestQueryHandler(t *testing.T) {
	type checkFn func(*testing.T, []byte, int, error)
	check := func(fns ...checkFn) []checkFn { return fns }

	hasExpectedStatusCode := func(expectedStatus int) func(t *testing.T, body []byte, statusCode int, err error) {
		return func(t *testing.T, _ []byte, statusCode int, _ error) {
			assert.Equal(t, expectedStatus, statusCode)
		}
	}

	hasExpectedResponse := func() func(t *testing.T, body []byte, statusCode int, err error) {
		return func(t *testing.T, body []byte, _ int, err error) {
			assert.Nil(t, err)

			var result []service.Table
			err = json.Unmarshal(body, &result)
			assert.Nil(t, err)

			assert.Equal(t, 1, len(result))
		}
	}

	testJSON := []byte(`
	{
		"app":"dashboard",
		"requestId":"Q107",
		"timezone":"browser",
		"targets": [
		   {
			 "target": "{\n   \"namespace\":\"ns-1\",\n   \"csi_driver\":\"powerstore\",\n   \"Status\":\"(Bound|Pending)\"\n}",
			 "refId": "A",
			 "hide": false,
			 "type": "table"
		   }
		]
	}`)

	tests := map[string]func(t *testing.T) (service.VolumeInfoGetter, testOverrides, []checkFn, io.Reader){
		"success": func(*testing.T) (service.VolumeInfoGetter, testOverrides, []checkFn, io.Reader) {
			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)

			volumeInfo := []k8s.VolumeInfo{
				{
					Namespace: "ns-1",
				},
			}

			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)

			return volumeFinder, testOverrides{}, check(hasExpectedStatusCode(http.StatusOK), hasExpectedResponse()), http.NoBody
		},
		"error getting volume info": func(*testing.T) (service.VolumeInfoGetter, testOverrides, []checkFn, io.Reader) {
			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)

			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(nil, errors.New("error"))

			return volumeFinder, testOverrides{}, check(hasExpectedStatusCode(http.StatusInternalServerError)), nil
		},
		"error marshalling response": func(*testing.T) (service.VolumeInfoGetter, testOverrides, []checkFn, io.Reader) {
			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)
			volumeInfo := []k8s.VolumeInfo{
				{
					Namespace: "ns-1",
				},
			}
			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)

			patch := testOverrides{
				marshalFn: func(_ interface{}) ([]byte, error) {
					return nil, errors.New("error")
				},
			}

			return volumeFinder, patch, check(hasExpectedStatusCode(http.StatusInternalServerError)), nil
		},
		"error decoding body": func(*testing.T) (service.VolumeInfoGetter, testOverrides, []checkFn, io.Reader) {
			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)

			volumeInfo := []k8s.VolumeInfo{
				{
					Namespace: "ns-1",
				},
			}
			patch := testOverrides{
				decodeBodyFn: func(_ io.Reader, _ interface{}) error {
					return errors.New("error")
				},
			}
			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)

			return volumeFinder, patch, check(hasExpectedStatusCode(http.StatusInternalServerError)), bytes.NewBuffer([]byte(testJSON))
		},
		"error unmashalling": func(*testing.T) (service.VolumeInfoGetter, testOverrides, []checkFn, io.Reader) {
			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)

			volumeInfo := []k8s.VolumeInfo{
				{
					Namespace: "ns-1",
				},
			}
			patch := testOverrides{
				unMarshalFn: func(_ []byte, _ interface{}) error {
					return errors.New("error")
				},
			}

			volumeFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Times(1).Return(volumeInfo, nil)

			return volumeFinder, patch, check(hasExpectedStatusCode(http.StatusInternalServerError)), bytes.NewBuffer([]byte(testJSON))
		},
		"error writing http": func(*testing.T) (service.VolumeInfoGetter, testOverrides, []checkFn, io.Reader) {
			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)

			volumeInfo := []k8s.VolumeInfo{
				{
					Namespace: "ns-1",
				},
			}
			patch := testOverrides{
				httpWrite: func(_ *http.ResponseWriter, _ []byte) (int, error) {
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

			res, err := http.Post(ctx.server.URL+"/topology.json", "application/json", body)
			resBody, errB := io.ReadAll(res.Body)
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

	tests := map[string]func(t *testing.T) (string, string, int, []checkFn, bool, bool){
		"error no certs": func(*testing.T) (string, string, int, []checkFn, bool, bool) {
			certFile := ""
			keyFile := ""

			return certFile, keyFile, 0, []checkFn{expectedError}, true, false
		},
		"invalid certs": func(*testing.T) (string, string, int, []checkFn, bool, bool) {
			certFile := "/not-valid-certs/ca.crt"
			keyFile := "/not-valid-certs/key.file"

			return certFile, keyFile, 0, []checkFn{expectedError}, true, false
		},
		"port specified - invalid certs": func(*testing.T) (string, string, int, []checkFn, bool, bool) {
			certFile := "/not-valid-certs/ca.crt"
			keyFile := "/not-valid-certs/key.file"

			return certFile, keyFile, 8443, []checkFn{expectedError}, true, false
		},
		"port specified - valid certs": func(*testing.T) (string, string, int, []checkFn, bool, bool) {
			certFile := "testdata/cert.crt"
			keyFile := "testdata/key.key"

			return certFile, keyFile, 8443, []checkFn{expectedError}, false, false
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			certFile, keyFile, port, checkFns, expectError, expectPanic := tc(t)

			ctx, teardown := setup(nil)
			defer teardown()
			ctx.svc.CertFile = certFile
			ctx.svc.KeyFile = keyFile
			ctx.svc.Port = port

			done := make(chan bool)

			go func() {
				defer func() {
					if r := recover(); r != nil {
						if !expectPanic {
							t.Errorf("Did not expect panic, but function panicked")
						}
					} else {
						if expectPanic {
							t.Errorf("Expected panic, but function did not panic")
						}
					}
					done <- true
				}()
				err := ctx.svc.Run()
				for _, checkFn := range checkFns {
					checkFn(t, expectError, err)
				}
			}()

			select {
			case <-done:
				// Test completed
			case <-time.After(5 * time.Second):
				if name == "port specified - valid certs" {
					t.Log("Test timed out as expected")
				} else {
					t.Fatal("Test timed out unexpectedly")
				}
			}
		})
	}
}

func TestGetSecuredCipherSuites(t *testing.T) {
	expectedSuites := tls.CipherSuites()
	expectedIDs := make([]uint16, len(expectedSuites))
	for i, suite := range expectedSuites {
		expectedIDs[i] = suite.ID
	}

	got := service.GetSecuredCipherSuites()

	if len(got) != len(expectedIDs) {
		t.Fatalf("Expected %d cipher suites, but got %d", len(expectedIDs), len(got))
	}

	for i, id := range expectedIDs {
		if got[i] != id {
			t.Errorf("Expected cipher suite ID %x at index %d, but got %x", id, i, got[i])
		}
	}
}
