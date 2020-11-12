<!--
Copyright (c) 2020 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->
# Karavi Topology

Karavi Topology is part of the Karavi open source suite of Kubernetes storage enablers for Dell EMC products.

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0%20adopted-ff69b4.svg)](CODE_OF_CONDUCT.md)
[![License](https://img.shields.io/github/license/dell/karavi-topology)](LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/dellemc/karavi-topology)](https://hub.docker.com/r/dellemc/karavi-topology)
[![Go version](https://img.shields.io/github/go-mod/go-version/dell/karavi-topology)](go.mod)
[![Latest Release](https://img.shields.io/github/v/release/dell/karavi-topology?label=latest&style=flat-square)](https://github.com/dell/karavi-topology/releases)

Karavi Topology provides visibility into Dell EMC CSI (Container Storage Interface) driver provisioned volume characteristics in Kubernetes correlated with volumes on the storage system.  

## Supported Dell EMC Products

Karavi Topology currently has support for the following Dell EMC storage systems and associated CSI drivers.

| Dell EMC Storge Product | CSI Driver |
| ----------------------- | ---------- |
| PowerFlex v3.0/3.5 | [CSI Driver for PowerFlex v1.1.5+](https://github.com/dell/csi-vxflexos) |

## Karavi Topology Data

Karavi Topology provides Kubernetes adminstrators with the following information related to containerized storage.

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

## Table of Content
- [Code of Conduct](./docs/CODE_OF_CONDUCT.md)
- Guides
  - [Maintainer Guide](./docs/MAINTAINERS.md)
  - [Committer Guide](./docs/COMMITTER_GUIDE.md)
  - [Contributing Guide](./docs/CONTRIBUTING.md)
  - [Getting Started Guide](./docs/GETTING_STARTED_GUIDE.md)
- [List of Adopters](./ADOPTERS.md)
- [Maintainers](./docs/MAINTAINERS.md)
- [Release Notes](./docs/RELEASE_NOTES.md)
- [Support](#support)
- [Security](./docs/SECURITY.md)
- [About](#about)

## Support

Don’t hesitate to ask! Contact the team and community on [our support](./docs/SUPPORT.md).
Open an issue if you found a bug on [Github Issues](https://github.com/dell/karavi-topology/issues).

## About

Karavi Topology is 100% open source and community-driven. All components are available
under [Apache 2 License](https://www.apache.org/licenses/LICENSE-2.0.html) on
GitHub.
