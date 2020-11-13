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

const (
	TestCert = "MIIDUjCCAjoCCQDyMDAiNfIGJTANBgkqhkiG9w0BAQsFADBrMQswCQYDVQQGEwJ1czESMBAGA1UECAwJYm9iIHNtaXRoMRUwEwYDVQQHDAxEZWZhdWx0IENpdHkxHDAaBgNVBAoME0RlZmF1bHQgQ29tcGFueSBMdGQxEzARBgNVBAMMCnRlc3QubG9jYWwwHhcNMjAxMTEzMTcxNjAyWhcNMjAxMjEzMTcxNjAyWjBrMQswCQYDVQQGEwJ1czESMBAGA1UECAwJYm9iIHNtaXRoMRUwEwYDVQQHDAxEZWZhdWx0IENpdHkxHDAaBgNVBAoME0RlZmF1bHQgQ29tcGFueSBMdGQxEzARBgNVBAMMCnRlc3QubG9jYWwwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCkUgm/+qHu2hhcHJOG+pgZKyC5fNr/G2ohKJkFwcISjefVZvD9zRiKGiHv/N3b9ZsrHs6JeT61oDysT4FgcU9QxBhgr/8efsXvjWKbk0A9Mz0csecekdzWl5ch29q/NOg6dBn+pfTuh5B8d8bdr0XYzvfIMSoBfTmiTbFvHUkli4gBJkDMmqErfkc4tZqQA3QO4ka5thLBELz0fVAfgjwfjc375dnvVPYI/CnxRpZ0szdtAqYq2a/+cA4Wm6z0YG+mPUk0MxllFNcgngOaNA/pyIPJAV+UWI0OXkTARxuKBE+DptrmqpEdxbjsiEjBc5i01XNGUVZgOAF8WfTp1K85AgMBAAEwDQYJKoZIhvcNAQELBQADggEBABPggTdEtzmNu64RUMfzheerjJm0l6902AhqPNdhUVltAHYehmRqM9L7oxsw4fbzL89vgS2Jo7wnlPBmcDPlgGXsyJZaEVyXYr3wvoz44zdKlqFwFOpglkgtz22ePAAs5d3NE3lJ4GTj46eCOCZu+pwYUnguW3KLTct9dUToenz40ZQTEbRzFOGd0pp8/lWKcUPkgHDo6SVfitlbm9dp45r1/3vgSL2SL//AzNiTZPMnqD74pbCOEtWDgLXPEO2LHn9+84MHLEAWx6MwQDES+gcf1OXjF9gQOiS5sSR5VL/rTJABn54Ytw+XsGFXOaErD9IKOC1e6bKX9/dR498PpAw="
	TestKey  = "MIIEuwIBADANBgkqhkiG9w0BAQEFAASCBKUwggShAgEAAoIBAQCkUgm/+qHu2hhcHJOG+pgZKyC5fNr/G2ohKJkFwcISjefVZvD9zRiKGiHv/N3b9ZsrHs6JeT61oDysT4FgcU9QxBhgr/8efsXvjWKbk0A9Mz0csecekdzWl5ch29q/NOg6dBn+pfTuh5B8d8bdr0XYzvfIMSoBfTmiTbFvHUkli4gBJkDMmqErfkc4tZqQA3QO4ka5thLBELz0fVAfgjwfjc375dnvVPYI/CnxRpZ0szdtAqYq2a/+cA4Wm6z0YG+mPUk0MxllFNcgngOaNA/pyIPJAV+UWI0OXkTARxuKBE+DptrmqpEdxbjsiEjBc5i01XNGUVZgOAF8WfTp1K85AgMBAAECggEAVWfqd2T+a5Xh2WZk62AuU86NuLsDXFrUY52lQ0+83kXxfIJ/uXqzSXxhrUnByoSyfMwLT3q8NEyvnGPKe+UI85KolQCE2kXL/UGAQhpb5jNOjS6OUN5BaSvrOob6AC2wkkksBaYeUygU2gyrSSfDZvfT47JiAdojbY9yLK2NCjzRNzqmLwcunXtffwNMLtPpONxpZ4u2Bvq++6KV9H6yEhp/s7AWFrrDVGHWmwi7SlHFbJJYql8kmq3Psmmoi2FE/VG+U6BsJF852lGrm6GqpkqQYDskVX7TihwYXQXUF70wxU7vzLP1jFBE3azxY2XPGde6diyXx2t14HBWU7YGXQKBgQDVliZqCJDj/am7ZPTiqtcMBoA/zsWuuYGjEgUWn0cE2pFKhg3FDAsVmDUofMcViGX8d66CklMIodBNWXSSnSx3cQeFsxPRbTuTloSQZX+AuRocjHZDDpsFQAtyTFyy/MVcyrE4n6zBxardVlxS9t31PBVsblsDwUA0vxzZ0J2NWwKBgQDE82gNlg44jFpE99+wLQVTUmApQ4nI9v8epYegfhWiW/EhB1KrGet1j8zOx3OS96Rf0bbpzcg76/imlz/d4sJsPhZ6uL0ZvfBo6tmPuZHMUF1tKQWUXAz90IkrpzjxwfjwA19eUzLKz23+Ir4ojGKHjeR3j8SdnEK2PwHJsqn1+wKBgEbKChdQmX0HAK9cSZGqn7WbnfwH8xry3tWGTmtuBOLF8iup/HxXfoD8vnmZyX4IhAzGOe+Kwbx1rQ1F3c4OC8PWkXCtpp7dvkYvN+aHzVeDgfT+VN/qwlReIq0SRBMKlfsoLs6elWpvsi7Dxbu1mGEENfGHLeEztq0EvnIuo1lLAn9LZOJwUQEgpJnpzPnUd2eSffZR1YjpZaREFxnUVm/xt0CXZDZBSarZVjMQ9UlI+YPzKlTbK+t7BNoq67uHNUc4KIxybkX1lMBzaXPfkSo/DIS3RPzdzl8qyqm4DEvAQIELYD8h3LeU69Mvdh1VaGhPfAH5ww+BRlBDc9s7Wym1AoGBAMZ+KfLg3HKiSHa8DPQ2c2mmW46KRBNQZh+ljn+GmjKBttZu6bci38k2ERGrWJWe6ImLXPMIipGSZPtaOUT9OVx+/PHhXTRoml5BmZYSl0NYN47jkjZLeKFeulrJhZol2QM8aKVuAGfe6wayIa+xUIqKnKU4GrM4FhPb0I9nvlk3"
)

