ARG PARENT_IMAGE=scratch
FROM ${PARENT_IMAGE}

RUN <<EOR
# Install hpc dev env
set -xeuo pipefail
dnf -q makecache --refresh
dnf -q -y install openssl-devel libevent-devel perl-devel hwloc-devel numactl-devel \
    python3 python3-pip procps-ng git vim-enhanced hostname man-db hwloc numactl libevent iproute

dnf -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm
# redhat-release >= 9.1 is needed by ohpc-release-3-1.el9.x86_64 but we have 9.0 in stream9
rpm -ivh --nodeps http://repos.openhpc.community/OpenHPC/3/EL_9/x86_64/ohpc-release-3-1.el9.x86_64.rpm
dnf -y install lmod-ohpc lmod-defaults-gnu12-openmpi4-ohpc gnu12-compilers-ohpc spack-ohpc examples-ohpc \
  automake-ohpc autoconf-ohpc hwloc-ohpc libtool-ohpc \
  openmpi4-pmix-gnu12-ohpc openblas-gnu12-ohpc omb-gnu12-openmpi4-ohpc

dnf clean all
EOR

SHELL ["/bin/sh", "-lc"]
