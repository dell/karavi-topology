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

package k8s_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dell/karavi-topology/internal/k8s"
	"github.com/dell/karavi-topology/internal/k8s/mocks"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_K8sPersistentVolumeFinder(t *testing.T) {
	type checkFn func(*testing.T, []k8s.VolumeInfo, error)
	check := func(fns ...checkFn) []checkFn { return fns }

	hasNoError := func(t *testing.T, _ []k8s.VolumeInfo, err error) {
		if err != nil {
			t.Fatalf("expected no error")
		}
	}

	checkExpectedOutput := func(expectedOutput []k8s.VolumeInfo) func(t *testing.T, volumes []k8s.VolumeInfo, err error) {
		return func(t *testing.T, volumes []k8s.VolumeInfo, _ error) {
			assert.Equal(t, expectedOutput, volumes)
		}
	}

	hasError := func(t *testing.T, _ []k8s.VolumeInfo, err error) {
		if err == nil {
			t.Fatalf("expected error")
		}
	}

	tests := map[string]func(t *testing.T) (k8s.VolumeFinder, []checkFn, *gomock.Controller){
		"success selecting the matching driver name with multiple volumes": func(*testing.T) (k8s.VolumeFinder, []checkFn, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			api := mocks.NewMockVolumeGetter(ctrl)

			t1, err := time.Parse(time.RFC3339, "2020-07-28T20:00:00+00:00")
			assert.Nil(t, err)

			volumes := &corev1.PersistentVolumeList{
				Items: []corev1.PersistentVolume{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "persistent-volume-name",
							CreationTimestamp: metav1.Time{Time: t1},
						},
						Spec: corev1.PersistentVolumeSpec{
							Capacity: map[corev1.ResourceName]resource.Quantity{
								v1.ResourceStorage: resource.MustParse("16Gi"),
							},
							PersistentVolumeSource: corev1.PersistentVolumeSource{
								CSI: &corev1.CSIPersistentVolumeSource{
									Driver: "csi-vxflexos.dellemc.com",
									VolumeAttributes: map[string]string{
										"Name":            "storage-system-volume-name",
										"StoragePoolName": "storage-pool-name",
									},
								},
							},
							ClaimRef: &corev1.ObjectReference{
								Name:      "pvc-name",
								Namespace: "namespace-1",
								UID:       "pvc-uid",
							},
							StorageClassName: "storage-class-name",
						},
						Status: corev1.PersistentVolumeStatus{
							Phase: "Bound",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "persistent-volume-name",
							CreationTimestamp: metav1.Time{Time: t1},
						},
						Spec: corev1.PersistentVolumeSpec{
							Capacity: map[corev1.ResourceName]resource.Quantity{
								v1.ResourceStorage: resource.MustParse("16Gi"),
							},
							PersistentVolumeSource: corev1.PersistentVolumeSource{
								CSI: &corev1.CSIPersistentVolumeSource{
									Driver: "another-csi-driver.dellemc.com",
									VolumeAttributes: map[string]string{
										"Name":            "storage-system-volume-name",
										"StoragePoolName": "storage-pool-name",
									},
								},
							},
							ClaimRef: &corev1.ObjectReference{
								Name:      "pvc-name",
								Namespace: "namespace-1",
								UID:       "pvc-uid",
							},
							StorageClassName: "storage-class-name",
						},
						Status: corev1.PersistentVolumeStatus{
							Phase: "Bound",
						},
					},
				},
			}

			api.EXPECT().GetPersistentVolumes().Times(1).Return(volumes, nil)

			finder := k8s.VolumeFinder{
				API:         api,
				DriverNames: []string{"csi-vxflexos.dellemc.com"},
				Logger:      logrus.New(),
			}
			return finder, check(hasNoError, checkExpectedOutput([]k8s.VolumeInfo{
				{
					Namespace:               "namespace-1",
					PersistentVolumeClaim:   "pvc-uid",
					PersistentVolumeStatus:  "Bound",
					VolumeClaimName:         "pvc-name",
					PersistentVolume:        "persistent-volume-name",
					StorageClass:            "storage-class-name",
					Driver:                  "csi-vxflexos.dellemc.com",
					ProvisionedSize:         "16Gi",
					StorageSystemVolumeName: "storage-system-volume-name",
					StoragePoolName:         "storage-pool-name",
					CreatedTime:             t1.String(),
				},
			})), ctrl
		},
		"success selecting multiple volumes matching multiple driver names": func(*testing.T) (k8s.VolumeFinder, []checkFn, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			api := mocks.NewMockVolumeGetter(ctrl)

			t1, err := time.Parse(time.RFC3339, "2020-07-28T20:00:00+00:00")
			assert.Nil(t, err)

			volumes := &corev1.PersistentVolumeList{
				Items: []corev1.PersistentVolume{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "persistent-volume-name",
							CreationTimestamp: metav1.Time{Time: t1},
						},
						Spec: corev1.PersistentVolumeSpec{
							Capacity: map[corev1.ResourceName]resource.Quantity{
								v1.ResourceStorage: resource.MustParse("16Gi"),
							},
							PersistentVolumeSource: corev1.PersistentVolumeSource{
								CSI: &corev1.CSIPersistentVolumeSource{
									Driver: "csi-vxflexos.dellemc.com",
									VolumeAttributes: map[string]string{
										"Name":            "storage-system-volume-name",
										"StoragePoolName": "storage-pool-name",
									},
								},
							},
							ClaimRef: &corev1.ObjectReference{
								Name:      "pvc-name",
								Namespace: "namespace-1",
								UID:       "pvc-uid",
							},
							StorageClassName: "storage-class-name",
						},
						Status: corev1.PersistentVolumeStatus{
							Phase: "Bound",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "persistent-volume-name-2",
							CreationTimestamp: metav1.Time{Time: t1},
						},
						Spec: corev1.PersistentVolumeSpec{
							Capacity: map[corev1.ResourceName]resource.Quantity{
								v1.ResourceStorage: resource.MustParse("8Gi"),
							},
							PersistentVolumeSource: corev1.PersistentVolumeSource{
								CSI: &corev1.CSIPersistentVolumeSource{
									Driver: "another-csi-driver.dellemc.com",
									VolumeAttributes: map[string]string{
										"Name":     "persistent-volume-name-2",
										"arrayID":  "1.0.1.1",
										"Protocol": "scsi",
									},
								},
							},
							ClaimRef: &corev1.ObjectReference{
								Name:      "pvc-name-2",
								Namespace: "namespace-2",
								UID:       "pvc-uid-2",
							},
							StorageClassName: "storage-class-name-2",
						},
						Status: corev1.PersistentVolumeStatus{
							Phase: "Bound",
						},
					},
					{ // powerscale pvc object
						ObjectMeta: metav1.ObjectMeta{
							Name:              "persistent-volume-name-3",
							CreationTimestamp: metav1.Time{Time: t1},
						},
						Spec: corev1.PersistentVolumeSpec{
							Capacity: map[corev1.ResourceName]resource.Quantity{
								v1.ResourceStorage: resource.MustParse("16Gi"),
							},
							PersistentVolumeSource: corev1.PersistentVolumeSource{
								CSI: &corev1.CSIPersistentVolumeSource{
									Driver: "csi-isilon.dellemc.com",
									VolumeAttributes: map[string]string{
										"Name":              "persistent-volume-name-3",
										"AccessZone":        "System",
										"AzServiceIP":       "192.0.0.1",
										"ClusterName":       "pieisi93x",
										"ID":                "15",
										"Path":              "/ifs/data/csi/persistent-volume-name-3",
										"RootClientEnabled": "false",
									},
								},
							},
							ClaimRef: &corev1.ObjectReference{
								Name:      "pvc-name-3",
								Namespace: "namespace-3",
								UID:       "pvc-uid-3",
							},
							StorageClassName: "storage-class-name-3",
						},
						Status: corev1.PersistentVolumeStatus{
							Phase: "Bound",
						},
					},
					{ // powermax pvc object
						ObjectMeta: metav1.ObjectMeta{
							Name:              "persistent-volume-name-4",
							CreationTimestamp: metav1.Time{Time: t1},
						},
						Spec: corev1.PersistentVolumeSpec{
							Capacity: map[corev1.ResourceName]resource.Quantity{
								v1.ResourceStorage: resource.MustParse("8390400Ki"),
							},
							PersistentVolumeSource: corev1.PersistentVolumeSource{
								CSI: &corev1.CSIPersistentVolumeSource{
									Driver:       "csi-powermax.dellemc.com",
									VolumeHandle: "csi-ZYA-pmax-4723028a00-powermax-000120000606-0012D",
									VolumeAttributes: map[string]string{
										"SRP":            "SRP_1",
										"powermax/SYMID": "000120000606",
										"CreationTime":   "20221128061234",
									},
								},
							},
							ClaimRef: &corev1.ObjectReference{
								Name:      "pvc-name-4",
								Namespace: "namespace-4",
								UID:       "pvc-uid-4",
							},
							StorageClassName: "storage-class-name-4",
						},
						Status: corev1.PersistentVolumeStatus{
							Phase: "Bound",
						},
					},
					{ // non-CSI PV
						ObjectMeta: metav1.ObjectMeta{
							Name:              "persistent-volume-name-3",
							CreationTimestamp: metav1.Time{Time: t1},
						},
						Spec: corev1.PersistentVolumeSpec{
							Capacity: map[corev1.ResourceName]resource.Quantity{
								v1.ResourceStorage: resource.MustParse("16Gi"),
							},
							PersistentVolumeSource: corev1.PersistentVolumeSource{
								NFS: &corev1.NFSVolumeSource{
									Server: "nas-server",
									Path:   "file-path",
								},
							},
							ClaimRef: &corev1.ObjectReference{
								Name:      "pvc-name-4",
								Namespace: "namespace-4",
								UID:       "pvc-uid-4",
							},
							StorageClassName: "storage-class-name-4",
						},
						Status: corev1.PersistentVolumeStatus{
							Phase: "Bound",
						},
					},
				},
			}

			api.EXPECT().GetPersistentVolumes().Times(1).Return(volumes, nil)

			finder := k8s.VolumeFinder{
				API:         api,
				DriverNames: []string{"csi-vxflexos.dellemc.com", "another-csi-driver.dellemc.com", "csi-isilon.dellemc.com", "csi-powermax.dellemc.com"},
				Logger:      logrus.New(),
			}
			return finder, check(hasNoError, checkExpectedOutput([]k8s.VolumeInfo{
				{
					Namespace:               "namespace-1",
					PersistentVolumeClaim:   "pvc-uid",
					PersistentVolumeStatus:  "Bound",
					VolumeClaimName:         "pvc-name",
					PersistentVolume:        "persistent-volume-name",
					StorageClass:            "storage-class-name",
					Driver:                  "csi-vxflexos.dellemc.com",
					ProvisionedSize:         "16Gi",
					StorageSystemVolumeName: "storage-system-volume-name",
					StoragePoolName:         "storage-pool-name",
					CreatedTime:             t1.String(),
				},
				{
					Namespace:               "namespace-2",
					PersistentVolumeClaim:   "pvc-uid-2",
					PersistentVolumeStatus:  "Bound",
					VolumeClaimName:         "pvc-name-2",
					PersistentVolume:        "persistent-volume-name-2",
					StorageClass:            "storage-class-name-2",
					Driver:                  "another-csi-driver.dellemc.com",
					ProvisionedSize:         "8Gi",
					StorageSystemVolumeName: "persistent-volume-name-2",
					StoragePoolName:         "N/A",
					StorageSystem:           "1.0.1.1",
					Protocol:                "scsi",
					CreatedTime:             t1.String(),
				},
				{
					Namespace:               "namespace-3",
					PersistentVolumeClaim:   "pvc-uid-3",
					PersistentVolumeStatus:  "Bound",
					VolumeClaimName:         "pvc-name-3",
					PersistentVolume:        "persistent-volume-name-3",
					StorageClass:            "storage-class-name-3",
					Driver:                  "csi-isilon.dellemc.com",
					ProvisionedSize:         "16Gi",
					StorageSystemVolumeName: "persistent-volume-name-3",
					StoragePoolName:         "N/A",
					StorageSystem:           "pieisi93x:System",
					Protocol:                "nfs",
					CreatedTime:             t1.String(),
				},
				{
					Namespace:               "namespace-4",
					PersistentVolumeClaim:   "pvc-uid-4",
					PersistentVolumeStatus:  "Bound",
					VolumeClaimName:         "pvc-name-4",
					PersistentVolume:        "persistent-volume-name-4",
					StorageClass:            "storage-class-name-4",
					Driver:                  "csi-powermax.dellemc.com",
					ProvisionedSize:         "8390400Ki",
					StorageSystemVolumeName: "0012D:csi-ZYA-pmax-4723028a00-powermax",
					StoragePoolName:         "SRP_1",
					StorageSystem:           "000120000606",
					Protocol:                "N/A",
					CreatedTime:             t1.String(),
				},
			})), ctrl
		},
		"error calling k8s": func(*testing.T) (k8s.VolumeFinder, []checkFn, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			api := mocks.NewMockVolumeGetter(ctrl)
			api.EXPECT().GetPersistentVolumes().Times(1).Return(nil, errors.New("error"))
			finder := k8s.VolumeFinder{
				API:    api,
				Logger: logrus.New(),
			}
			return finder, check(hasError), ctrl
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			finder, checkFns, ctrl := tc(t)
			volumes, err := finder.GetPersistentVolumes(context.TODO())
			for _, checkFn := range checkFns {
				checkFn(t, volumes, err)
			}
			ctrl.Finish()
		})
	}
}
