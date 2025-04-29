# syntax=docker/dockerfile:1
# SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
# SPDX-License-Identifier: Apache-2.0

################################################################################

ARG SLURM_VERSION=master

################################################################################
ARG PARENT_IMAGE=rockylinux:9
FROM ${PARENT_IMAGE} AS build

SHELL ["bash", "-c"]

ARG SLURM_VERSION
ENV SLURM_VERSION=${SLURM_VERSION}

USER root
WORKDIR /tmp/

COPY patches/ patches/

# Ref: https://slurm.schedmd.com/quickstart_admin.html#rpmbuild
RUN <<EOR
# Download and Build Slurm
set -xeuo pipefail
dnf -q makecache --refresh
dnf -q -y install --setopt='install_weak_deps=False' \
  '@Development' rpm-build 'dnf-command(builddep)' 'dnf-command(config-manager)' epel-release curl-minimal
dnf -q config-manager --enable crb
dnf -q -y upgrade --refresh
## Download
SLURM_DIR="slurm-${SLURM_VERSION}"
mkdir -p $SLURM_DIR
if [ $(curl -s -Iw '%{http_code}' -o /dev/null https://download.schedmd.com/slurm/${SLURM_DIR}.tar.bz2) -eq 200 ]; then
  curl -s -O https://download.schedmd.com/slurm/${SLURM_DIR}.tar.bz2
  tar --strip-components=1 -jxvf ${SLURM_DIR}.tar.bz2 -C $SLURM_DIR
else
  curl -s -L -H "Accept: application/vnd.github+json" -O https://github.com/SchedMD/slurm/archive/${SLURM_VERSION}.tar.gz
  tar --strip-components=1 -zxvf ${SLURM_VERSION}.tar.gz -C $SLURM_DIR
fi
## Patch
find $(pwd)/patches/ -type f -name "*.patch" -print0 | sort -z | xargs -t0r -n1 patch -p1 -d $SLURM_DIR -i
## Build
SLURM_RPM_DIR="$(rpmspec -q ${SLURM_DIR}/slurm.spec 2> /dev/null | grep -Eo "slurm-[[:digit:]]*\.[[:digit:]]*\.[[:digit:]]*(-[[:alnum:]]*)?" | head -n1 | sed -E 's/-[[:alnum:]]+$//g')"
if [ "$SLURM_DIR" != "$SLURM_RPM_DIR" ]; then
  mv $SLURM_DIR $SLURM_RPM_DIR
fi
rm -f ${SLURM_RPM_DIR}.tar.bz2
tar -caf ${SLURM_RPM_DIR}.tar.bz2 $SLURM_RPM_DIR
dnf -q -y builddep -D '_with_slurmrestd 1' -D '_with_jwt 1' -D '_with_yaml 1' -D '_with_numa 1' ${SLURM_RPM_DIR}/slurm.spec
rpmbuild -ta --with slurmrestd --with jwt ${SLURM_RPM_DIR}.tar.bz2 >/dev/null
## Cleanup
dnf clean all
EOR

################################################################################
FROM ${PARENT_IMAGE} AS base

SHELL ["bash", "-c"]

ARG SLURM_VERSION
ENV SLURM_VERSION=${SLURM_VERSION}

# OCI Annotations
# https://github.com/opencontainers/image-spec/blob/v1.0/annotations.md
LABEL \
  org.opencontainers.image.authors="slinky@schedmd.com" \
  org.opencontainers.image.documentation="https://slurm.schedmd.com/documentation.html" \
  org.opencontainers.image.license="GPL-2.0-or-later WITH openssl-exception" \
  org.opencontainers.image.vendor="SchedMD LLC." \
  org.opencontainers.image.version="${SLURM_VERSION}" \
  org.opencontainers.image.source="https://github.com/SlinkyProject/containers"

# HasRequiredLabel requirement from Red Hat OpenShift Software Certification
# https://access.redhat.com/documentation/en-us/red_hat_software_certification/2024/html/red_hat_openshift_software_certification_policy_guide/assembly-requirements-for-container-images_openshift-sw-cert-policy-introduction#con-image-metadata-requirements_openshift-sw-cert-policy-container-images
LABEL \
  vendor="SchedMD LLC." \
  version="${SLURM_VERSION}" \
  release="https://github.com/SlinkyProject/containers"

USER root
WORKDIR /tmp/

ARG SLURM_USER=slurm
ARG SLURM_USER_UID=401
ARG SLURM_USER_GID=401

RUN <<EOR
# Create SlurmUser
set -xeuo pipefail
groupadd --system --gid=${SLURM_USER_GID} ${SLURM_USER}
useradd --system --no-log-init --uid=${SLURM_USER_UID} --gid=${SLURM_USER_GID} --shell=/usr/sbin/nologin ${SLURM_USER}
EOR

RUN <<EOR
# Remove Users
set -xeuo pipefail
for user in $(ls /home/); do
  userdel --remove $user
done
EOR

COPY --from=build /root/rpmbuild/RPMS/**/*.rpm /tmp/

RUN <<EOR
# Install Dependencies
set -xeuo pipefail
dnf -q makecache --refresh
# For: Slurm RPM Dependencies
dnf -q -y install --setopt='install_weak_deps=False' \
  'dnf-command(config-manager)' epel-release
dnf config-manager --enable crb
dnf -q -y upgrade --refresh
# For: Helm Chart
dnf -q -y install --setopt='install_weak_deps=False' \
  rsync gettext iputils
# Miscellaneous
dnf -q -y install --setopt='install_weak_deps=False' \
  ncurses
# Init System
dnf -q -y install --setopt='install_weak_deps=False' \
  supervisor tini
dnf clean all
EOR

ENV KUBECTL_VERSION=1.32

# Ref: https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/#install-using-native-package-management
RUN <<EOR
# Install kubectl
set -xeuo pipefail
dnf -q makecache --refresh
cat <<EOF | tee /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://pkgs.k8s.io/core:/stable:/v${KUBECTL_VERSION}/rpm/
enabled=1
gpgcheck=1
gpgkey=https://pkgs.k8s.io/core:/stable:/v${KUBECTL_VERSION}/rpm/repodata/repomd.xml.key
EOF
dnf -q -y install --setopt='install_weak_deps=False' kubectl
dnf clean all
EOR

## START DEBUG SECTION ###
ARG DEBUG=0
RUN <<EOR
# For: Development and Debugging
set -xeuo pipefail
if [ "$DEBUG" = true ] || [ "$DEBUG" = 1 ]; then
  dnf -q makecache --refresh
  dnf -q -y install sudo
  echo "${SLURM_USER} ALL=(ALL) NOPASSWD: ALL" >/etc/sudoers.d/slurm
  dnf clean all
fi
EOR
### END DEBUG SECTION ###

################################################################################
# SLURM: slurmctld
# BUILD: `docker build --target=slurmctld -t [<registry>/]slurmctld:<tag> .`
################################################################################
FROM base AS slurmctld

SHELL ["bash", "-c"]

# OCI Annotations
# https://github.com/opencontainers/image-spec/blob/v1.0/annotations.md
LABEL \
  org.opencontainers.image.title="Slurm Control Plane" \
  org.opencontainers.image.description="slurmctld - The central management daemon of Slurm" \
  org.opencontainers.image.documentation="https://slurm.schedmd.com/slurmctld.html"

# HasRequiredLabel requirement from Red Hat OpenShift Software Certification
# https://access.redhat.com/documentation/en-us/red_hat_software_certification/2024/html/red_hat_openshift_software_certification_policy_guide/assembly-requirements-for-container-images_openshift-sw-cert-policy-introduction#con-image-metadata-requirements_openshift-sw-cert-policy-container-images
LABEL \
  name="Slurm Control Plane" \
  summary="slurmctld - The central management daemon of Slurm" \
  description="slurmctld - The central management daemon of Slurm"

USER root
WORKDIR /tmp/

# Ref: https://slurm.schedmd.com/quickstart_admin.html#pkg_install
RUN <<EOR
# Install Slurm Packages
set -xeuo pipefail
dnf -q makecache --refresh
dnf -q -y install --setopt='install_weak_deps=False' \
  ./slurm-slurmctld-[0-9]*.rpm \
  ./slurm-[0-9]*.rpm
rm *.rpm && dnf clean all
EOR

COPY files/slurmctld/ /

EXPOSE 6817/tcp
ENTRYPOINT ["entrypoint.sh"]

################################################################################
# SLURM: slurmd
# BUILD: `docker build --target=slurmd -t [<registry>/]slurmd:<tag> .`
################################################################################
FROM base AS slurmd

SHELL ["bash", "-c"]

# OCI Annotations
# https://github.com/opencontainers/image-spec/blob/v1.0/annotations.md
LABEL \
  org.opencontainers.image.title="Slurm Worker Agent" \
  org.opencontainers.image.description="slurmd - The compute node daemon for Slurm" \
  org.opencontainers.image.documentation="https://slurm.schedmd.com/slurmd.html"

# HasRequiredLabel requirement from Red Hat OpenShift Software Certification
# https://access.redhat.com/documentation/en-us/red_hat_software_certification/2024/html/red_hat_openshift_software_certification_policy_guide/assembly-requirements-for-container-images_openshift-sw-cert-policy-introduction#con-image-metadata-requirements_openshift-sw-cert-policy-container-images
LABEL \
  name="Slurm Worker Agent" \
  summary="slurmd - The compute node daemon for Slurm" \
  description="slurmd - The compute node daemon for Slurm"

USER root
WORKDIR /tmp/

# Ref: https://slurm.schedmd.com/quickstart_admin.html#pkg_install
RUN <<EOR
# Install Slurm Packages
set -xeuo pipefail
dnf -q makecache --refresh
dnf -q -y install --setopt='install_weak_deps=False' \
  ./slurm-devel-[0-9]*.rpm \
  ./slurm-libpmi-[0-9]*.rpm \
  ./slurm-pam_slurm-[0-9]*.rpm \
  ./slurm-slurmd-[0-9]*.rpm \
  ./slurm-[0-9]*.rpm
rm *.rpm && dnf clean all
# Configure
cp -v /etc/nsswitch.conf{,.bak}
sed -i -E "s/^passwd:[[:space:]]+/&slurm /g" /etc/nsswitch.conf
sed -i -E "s/^group:[[:space:]]+/&slurm /g" /etc/nsswitch.conf
EOR

COPY files/slurmd/ /

EXPOSE 6818/tcp
ENTRYPOINT ["entrypoint.sh"]

################################################################################
# SLURM: slurmdbd
# BUILD: `docker build --target=slurmdbd -t [<registry>/]slurmdbd:<tag> .`
################################################################################
FROM base AS slurmdbd

SHELL ["bash", "-c"]

# OCI Annotations
# https://github.com/opencontainers/image-spec/blob/v1.0/annotations.md
LABEL \
  org.opencontainers.image.title="Slurm Database Agent" \
  org.opencontainers.image.description="slurmdbd - Slurm Database Daemon" \
  org.opencontainers.image.documentation="https://slurm.schedmd.com/slurmdbd.html"

# HasRequiredLabel requirement from Red Hat OpenShift Software Certification
# https://access.redhat.com/documentation/en-us/red_hat_software_certification/2024/html/red_hat_openshift_software_certification_policy_guide/assembly-requirements-for-container-images_openshift-sw-cert-policy-introduction#con-image-metadata-requirements_openshift-sw-cert-policy-container-images
LABEL \
  name="Slurm Database Agent" \
  summary="slurmdbd - Slurm Database Daemon" \
  description="slurmdbd - Slurm Database Daemon"

USER root
WORKDIR /tmp/

# Ref: https://slurm.schedmd.com/quickstart_admin.html#pkg_install
RUN <<EOR
# Install Slurm Packages
set -xeuo pipefail
dnf -q makecache --refresh
dnf -q -y install --setopt='install_weak_deps=False' \
  ./slurm-slurmdbd-[0-9]*.rpm \
  ./slurm-[0-9]*.rpm
rm *.rpm && dnf clean all
EOR

EXPOSE 6819/tcp
ENTRYPOINT ["tini", "-g", "--", "slurmdbd", "-D"]

################################################################################
# SLURM: slurmrestd
# BUILD: `docker build --target=slurmrestd -t [<registry>/]slurmrestd:<tag> .`
################################################################################
FROM base AS slurmrestd

SHELL ["bash", "-c"]

# OCI Annotations
# https://github.com/opencontainers/image-spec/blob/v1.0/annotations.md
LABEL \
  org.opencontainers.image.title="Slurm REST API Agent" \
  org.opencontainers.image.description="slurmrestd - Interface to Slurm via REST API" \
  org.opencontainers.image.documentation="https://slurm.schedmd.com/slurmrestd.html"

# HasRequiredLabel requirement from Red Hat OpenShift Software Certification
# https://access.redhat.com/documentation/en-us/red_hat_software_certification/2024/html/red_hat_openshift_software_certification_policy_guide/assembly-requirements-for-container-images_openshift-sw-cert-policy-introduction#con-image-metadata-requirements_openshift-sw-cert-policy-container-images
LABEL \
  name="Slurm REST API Agent" \
  summary="slurmrestd - Interface to Slurm via REST API" \
  description="slurmrestd - Interface to Slurm via REST API"

USER root
WORKDIR /tmp/

# Ref: https://slurm.schedmd.com/quickstart_admin.html#pkg_install
RUN <<EOR
# Install Slurm Packages
set -xeuo pipefail
dnf -q makecache --refresh
dnf -q -y install --setopt='install_weak_deps=False' \
  ./slurm-slurmrestd-[0-9]*.rpm \
  ./slurm-[0-9]*.rpm
rm *.rpm && dnf clean all
EOR

EXPOSE 6820/tcp
ENTRYPOINT ["tini", "-g", "--", "slurmrestd"]
CMD ["0.0.0.0:6820"]

################################################################################
# SLURM: sackd
# BUILD: `docker build --target=sackd -t [<registry>/]sackd:<tag> .`
################################################################################
FROM base AS sackd

SHELL ["bash", "-c"]

# OCI Annotations
# https://github.com/opencontainers/image-spec/blob/v1.0/annotations.md
LABEL \
  org.opencontainers.image.title="Slurm Auth/Cred Server" \
  org.opencontainers.image.description="sackd - Slurm Auth and Cred Kiosk Daemon" \
  org.opencontainers.image.documentation="https://slurm.schedmd.com/sackd.html"

# HasRequiredLabel requirement from Red Hat OpenShift Software Certification
# https://access.redhat.com/documentation/en-us/red_hat_software_certification/2024/html/red_hat_openshift_software_certification_policy_guide/assembly-requirements-for-container-images_openshift-sw-cert-policy-introduction#con-image-metadata-requirements_openshift-sw-cert-policy-container-images
LABEL \
  name="Slurm Auth/Cred Server" \
  summary="sackd - Slurm Auth and Cred Kiosk Daemon" \
  description="sackd - Slurm Auth and Cred Kiosk Daemon"

USER root
WORKDIR /tmp/

# Ref: https://slurm.schedmd.com/quickstart_admin.html#pkg_install
RUN <<EOR
# Install Slurm Packages
set -xeuo pipefail
dnf -q makecache --refresh
dnf -q -y install --setopt='install_weak_deps=False' \
  ./slurm-sackd-[0-9]*.rpm \
  ./slurm-[0-9]*.rpm
rm *.rpm && dnf clean all
# Prepare Directories
mkdir -p /run/slurm/
EOR

COPY files/sackd/ /

ENTRYPOINT ["entrypoint.sh"]

################################################################################
# NAME : login
# BUILD: `docker build --target=login -t [<registry>/]login:<tag> .`
################################################################################
FROM sackd AS login

SHELL ["bash", "-c"]

# OCI Annotations
# https://github.com/opencontainers/image-spec/blob/v1.0/annotations.md
LABEL \
  org.opencontainers.image.title="Slurm Login Container" \
  org.opencontainers.image.description="An authenticated environment to submit Slurm workload from." \
  org.opencontainers.image.documentation="https://slurm.schedmd.com/quickstart_admin.html#login"

# HasRequiredLabel requirement from Red Hat OpenShift Software Certification
# https://access.redhat.com/documentation/en-us/red_hat_software_certification/2024/html/red_hat_openshift_software_certification_policy_guide/assembly-requirements-for-container-images_openshift-sw-cert-policy-introduction#con-image-metadata-requirements_openshift-sw-cert-policy-container-images
LABEL \
  name="Slurm Login Container" \
  summary="An authenticated environment to submit Slurm workload from." \
  description="An authenticated environment to submit Slurm workload from."

USER root
WORKDIR /tmp/

# Ref: https://slurm.schedmd.com/quickstart_admin.html#pkg_install
RUN <<EOR
# Install Packages
set -xeuo pipefail
dnf -q makecache --refresh
dnf -q -y install --setopt='install_weak_deps=False' \
  openssh-server \
  dbus-daemon \
  authselect sssd sssd-ad sssd-ldap sssd-dbus
rm *.rpm && dnf clean all
# Configure
mkdir -p /etc/authselect
authselect select sssd with-mkhomedir --force
rm -f /etc/ssh/ssh_host_*
# Remove Packages
dnf -q -y remove \
  authselect
dnf -q -y autoremove
EOR

COPY files/login/ /

EXPOSE 22/tcp
ENTRYPOINT ["entrypoint.sh"]
