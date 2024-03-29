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

package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"

	tracer "github.com/dell/karavi-topology/internal/tracers"
	"github.com/sirupsen/logrus"
)

const (
	// ProtocolNfs Protocol NFS in lower case
	ProtocolNfs = "nfs"
	// CsiDriverNamePowerScale CSI PowerScale Name
	CsiDriverNamePowerScale = "isilon"
	// CsiDriverNamePowerMax CSI PowerMax Name
	CsiDriverNamePowerMax = "powermax"
)

// VolumeGetter is an interface for getting a list of persistent volume information
//
//go:generate mockgen -destination=mocks/volume_getter_mocks.go -package=mocks github.com/dell/karavi-topology/internal/k8s VolumeGetter
type VolumeGetter interface {
	GetPersistentVolumes() (*corev1.PersistentVolumeList, error)
}

// VolumeFinder is a volume finder that will query the Kubernetes API for Persistent Volumes created by a matching DriverName
type VolumeFinder struct {
	API         VolumeGetter
	DriverNames []string
	Logger      *logrus.Logger
}

// VolumeInfo contains information about mapping a Persistent Volume to the volume created on a storage system
type VolumeInfo struct {
	Namespace               string `json:"namespace"`
	PersistentVolumeClaim   string `json:"persistent_volume_claim"`
	PersistentVolumeStatus  string `json:"volume_status"`
	VolumeClaimName         string `json:"volume_claim_name"`
	PersistentVolume        string `json:"persistent_volume"`
	StorageClass            string `json:"storage_class"`
	Driver                  string `json:"driver"`
	ProvisionedSize         string `json:"provisioned_size"`
	StorageSystemVolumeName string `json:"storage_system_volume_name"`
	StoragePoolName         string `json:"storage_pool_name"`
	StorageSystem           string `json:"storage_system"`
	Protocol                string `json:"protocol"`
	CreatedTime             string `json:"created_time"`
}

// GetPersistentVolumes will return a list of persistent volume information
func (f VolumeFinder) GetPersistentVolumes(ctx context.Context) ([]VolumeInfo, error) {
	ctx, span := tracer.GetTracer(ctx, "GetPersistentVolumes")
	defer span.End()

	start := time.Now()
	defer f.timeSince(start, "GetPersistentVolumes")

	volumeInfo := make([]VolumeInfo, 0)

	volumes, err := f.API.GetPersistentVolumes()
	if err != nil {
		return nil, err
	}

	for _, volume := range volumes.Items {
		if volume.Spec.CSI != nil && Contains(f.DriverNames, volume.Spec.CSI.Driver) {
			capacity := volume.Spec.Capacity[v1.ResourceStorage]
			claim := volume.Spec.ClaimRef
			status := volume.Status
			volumeHandle := volume.Spec.CSI.VolumeHandle

			f.Logger.WithField("volume_attributes", volume.Spec.CSI.VolumeAttributes).Debug("volumefinder volumes attributes map")
			info := VolumeInfo{
				Namespace:               claim.Namespace,
				PersistentVolumeClaim:   string(claim.UID),
				VolumeClaimName:         claim.Name,
				PersistentVolumeStatus:  string(status.Phase),
				PersistentVolume:        volume.Name,
				StorageClass:            volume.Spec.StorageClassName,
				Driver:                  volume.Spec.CSI.Driver,
				ProvisionedSize:         capacity.String(),
				StorageSystemVolumeName: volume.Spec.CSI.VolumeAttributes["Name"],
				StoragePoolName:         volume.Spec.CSI.VolumeAttributes["StoragePoolName"],
				StorageSystem:           volume.Spec.CSI.VolumeAttributes["StorageSystem"],
				Protocol:                volume.Spec.CSI.VolumeAttributes["Protocol"],
				CreatedTime:             volume.CreationTimestamp.String(),
			}
			// powerstore do not return this value, csi created volume has storage volume name and pv name same
			if info.StorageSystemVolumeName == "" || len(info.StorageSystemVolumeName) == 0 {
				info.StorageSystemVolumeName = volume.Name
			}

			// powerflex will provide storagesystem id and powerstore will provide array IP
			if info.StorageSystem == "" || len(info.StorageSystem) == 0 {
				info.StorageSystem = volume.Spec.CSI.VolumeAttributes["arrayID"]
			}

			// set StoregeSystem and Protocol for PowerScale
			if strings.Contains(info.Driver, CsiDriverNamePowerScale) {
				info.StorageSystem = volume.Spec.CSI.VolumeAttributes["ClusterName"] + ":" + volume.Spec.CSI.VolumeAttributes["AccessZone"]
				info.Protocol = ProtocolNfs
			}

			// set attributes for PowerMax
			if strings.Contains(info.Driver, CsiDriverNamePowerMax) {
				info.StorageSystemVolumeName = parsePowerMaxVolumeName(volumeHandle)
				info.StorageSystem = volume.Spec.CSI.VolumeAttributes["powermax/SYMID"]
				info.StoragePoolName = volume.Spec.CSI.VolumeAttributes["SRP"]
				if info.Protocol == "" {
					info.Protocol = "N/A"
				}
			}

			// powerstore volume do not have storage pool unlike powerflex
			if info.StoragePoolName == "" || len(info.StoragePoolName) == 0 {
				info.StoragePoolName = "N/A"
			}

			volumeInfo = append(volumeInfo, info)
		}
	}
	return volumeInfo, nil
}

// parsePowerMaxVolumeName parse PowerMax PV volumeHandle and return storage volume name
func parsePowerMaxVolumeName(volumeHandle string) string {
	ele := strings.Split(volumeHandle, "-")
	if len(ele) == 7 {
		return fmt.Sprintf("%s:%s", ele[6], strings.Join(ele[0:5], "-"))
	}

	return volumeHandle
}

// Contains will return true if the slice contains the given value
func Contains(slice []string, value string) bool {
	for _, element := range slice {
		if element == value {
			return true
		}
	}
	return false
}

func (f VolumeFinder) timeSince(start time.Time, fName string) {
	f.Logger.WithFields(logrus.Fields{
		"duration": fmt.Sprintf("%v", time.Since(start)),
		"function": fName,
	}).Debug("function duration")
}
