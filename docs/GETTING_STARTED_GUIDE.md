<!--
Copyright (c) 2020 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Getting Started Guide

Karavi Topology provides Kubernetes administrators, via Grafana, the topology data related to containerized storage that are provisioned by a CSI (Container Storage Interface) Driver for Dell EMC storage products.

This document steps through the deployment and configuration of Karavi Topology.

## Kubernetes

The topology service requires a Kubernetes cluster that aligns with the supported versions listed below.

| Version   |
| --------- |
| 1.17-1.19 |

## Openshift

For Openshift, the topology service requires the cluster to align with these supported versions.

| Version |
| ------- |
| 4.5,4.6 |

## Supported Dell EMC Products

The topology service currently supports the following versions of Dell EMC storage systems.
| Dell EMC Storage Product      |
| ----------------------------- |
| Dell EMC PowerFlex v3.0, v3.5 |

## CSI Driver for Dell EMC Storage

The topology service provides Kubernetes administrators with the topology data related to containerized storage that are provisioned by CSI (Container Storage Interface) Driver for Dell EMC storage products. The topology service requires that the CSI Driver for Dell EMC storage product is deployed in the Kubernetes cluster. Supported CSI driver for Dell EMC storage product are listed below:

| CSI Driver |
| ---------- |
| [CSI Driver for Dell EMC PowerFlex v1.1.5, 1.2.0, 1.2.1](https://github.com/dell/csi-vxflexos) |

## Deploying the Topology Service

The topology service is deployed using Helm.  Usage information and available release versions can be found here: [Karavi Observability Helm chart](https://github.com/dell/helm-charts/tree/main/charts/karavi-observability).

If you built the Docker image and pushed it to a local registry, you can deploy it using the same Helm chart above.  You simply need to override the helm chart value pointing to where the Karavi Topology image lives.  See [Karavi Observability Helm chart](https://github.com/dell/helm-charts/tree/main/charts/karavi-observability) for more details.

__Note:__ The topology service must be deployed first. Once the topology service has been deployed, you can proceed to deploying/configuring the required components below.

## Required Components

The topology service requires the following third party components to be deployed in the same Kubernetes cluster as the karavi-topology service:

* Grafana

It is the user's responsibility to deploy Grafana according to the specifications defined below.

### Grafana

The [Grafana Topology dashboard](../grafana/dashboards) requires Grafana to be deployed in the same Kubernetes cluster as the topology service.
Configure your Grafana instance after successful deployment of the topology service.

| Supported Version | Helm Chart                                                |
| ----------------- | --------------------------------------------------------- |
| 7.3.0-7.3.2       | [Grafana Helm chart](https://github.com/grafana/helm-charts/tree/main/charts/grafana) |

Volume visibility is provided through the topology service.  Topology service exposes an endpoint that can be consumed by Grafana to display CSI driver provisioned volume characteristics correlated with volumes on the storage system.

#### Import the Topology Dashboard

To add the [topology dashboard](../grafana/dashboards) to Grafana, log in and click the + icon in the side menu. Then click Import. From here you can upload the JSON file or paste the JSON text directly into the text area.

#### Configure Plugins

Grafana must be configured with the following pre-requisite plugins:

| Name                   | Additional Information                                                     |
| ---------------------- | -------------------------------------------------------------------------- |
| SimpleJson data source | [SimpleJson data source](https://grafana.com/grafana/plugins/grafana-simple-json-datasource)                 |
| Data Table plugin      | [Data Table plugin](https://grafana.com/grafana/plugins/briangann-datatable-panel/installation) |

#### Configure Topology Data Source

Configure topology service SimpleJson data source at Grafana:

| Setting             | Value                             |
| ------------------- | --------------------------------- |
| Name                | Karavi-Topology |
| URL                 | Access Karavi Topology at https://karavi-topology.*namespace*.svc.cluster.local |
| Skip TLS Verify     | Enabled (If not using CA certificate) |
| With CA Cert        | Enabled (If using CA certificate) |

__Note:__ If you have a CA certificate that can validate the Karavi Topology service certificates, you can provide it to Grafana to have a trusted TLS connection between Grafana and the Karavi topology service. Otherwise, you may enable `Skip TLS Verify`.

## Building Service

__Note:__ Supported in linux only.

If you wish to clone and build karavi-topology, a Linux host is required with the following installed:

| Component       | Version   | Additional Information                                                                                                                     |
| --------------- | --------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Docker          | v19+      | [Docker installation](https://docs.docker.com/engine/install/)                                                                                                    |
| Docker Registry |           | Access to a local/corporate [Docker registry](https://docs.docker.com/registry/)                                                           |
| Golang          | v1.14+    | [Golang installation](https://github.com/travis-ci/gimme)                                                                                                         |
| gosec           |           | [gosec](https://github.com/securego/gosec)                                                                                                          |
| gomock          | v.1.4.3   | [Go Mock](https://github.com/golang/mock)                                                                                                             |
| git             | latest    | [Git installation](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)                                                                              |
| gcc             |           | Run ```sudo apt install build-essential```                                                                                                 |
| kubectl         | 1.16-1.17 | Ensure you copy the kubeconfig file from the Kubernetes cluster to the linux host. [kubectl installation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) |
| Helm            | v.3.3.0   | [Helm installation](https://helm.sh/docs/intro/install/)                                                                                                        |

Once all prerequisites are on the Linux host, follow the steps below to clone and build karavi-topology:

1. Clone the karavi-topology repository: `git clone https://github.com/dell/karavi-topology.git`
1. Set the DOCKER_REPO environment variable to point to the local Docker repository, example: `export DOCKER_REPO=<ip-address>:<port>`
1. In the karavi-topology directory, run the following to build the Docker image called karavi-topology: `make clean build docker`
1. To tag (with the "latest" tag) and push the image to the local Docker repository run the following: `make tag push`

__Note:__ If you are using a local insecure docker registry, ensure you configure the insecure registries on each of the Kubernetes worker nodes to allow access to the local docker repository

## Testing

From the root directory where the repo was cloned, the unit tests can be executed as follows:

```console
make test
```

This will also provide code coverage statistics for the various Go packages.
