# Kubernetes Operator for Slurm Clusters

<div align="center">

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=for-the-badge)](./LICENSES/Apache-2.0.txt)
[![Tag](https://img.shields.io/github/v/tag/SlinkyProject/slurm-operator?style=for-the-badge)](https://github.com/SlinkyProject/slurm-operator/tags/)
[![Go-Version](https://img.shields.io/github/go-mod/go-version/SlinkyProject/slurm-operator?style=for-the-badge)](./go.mod)
[![Last-Commit](https://img.shields.io/github/last-commit/SlinkyProject/slurm-operator?style=for-the-badge)](https://github.com/SlinkyProject/slurm-operator/commits/)

</div>

Run [Slurm] on [Kubernetes], by [SchedMD]. A [Slinky] project.

## Table of Contents

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [Kubernetes Operator for Slurm Clusters](#kubernetes-operator-for-slurm-clusters)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
    - [Slurm Cluster](#slurm-cluster)
  - [Features](#features)
    - [NodeSets](#nodesets)
    - [Slurm](#slurm)
  - [Limitations](#limitations)
  - [Installation](#installation)
  - [Upgrades](#upgrades)
    - [0.X Releases](#0x-releases)
  - [Documentation](#documentation)
  - [License](#license)

<!-- mdformat-toc end -->

## Overview

[Slurm] and [Kubernetes] are workload managers originally designed for different
kinds of workloads. In broad strokes: Kubernetes excels at scheduling workloads
that typically run for an indefinite amount of time, with potentially vague
resource requirements, on a single node, with loose policy, but can scale its
resource pool infinitely to meet demand; Slurm excels at quickly scheduling
workloads that run for a finite amount of time, with well defined resource
requirements and topology, on multiple nodes, with strict policy, but its
resource pool is known.

This project enables the best of both workload managers, unified on Kubernetes.
It contains a [Kubernetes] operator to deploy and manage certain components of
[Slurm] clusters. This repository implements [custom-controllers] and
[custom resource definitions (CRDs)][crds] designed for the lifecycle (creation,
upgrade, graceful shutdown) of Slurm clusters.

<img src="./docs/assets/architecture-operator.svg" alt="Slurm Operator Architecture" width="100%" height="auto" />

For additional architectural notes, see the [architecture] docs.

### Slurm Cluster

Slurm clusters are very flexible and can be configured in various ways. Our
Slurm helm chart provides a reference implementation that is highly customizable
and tries to expose everything Slurm has to offer.

<img src="./docs/assets/architecture-slurm.svg" alt="Slurm Architecture" width="100%" height="auto" />

For additional information about Slurm, see the [slurm][slurm-docs] docs.

## Features

### NodeSets

A set of homogeneous Slurm nodes (compute nodes, workers), which are delegated
to execute the Slurm workload.

The operator will take into consideration the running workload among Slurm nodes
as it needs to scale-in, upgrade, or otherwise handle node failures. Slurm nodes
will be marked as [drain][slurm-drain] before their eventual termination pending
scale-in or upgrade.

The operator supports NodeSet scale to zero, scaling the resource down to zero
replicas. Hence, any Horizontal Pod Autoscaler (HPA) that also support scale to
zero can be best paired with NodeSets.

### Slurm

Slurm is a full featured HPC workload manager. To highlight a few features:

- [**Accounting**][slurm-accounting]: collect accounting information for every
  job and job step executed.
- [**Partitions**][slurm-arch]: job queues with sets of resources and
  constraints (e.g. job size limit, job time limit, users permitted).
- [**Reservations**][slurm-reservations]: reserve resources for jobs being
  executed by select users and/or select accounts.
- [**Job Dependencies**][slurm-dependency]: defer the start of jobs until the
  specified dependencies have been satisfied.
- [**Job Containers**][slurm-containers]: jobs which run an unprivileged OCI
  container bundle.
- [**MPI**][slurm-mpi]: launch parallel MPI jobs, supports various MPI
  implementations.
- [**Priority**][slurm-priority]: assigns priorities to jobs upon submission and
  on an ongoing basis (e.g. as they age).
- [**Preemption**][slurm-preempt]: stop one or more low-priority jobs to let a
  high-priority job run.
- [**QoS**][slurm-qos]: sets of policies affecting scheduling priority,
  preemption, and resource limits.
- [**Fairshare**][slurm-fairshare]: distribute resources equitably among users
  and accounts based on historical usage.
- [**Node Health Check**][slurm-healthcheck]: periodically check node health via
  script.

## Limitations

- **Kubernetes Version**: >=
  [v1.29](https://kubernetes.io/blog/2023/12/13/kubernetes-v1-29-release/)
- **Slurm Version**: >=
  [24.11](https://www.schedmd.com/slurm-version-24-11-0-is-now-available/)

## Installation

Install the slurm-operator:

```sh
helm install slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator \
  --namespace=slinky --create-namespace
```

Install a Slurm cluster:

```sh
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --namespace=slurm --create-namespace
```

For additional instructions, see the [quickstart] guide.

## Upgrades

### 0.X Releases

Breaking changes may be introduced into newer [CRDs]. To upgrade between these
versions, uninstall all Slinky charts and delete Slinky CRDs, then install the
new release like normal.

```bash
helm --namespace=slurm uninstall slurm
helm --namespace=slinky uninstall slurm-operator
kubectl delete clusters.slinky.slurm.net
kubectl delete nodesets.slinky.slurm.net
```

## Documentation

Project documentation is located in the [docs] directory of this repository.

Slinky documentation can be found [here][slinky-docs].

## License

Copyright (C) SchedMD LLC.

Licensed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0) you
may not use project except in compliance with the license.

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied. See the License for the
specific language governing permissions and limitations under the License.

<!-- links -->

[architecture]: ./docs/architecture.md
[crds]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions
[custom-controllers]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#custom-controllers
[docs]: ./docs/
[kubernetes]: https://kubernetes.io/
[quickstart]: ./docs/quickstart.md
[schedmd]: https://schedmd.com/
[slinky]: https://slinky.ai/
[slinky-docs]: https://slinky.schedmd.com/docs/
[slurm]: https://slurm.schedmd.com/overview.html
[slurm-accounting]: https://slurm.schedmd.com/accounting.html
[slurm-arch]: https://slurm.schedmd.com/quickstart.html#arch
[slurm-containers]: https://slurm.schedmd.com/containers.html
[slurm-dependency]: https://slurm.schedmd.com/sbatch.html#OPT_dependency
[slurm-docs]: ./docs/slurm.md
[slurm-drain]: https://slurm.schedmd.com/scontrol.html#OPT_DRAIN
[slurm-fairshare]: https://slurm.schedmd.com/fair_tree.html
[slurm-healthcheck]: https://slurm.schedmd.com/slurm.conf.html#OPT_HealthCheckProgram
[slurm-mpi]: https://slurm.schedmd.com/mpi_guide.html
[slurm-preempt]: https://slurm.schedmd.com/preempt.html
[slurm-priority]: https://slurm.schedmd.com/priority_multifactor.html
[slurm-qos]: https://slurm.schedmd.com/qos.html
[slurm-reservations]: https://slurm.schedmd.com/reservations.html
