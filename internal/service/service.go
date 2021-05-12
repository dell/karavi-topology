package service

// Copyright (c) 2020 Dell Inc., or its subsidiaries. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dell/karavi-topology/internal/k8s"
	"github.com/sirupsen/logrus"

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
}

// VolumeInfoGetter is an interface used to get a list of volume information
//go:generate mockgen -destination=mocks/volume_info_getter_mocks.go -package=mocks github.com/dell/karavi-topology/internal/service VolumeInfoGetter
type VolumeInfoGetter interface {
	GetPersistentVolumes() ([]k8s.VolumeInfo, error)
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

func (s *Service) queryRequest(w http.ResponseWriter, r *http.Request) {
	volumes, err := s.VolumeFinder.GetPersistentVolumes()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Logger.WithError(err).Error("getting persistent volumes")
		return
	}
	s.Logger.WithField("volumes", len(volumes)).Debug("volumefinder returned persistent volumes")
	table := generateVolumeTableJSON(volumes)

	output, err := MarshalFn(table)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Logger.WithError(err).Error("marshalling response")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(output))
	if err != nil {
		s.Logger.WithError(err).Error("writing response")
		return
	}
}

// MarshalFn returns the JSON encoding of v
var MarshalFn = func(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func generateVolumeTableJSON(volumes []k8s.VolumeInfo) []*TableResponse {
	table := &TableResponse{
		Type: "table",
	}

	table.Columns = generateColumns("Namespace", "Persistent Volume", "Status", "Persistent Volume Claim", "CSI Driver",
		"Created", "Provisioned Size", "Storage Class", "Storage System Volume Name", "Storage Pool", "Storage System")

	table.Rows = make([][]string, 0)
	for _, volume := range volumes {
		table.Rows = append(table.Rows, []string{volume.Namespace, volume.PersistentVolume, volume.PersistentVolumeStatus, volume.VolumeClaimName, volume.Driver, volume.CreatedTime,
			volume.ProvisionedSize, volume.StorageClass, volume.StorageSystemVolumeName, volume.StoragePoolName, volume.StorageSystem})
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
