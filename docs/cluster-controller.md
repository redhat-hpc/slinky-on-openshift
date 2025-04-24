# Cluster Controller

## Table of Contents

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [Cluster Controller](#cluster-controller)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Sequence Diagram](#sequence-diagram)

<!-- mdformat-toc end -->

## Overview

This controller is responsible for managing and reconciling the Cluster CRD. A
CRD represents communication to a Slurm cluster via slurmrestd and `auth/jwt`.

This controller uses the [Slurm client] library.

## Sequence Diagram

```mermaid
sequenceDiagram
    autonumber

    actor User as User
    participant KAPI as Kubernetes API
    participant CC as Cluster Controller
    box Operator Internals
        participant SCM as Slurm Client Map
        participant SEC as Slurm Event Channel
    end %% Operator Internals

    note over KAPI: Handle CR Creation
    User->>KAPI: Create Cluster CR
    KAPI-->>CC: Watch Cluster CRD
    CC->>+KAPI: Get referenced secret
    KAPI-->>-CC: Return secret
    create participant SC as Slurm Client
    CC->>+SC: Create Slurm Client for Cluster
    SC-->>-CC: Return Slurm Client Status
    loop Watch Slurm Nodes
        SC->>+SAPI: Get Slurm Nodes
        SAPI-->>-SC: Return Slurm Nodes
        SC->>SEC: Add Event for Cache Delta
    end %% loop Watch Slurm Nodes
    CC->>SCM: Add Slurm Client to Map
    CC->>+SC: Ping Slurm Control Plane
    SC->>+SAPI: Ping Slurm Control Plane
    SAPI-->>-SC: Return Ping
    SC-->>-CC: Return Ping
    CC->>KAPI: Update Cluster CR Status

    note over KAPI: Handle CR Deletion
    User->>KAPI: Delete Cluster CR
    KAPI-->>CC: Watch Cluster CRD
    SCM-->>CC: Lookup Slurm Client
    destroy SC
    CC-)SC: Shutdown Slurm Client
    CC->>SCM: Remove Slurm Client from Map

    participant SAPI as Slurm REST API
```

<!-- Links -->

[slurm client]: https://github.com/SlinkyProject/slurm-client
