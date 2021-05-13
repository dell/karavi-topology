<!--
Copyright (c) 2020 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Karavi Topology

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0%20adopted-ff69b4.svg)](docs/CODE_OF_CONDUCT.md)
[![License](https://img.shields.io/github/license/dell/karavi-topology)](LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/dellemc/karavi-topology)](https://hub.docker.com/r/dellemc/karavi-topology)
[![Go version](https://img.shields.io/github/go-mod/go-version/dell/karavi-topology)](go.mod)
[![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/dell/karavi-topology?include_prereleases&label=latest&style=flat-square)](https://github.com/dell/karavi-topology/releases/latest)

Karavi Topology is part of the Karavi Observability storage enabler, which provides Kubernetes administrators standardized approaches for storage observability in Kuberenetes environments.

Karavi Topology provides Kubernetes administrators with the topology data related to containerized storage that is provisioned by CSI (Container Storage Interface) Driver for Dell EMC storage products.

## Topology Data

This project provides Kubernetes administrators with the topology data related to containerized storage. This topology data is visualized using Grafana.

| Field                      | Description                                                                                                                                        |
| -------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Namespace                  | The namespace associated with the persistent volume claim                                                                                          |
| Persistent Volume          | The name of the persistent volume                                                                                                                  |
| Status                     | The status of the persistent volume. "Released" indicating the persistent volume has a claim. "Bound" indicating the persistent volume has a claim |
| Persistent Volume Claim    | The name of the persistent volume claim associated with the persistent volume                                                                      |
| CSI Driver                 | The name of the CSI driver that was responsible for provisioning the volume on the storage system                                                  |
| Created                    | The date the persistent volume was created                                                                                                         |
| Provisioned Size           | The provisioned size of the persistent volume                                                                                                      |
| Storage Class              | The storage class associated with the persistent volume                                                                                            |
| Storage System Volume Name | The name of the volume on the storage system that is associated with the persistent volume                                                         |
| Storage Pool               | The storage pool name the volume/storage class is associated with                                                                                  |
| Storage System             | The storage system ID or IP address the volume is associated with                                                                                  |

## Table of Contents

- [Code of Conduct](https://github.com/dell/karavi-observability/blob/main/docs/CODE_OF_CONDUCT.md)
- Guides
  - [Maintainer Guide](https://github.com/dell/karavi-observability/blob/main/docs/MAINTAINER_GUIDE.md)
  - [Committer Guide](https://github.com/dell/karavi-observability/blob/main/docs/COMMITTER_GUIDE.md)
  - [Contributing Guide](https://github.com/dell/karavi-observability/blob/main/docs/CONTRIBUTING.md)
  - [Getting Started Guide](https://github.com/dell/karavi-observability/blob/main/docs/GETTING_STARTED_GUIDE.md)
  - [Branching Strategy](./docs/BRANCHING.md)
- [List of Adopters](https://github.com/dell/karavi-observability/blob/main/ADOPTERS.md)
- [Maintainers](./docs/MAINTAINERS.md)
- [Support](https://github.com/dell/karavi-observability/blob/main/docs/SUPPORT.md)
- [Security](./docs/SECURITY.md)
- [About](#about)

## Building the Service

If you wish to clone and build the Karavi Topology services, a Linux host is required with the following installed:

| Component       | Version   | Additional Information                                                                                                                     |
| --------------- | --------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Docker          | v19+      | [Docker installation](https://docs.docker.com/engine/install/)                                                                                                    |
| Docker Registry |           | Access to a local/corporate [Docker registry](https://docs.docker.com/registry/)                                                           |
| Golang          | v1.14+    | [Golang installation](https://github.com/travis-ci/gimme)                                                                                                         |
| gosec           |           | [gosec](https://github.com/securego/gosec)                                                                                                          |
| gomock          | v.1.4.3   | [Go Mock](https://github.com/golang/mock)                                                                                                             |
| git             | latest    | [Git installation](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)                                                                              |
| gcc             |           | Run ```sudo apt install build-essential```                                                                                                 |
| kubectl         | 1.18-1.20 | Ensure you copy the kubeconfig file from the Kubernetes cluster to the linux host. [kubectl installation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) |
| Helm            | v.3.3.0   | [Helm installation](https://helm.sh/docs/intro/install/)                                                                                                        |

Once all prerequisites are on the Linux host, follow the steps below to clone and build the metrics service:

1. Clone the repository using the following command: `git clone https://github.com/dell/karavi-topology.git`
1. Set the DOCKER_REPO environment variable to point to the local Docker repository, for example: `export DOCKER_REPO=<ip-address>:<port>`
1. In the karavi-topology directory, run the following command to build the Docker image called karavi-topology: `make clean build docker`
1. To tag (with the "latest" tag) and push the image to the local Docker repository run the following command: `make tag push`

__Note:__ This only supports Linux. If you are using a local insecure docker registry, ensure you configure the insecure registries on each of the Kubernetes worker nodes to allow access to the local docker repository

## Testing Karavi Topology

From the root directory where the repo was cloned, the unit tests can be executed by running the command as follows:

```console
make test
```

This will also provide code coverage statistics for the various Go packages.

## Support

Don’t hesitate to ask! Contact the team and community on [our support page](https://github.com/dell/karavi-observability/blob/main/docs/SUPPORT.md).
Open an issue if you found a bug on [Github Issues](https://github.com/dell/karavi-observability/issues).

## Versioning

This project is adhering to [Semantic Versioning](https://semver.org/).

## About

Karavi Topology is 100% open source and community-driven. All components are available
under [Apache 2 License](https://www.apache.org/licenses/LICENSE-2.0.html) on
GitHub.
