package service

// Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dell/karavi-topology/internal/k8s"
	"github.com/sirupsen/logrus"

	"expvar"
	"net/http/pprof"

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
//go:generate mockgen -destination=mocks/volume_info_getter_mocks.go -package=mocks github.com/dell/karavi-topology/internal/service VolumeInfoGetter
type VolumeInfoGetter interface {
	GetPersistentVolumes(ctx context.Context) ([]k8s.VolumeInfo, error)
}

// TableResponse is the expected response for getting a list of volumes (reference: https://grafana.com/grafana/plugins/grafana-simple-json-datasource)
type TableResponse struct {
	Columns []map[string]string `json:"columns"`
	Rows    [][]string          `json:"rows"`
	Type    string              `json:"type"`
}

// Run will start the service and listen for HTTP requests
func (s *Service) Run() error {
	if s.CertFile == "" || s.KeyFile == "" {
		return fmt.Errorf("One or more TLS certificates not supplied: CertFile: %s, KeyFile: %s", s.CertFile, s.KeyFile)
	}
	if s.Port == 0 {
		s.Port = port
	}
	return http.ListenAndServeTLS(fmt.Sprintf(":%d", s.Port), s.CertFile, s.KeyFile, s.Routes())
}

// Routes contains the list of routes for the service
func (s *Service) Routes() *mux.Router {
	s.Logger.Debug("setting up routes")
	r := mux.NewRouter()
	r.HandleFunc("/", s.logHandler(s.rootRequest))
	r.HandleFunc("/query", s.logHandler(s.queryRequest))
	r.HandleFunc("/search", s.logHandler(s.searchRequest))
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

func (s *Service) rootRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Service) searchRequest(w http.ResponseWriter, r *http.Request) {

	write := func(out []byte) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		_, err := HTTPWrite(&w, []byte(out))
		if err != nil {
			s.Logger.WithError(err).Error("writing response")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	}

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
		Target string `json:"target"`
	}
	if err := DecodeBodyFn(r.Body, &requestBody); err != nil {
		if err == io.EOF { // no body
			write([]byte("[]"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		s.Logger.WithError(err).Error("decoding body")
		return
	}

	list := generateVolumeAvailableMetrics(volumes, requestBody.Target)
	output, err := MarshalFn(list)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Logger.WithError(err).Error("marshalling list response")
		return
	}
	write(output)
}

func generateVolumeAvailableMetrics(volumes []k8s.VolumeInfo, key string) []string {
	found := make(map[string]bool)
	for _, volume := range volumes {
		filter := supportedColumnPair(volume)
		if val, ok := filter[key]; ok {
			found[val] = true
		}
	}

	metrics := []string{}
	for m := range found {
		metrics = append(metrics, m)
	}
	return metrics
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

func supportedColumnPair(volume k8s.VolumeInfo) map[string]string {
	return map[string]string{
		"Namespace":      volume.Namespace,
		"Protocol":       volume.Protocol,
		"Status":         volume.PersistentVolumeStatus,
		"CSI Driver":     volume.Driver,
		"Storage Pool":   volume.StoragePoolName,
		"Storage System": volume.StorageSystem,
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

func generateVolumeTableJSON(volumes []k8s.VolumeInfo, lookUp []map[string]string) []*TableResponse {
	table := &TableResponse{
		Type: "table",
	}

	table.Columns = generateColumns("Namespace", "Persistent Volume", "Status", "Persistent Volume Claim", "CSI Driver",
		"Created", "Provisioned Size", "Storage Class", "Storage System Volume Name", "Storage Pool", "Storage System", "Protocol")

	table.Rows = make([][]string, 0)
	for _, volume := range volumes {
		if canAddRow(volume, lookUp) {
			table.Rows = append(table.Rows, []string{volume.Namespace, volume.PersistentVolume, volume.PersistentVolumeStatus, volume.VolumeClaimName, volume.Driver, volume.CreatedTime,
				volume.ProvisionedSize, volume.StorageClass, volume.StorageSystemVolumeName, volume.StoragePoolName, volume.StorageSystem, volume.Protocol})
		}
	}
	return []*TableResponse{table}
}

func generateColumns(columns ...string) []map[string]string {
	result := make([]map[string]string, 0)
	for _, column := range columns {
		result = append(result, map[string]string{"text": column, "type": "string"})
	}
	return result
}
