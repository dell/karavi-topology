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
	"log"
	"net/http"

	"github.com/dell/karavi-topology/internal/k8s"

	"github.com/gorilla/mux"
)

const (
	port = 8080
)

// Service contains data required by the service
type Service struct {
	VolumeFinder VolumeInfoGetter
}

// VolumeInfoGetter is an interface used to get a list of volume information
//go:generate mockgen -destination=mocks/volume_info_getter_mocks.go -package=mocks github.com/dell/karavi-topology/internal/service VolumeInfoGetter
type VolumeInfoGetter interface {
	GetPersistentVolumes() ([]k8s.VolumeInfo, error)
}

// TableResponse is the expected response for getting a list of volumes (reference: https://grafana.com/grafana/plugins/simpod-json-datasource)
type TableResponse struct {
	Columns []map[string]string `json:"columns"`
	Rows    [][]string          `json:"rows"`
	Type    string              `json:"type"`
}

// Run will start the service and listen for HTTP requests
func (s *Service) Run() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", port), s.Routes())
}

// Routes contains the list of routes for the service
func (s *Service) Routes() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", s.rootRequest)
	r.HandleFunc("/query", s.queryRequest)
	return r
}

func (s *Service) rootRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Service) queryRequest(w http.ResponseWriter, r *http.Request) {
	volumes, err := s.VolumeFinder.GetPersistentVolumes()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("%v", err)
		return
	}

	table := generateVolumeTableJSON(volumes)

	output, err := MarshalFn(table)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("%v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(output))
	if err != nil {
		log.Printf("%v", err)
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
		"Created", "Provisioned Size", "Storage Class", "Storage System Volume Name", "Storage Pool")

	table.Rows = make([][]string, 0)
	for _, volume := range volumes {
		table.Rows = append(table.Rows, []string{volume.Namespace, volume.PersistentVolume, volume.PersistentVolumeStatus, volume.VolumeClaimName, volume.Driver, volume.CreatedTime,
			volume.ProvisionedSize, volume.StorageClass, volume.StorageSystemVolumeName, volume.StoragePoolName})
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
