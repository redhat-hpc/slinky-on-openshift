# ChangeLog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

### Fixed

### Changed

### Removed

## v0.3.0

### Added

- Added `NodeSets[].Volumes` to mount additional volumes.
- Added `Tolerations[]` to components.
- Added `controller.persistence.enabled` option.
- Added Slurm chart restapi service configuration options.
- Added login nodes to the Slurm chart.
- Added Slurm chart controller service configuration options.
- Added login node capabilities for chroot.
- Added `cgroup.conf` as configurable and cgroups is enabled by default.
- Added `nodeSelector` options to all Slurm components.
- Added `compute.nodesets[].useResourceLimits` option.
- Added tolerations and affinity to reconfigure and token jobs.
- Added `login.securityContext` option.

### Fixed

- Fixed Slurm chart `app.kubernetes.io/instance` labels.
- Fixed Slurm chart incorrect `imagePullPolicy` being used.
- Fixed Slurm chart not using token job `resources` constraints.
- Fixed Slurm chart not using token job `securityContext` constraints.
- Fixed mariadb subchart `innodb_*` configurations for Slurm.
- Fixed `ArchiveDir` not being a valid value.
- Fixed `slurm.extraSlurmdbdConf` not being used.
- Fixed slurmrestd failing to start when accounting is disabled.
- Fixed responsiveness of container scripts responding to termination signals.
- Fixed login pod resource templating.
- Fixed NodeSet selectorLabels matching multiple NodeSets.
- Fixed NodeSet controller only considering active pods when scaling.
- Fixed login config file permissions.
- Fixed incorrect mount top symlinked `/var/run`, instead of `/run`.
- Fixed regression where slurmd's would not register with all dynamic conf items
  (e.g. Features, Gres, Weight, etc..).
- Fixed operator and operator-webhook not using affinity in values.yaml.
- Fixed nodeset controller failing to apply a rolling update when there are too
  many unhealthy pods.
- Fixed update strategies employing `Recreate` when unnecessary.

### Changed

- Changed webhook to allow updates to `NodeSets[].VolumeClaimTemplates`.
- Changed Slurm daemon `readinessProbe` to use only tcpSocket.
- Changed Slurm chart to consume the mariadb secret directly.
- Changed how Slurm daemon containers log their logfile, no longer duplicated
  stdout streams.
- Changed `slurm.extra*Conf` expression to `map[string]string` or
  `map[string][]string`.
- Changed partition config expression to `map[string]string` or
  `map[string][]string`.
- Changed slurm-operator chart images tags, omit when equal to the default.
- Changed `ttlSecondsAfterFinished` to `helm.sh/hook-delete-policy`.
- Changed `accounting.external` to work with external database.
- Changed fields `existingSecret` to `secretName`.
- Changed `compute.nodesets[].resources` to allow empty resources.
- Changed how `compute.nodeset[]` expresses gres, weight, and features.
- Changed default `login.securityContext`, omit SYS_CHROOT.
- Changed Slurm version to 25.05
- Changed slurm-client to 0.3.0

### Removed

- Removed Slurm daemon `startupProbe`.
- Removed `initContainer` to wait on slurmdbd.
- Removed login pods service link environment variables.
- Removed `controller.enabled` option.
- Removed `{accounting,controller}.replicas` option.
