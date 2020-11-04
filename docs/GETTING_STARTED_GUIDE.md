<!--
Copyright (c) 2020 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->
# Getting Started Guide

This document steps through the deployment and configuration of Karavi Topology.

## Prerequisites

The following are prerequisites for building and deploying Karavi Topology.

### Kubernetes

A Kubernetes cluster with the appropriate version below is required for Karavi Topology

| Version   | 
| --------- |
| 1.16-1.17 |

### Dell EMC Storage and CSI Driver

Karavi Topology current has support for the following Dell EMC storage systems and associated CSI drivers.  One of the CSI drivers below must be deployed in the k8s cluster.  The k8s cluster must also have access to the associated storage system.

| Dell EMC Storge Product | CSI Driver |
| ----------------------- | ---------- |
| PowerFlex v3.0/3.5 | [CSI Driver for PowerFlex v1.1.5+](https://github.com/dell/csi-vxflexos) |

### Grafana

Volume visibility is provided through the karavi-topology service and Grafana.  This service exposes an endpoint that can be consumed by Grafana to display CSI driver provisioned volume characteristics correlated with volumes on the storage system.  The [topology Grafana dashboard](../grafana/dashboards/dashboards) requires the following version of Grafana deployed in the k8s cluster. 


| Supported Version | Image                   | Helm Chart                                                |
| ---------------- | ----------------------- | --------------------------------------------------------- |
| v.7.1.0+         | grafana/grafana:7.1.0   | https://github.com/grafana/helm-charts/tree/main/charts/grafana |

Grafana must be configured with the following data sources/plugins:

| Name                   | Additional Information                                                     |
| ---------------------- | -------------------------------------------------------------------------- | 
| JSON data source       | https://grafana.com/grafana/plugins/simpod-json-datasource                 |
| Data Table plugin      | https://grafana.com/grafana/plugins/briangann-datatable-panel/installation |

- Configure the Grafana JSON data source:
 
| Setting | Value                             |
| ------- | --------------------------------- |
| Name    | JSON |
| URL     | The Karavi Topology service can be accessed at the following URL from within your cluster: http://karavi-topology.*namespace*.svc.cluster.local:8080<br>From outside the cluster, the Karavi Topology service can be deployed as a [NodePort](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport) in which case the Karavi Topology service can be accessed at the URL following the [chart instructions](https://github.com/dell/helm-charts/blob/main/charts/karavi-topology/templates/NOTES.txt).<br>From outside the cluster, an [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) and an [Ingress Controller](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/) can be deployed to manage external access to the default ClusterIP configuration of the Karavi Topology service. The Karavi Topology service URL would then be the IP allocated by the Ingress controller to satisfy the Ingress.  |

## Building Karavi Topology (Linux Only)

If you wish to clone and build karavi-topology, a Linux host is required with the following installed:

| Component       | Version   | Additional Information                                                                                                                     |
| --------------- | --------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Docker          | v19+      | https://docs.docker.com/engine/install/                                                                                                    |
| Docker Registry |           | Access to a local/corporate [Docker registry](https://docs.docker.com/registry/)                                                           |
| Golang          | v1.14+    | https://github.com/travis-ci/gimme                                                                                                         |
| gosec           |           | https://github.com/securego/gosec                                                                                                          |
| gomock          | v.1.4.3   | https://github.com/golang/mock                                                                                                             |
| git             | latest    | https://git-scm.com/book/en/v2/Getting-Started-Installing-Git                                                                              |
| gcc             |           | Run ```sudo apt install build-essential```                                                                                                 |
| kubectl         | 1.16-1.17 | Ensure you copy the kubeconfig file from the Kubernetes cluster to the linux host. https://kubernetes.io/docs/tasks/tools/install-kubectl/ |
| Helm            | v.3.3.0   | https://helm.sh/docs/intro/install/                                                                                                        | 

Once all prerequisites are on the Linux host, follow the steps below to clone and build karavi-topology:

1. Clone the karavi-topology repository: `git clone https://github.com/dell/karavi-topology.git`
1. Set the DOCKER_REPO environment variable to point to the local Docker repository, example: `export DOCKER_REPO=<ip-address>:<port>`
1. In the karavi-topology directory, run the following to build the Docker image called karavi-topology: `make clean build docker`
1. To tag (with the "latest" tag) and push the image to the local Docker repository run the following: `make tag push`

__Note:__ If you are using a local insecure docker registry, ensure you configure the insecure registries on each of the Kubernetes worker nodes to allow access to the local docker repository

## Deploying Karavi Topology
Karavi Topology is deployed using Helm.  Usage information and available release versions can be found here: https://github.com/dell/helm-charts/tree/main/charts/karavi-topology.

If you built the Karavi Topology Docker image and pushed it to a local registry, you can deploy it using the same Helm chart above.  You simply need to override the helm chart value pointing to where the Karavi Topology image lives.  See https://github.com/dell/helm-charts/tree/main/charts/karavi-topology for more details.

## Testing Karavi Topology

From the karavi-topology root directory where the repo was cloned, the unit tests can be exectued as follows:
```console
$ make test
```
This will also provide code coverage statistics for the various Go packages.