type TestCtx struct {
	svc    *service.Service
	server *httptest.Server
}

func setup(volumeFinder service.VolumeInfoGetter, certFile string, keyFile string) (*TestCtx, func()) {
	svc := &service.Service{
		VolumeFinder: volumeFinder,
		CertFile:     certFile,
		KeyFile:      keyFile,
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

			ctx, teardown := setup(nil, "", "")
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

	tests := map[string]func(t *testing.T) (service.VolumeInfoGetter, string, string, marshalFn, []checkFn){
		"success": func(*testing.T) (service.VolumeInfoGetter, string, string, marshalFn, []checkFn) {

			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)
			certFile := TestCert
			keyFile := TestKey

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

			return volumeFinder, certFile, keyFile, nil, check(hasExpectedStatusCode(http.StatusOK), hasExpectedResponse(expectedType, expectedColumns, expectedRows))
		},
		"error getting volume info": func(*testing.T) (service.VolumeInfoGetter, string, string, marshalFn, []checkFn) {

			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)
			certFile := TestCert
			keyFile := TestKey

			volumeFinder.EXPECT().GetPersistentVolumes().Times(1).Return(nil, errors.New("error"))

			return volumeFinder, certFile, keyFile, nil, check(hasExpectedStatusCode(http.StatusInternalServerError))
		},
		"error marshalling response": func(*testing.T) (service.VolumeInfoGetter, string, string, marshalFn, []checkFn) {

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
			certFile := TestCert
			keyFile := TestKey

			volumeFinder.EXPECT().GetPersistentVolumes().Times(1).Return(volumeInfo, nil)

			marshal := func(v interface{}) ([]byte, error) {
				return nil, errors.New("error")
			}
			return volumeFinder, certFile, keyFile, marshal, check(hasExpectedStatusCode(http.StatusInternalServerError))
		},
		"error no certs provided": func(*testing.T) (service.VolumeInfoGetter, string, string, marshalFn, []checkFn) {
			ctrl := gomock.NewController(t)
			volumeFinder := mocks.NewMockVolumeInfoGetter(ctrl)
			certFile := ""
			keyFile := ""

			volumeInfo := []k8s.VolumeInfo{
				{
					Namespace: "ns-1",
				},
				{
					Namespace: "ns-2",
				},
			}

			volumeFinder.EXPECT().GetPersistentVolumes().Times(1).Return(volumeInfo, nil)

			return volumeFinder, certFile, keyFile, nil, check(hasExpectedStatusCode(http.StatusInternalServerError))
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			volumeFinder, certFile, keyFile, marshalFn, checkFns := tc(t)

			if marshalFn != nil {
				oldMarshal := service.MarshalFn
				defer func() { service.MarshalFn = oldMarshal }()
				service.MarshalFn = marshalFn
			}
			ctx, teardown := setup(volumeFinder, certFile, keyFile)
			defer teardown()

			res, err := http.Post(ctx.server.URL+"/query", "application/json", nil)

			for _, checkFn := range checkFns {
				checkFn(t, res, err)
			}
		})
	}
}
