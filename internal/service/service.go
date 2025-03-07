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

package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"expvar"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/dell/karavi-topology/internal/k8s"
	"github.com/sirupsen/logrus"

	tracer "github.com/dell/karavi-topology/internal/tracers"
	"github.com/gorilla/mux"
)

const (
	port = 443
)

// Service contains data required by the service
type Service struct {
	VolumeFinder VolumeInfoGetter
	CertFile     string
	KeyFile      string
	Port         int
	Logger       *logrus.Logger
	EnableDebug  bool
}

// VolumeInfoGetter is an interface used to get a list of volume information
//
//go:generate mockgen -destination=mocks/volume_info_getter_mocks.go -package=mocks github.com/dell/karavi-topology/internal/service VolumeInfoGetter
type VolumeInfoGetter interface {
	GetPersistentVolumes(ctx context.Context) ([]k8s.VolumeInfo, error)
}

// Run will start the service and listen for HTTP requests
func (s *Service) Run() error {
	if s.CertFile == "" || s.KeyFile == "" {
		return fmt.Errorf("One or more TLS certificates not supplied: CertFile: %s, KeyFile: %s", s.CertFile, s.KeyFile)
	}
	if s.Port == 0 {
		s.Port = port
	}

	cert, err := tls.LoadX509KeyPair(s.CertFile, s.KeyFile)
	if err != nil {
		return fmt.Errorf("tls.LoadX509KeyPair(%s, %s) failed: %s", s.CertFile, s.KeyFile, err)
	}

	addr := fmt.Sprintf(":%d", s.Port)
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: GetSecuredCipherSuites(),
	}

	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           s.Routes(),
		TLSConfig:         config,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on tcp port %d", s.Port)
	}
	defer func(ln net.Listener) {
		err := ln.Close()
		if err != nil {
			s.Logger.WithError(err).Error("failed to close listener")
		}
	}(ln)
	tlsListener := tls.NewListener(ln, config)

	return server.Serve(tlsListener)
}

// Routes contains the list of routes for the service
func (s *Service) Routes() *mux.Router {
	s.Logger.Debug("setting up routes")
	r := mux.NewRouter()
	r.HandleFunc("/", s.logHandler(s.rootRequest))
	r.HandleFunc("/topology.json", s.logHandler(s.queryRequest))
	if s.EnableDebug {
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/{action}", pprof.Index)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/vars", expvar.Handler().ServeHTTP)
	}
	return r
}

func (s *Service) logHandler(h func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	fn := func(w http.ResponseWriter, r *http.Request) {
		h(w, r)
		s.Logger.WithFields(logrus.Fields{
			"uri":         r.URL.String(),
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
		}).Debug("handling request")
	}
	return http.HandlerFunc(fn)
}

func (s *Service) rootRequest(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Service) queryRequest(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.GetTracer(context.Background(), "GetPersistentVolumes")
	defer span.End()

	volumes, err := s.VolumeFinder.GetPersistentVolumes(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Logger.WithError(err).Error("getting persistent volumes")
		return
	}
	s.Logger.WithField("volumes", len(volumes)).Debug("volumefinder returned persistent volumes")

	var requestBody struct {
		Targets []map[string]interface{} `json:"targets"`
	}

	if err := DecodeBodyFn(r.Body, &requestBody); err != nil {
		if err != io.EOF {
			w.WriteHeader(http.StatusInternalServerError)
			s.Logger.WithError(err).Error("decoding body")
			return
		}
		requestBody.Targets = [](map[string]interface{}){} // no body
	}

	var lookUp []map[string]string
	for _, v := range requestBody.Targets {
		m := make(map[string]string)
		target := strings.Replace(v["target"].(string), "\\", "", -1)
		if err = UnMarshalFn([]byte(target), &m); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.Logger.WithError(err).Errorf("unmarshalling target: %s", target)
			return
		}
		lookUp = append(lookUp, m)
	}

	table := generateVolumeTableJSON(volumes, lookUp)
	fmt.Printf("table::::::::::::::::::::::::::::: %v\n", table)
	s.Logger.WithField("table", len(table)).Debug("generating table response")

	output, err := MarshalFn(table)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Logger.WithError(err).Error("marshalling table response")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	_, err = HTTPWrite(&w, []byte(output))
	if err != nil {
		s.Logger.WithError(err).Error("writing response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// MarshalFn returns the JSON encoding of v
var MarshalFn = func(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// UnMarshalFn parse the JSON encoding of data and the result in value pointed by v
var UnMarshalFn = func(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// DecodeBodyFn reads the next JSON encoding of data and store the result in value pointed by v
var DecodeBodyFn = func(body io.Reader, v interface{}) error {
	return json.NewDecoder(body).Decode(v)
}

// HTTPWrite writes the data to the connection npart of any http request
var HTTPWrite = func(w *http.ResponseWriter, data []byte) (int, error) {
	(*w).WriteHeader(http.StatusOK)
	return (*w).Write([]byte(data))
}

// Table contains the
type Table struct {
	Namespace               string `json:"namespace"`
	PersistentVolume        string `json:"persistent_volume"`
	Status                  string `json:"status"`
	PersistentVolumeClaim   string `json:"persistent_volume_claim"`
	CSIDriver               string `json:"csi_driver"`
	Created                 string `json:"created"`
	ProvisionedSize         string `json:"provisioned_size"`
	StorageClass            string `json:"storage_class"`
	StorageSystemVolumeName string `json:"storage_system_volume_name"`
	StoragePool             string `json:"storage_pool"`
	StorageSystem           string `json:"storage_system"`
	Protocol                string `json:"protocol"`
}

func supportedColumnPair(volume k8s.VolumeInfo) map[string]string {
	return map[string]string{
		"Namespace":      volume.Namespace,
		"Protocol":       volume.Protocol,
		"Status":         volume.PersistentVolumeStatus,
		"CSI Driver":     volume.Driver,
		"Storage Pool":   volume.StoragePoolName,
		"Storage System": volume.StorageSystem,
		"Storage Class":  volume.StorageClass,
	}
}

func canAddRow(volume k8s.VolumeInfo, lookUp []map[string]string) bool {
	canADD := true
	filter := supportedColumnPair(volume)
	for _, look := range lookUp {
		for key, v := range look {
			val, ok := filter[key]
			canADD = canADD && ok && strings.Contains(v, val) // all keys must match
		}
	}
	return canADD
}

func generateVolumeTableJSON(volumes []k8s.VolumeInfo, lookUp []map[string]string) []Table {
	table := make([]Table, 0)

	for _, volume := range volumes {
		if canAddRow(volume, lookUp) {
			table = append(table, Table{
				Namespace:               volume.Namespace,
				PersistentVolume:        volume.PersistentVolume,
				PersistentVolumeClaim:   volume.VolumeClaimName,
				CSIDriver:               volume.Driver,
				Created:                 volume.CreatedTime,
				ProvisionedSize:         volume.ProvisionedSize,
				StorageClass:            volume.StorageClass,
				StorageSystemVolumeName: volume.StorageSystemVolumeName,
				StoragePool:             volume.StoragePoolName,
				StorageSystem:           volume.StorageSystem,
				Protocol:                volume.Protocol,
				Status:                  volume.PersistentVolumeStatus,
			})
		}
	}

	return table
}

// GetSecuredCipherSuites returns a set of secure cipher suites.
func GetSecuredCipherSuites() (suites []uint16) {
	securedSuite := tls.CipherSuites()
	for _, v := range securedSuite {
		suites = append(suites, v.ID)
	}
	return suites
}
