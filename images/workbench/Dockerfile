ARG PARENT_IMAGE=scratch
FROM ${PARENT_IMAGE} AS base

FROM quay.io/modh/odh-minimal-notebook-container:v3-20250818

USER 0

COPY --from=base /tmp/*rpm /tmp

RUN <<EOR
# Install Slurm Packages
set -xeuo pipefail
dnf -q -y install --setopt='install_weak_deps=False' \
  /tmp/slurm-sackd-[0-9]*.rpm \
  /tmp/slurm-[0-9]*.rpm
rm /tmp/*.rpm && dnf clean all
# Prepare Directories
mkdir -p /run/slurm/
EOR

USER 1001

RUN pip install jupyterlab_slurm
