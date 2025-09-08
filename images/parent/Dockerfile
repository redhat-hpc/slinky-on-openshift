ARG PARENT_IMAGE=scratch
FROM ${PARENT_IMAGE}

RUN <<EOR
# Install hpc dev env
set -xeuo pipefail
dnf -q makecache --refresh
dnf -q -y install openssl-devel libevent-devel perl-devel hwloc-devel numactl-devel \
    python3 python3-pip procps-ng git vim-enhanced hostname man-db hwloc numactl libevent iproute

dnf -y install https://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-9.0-30.el9.noarch.rpm \
    https://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-9.0-30.el9.noarch.rpm

dnf -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm
dnf -y install http://repos.openhpc.community/OpenHPC/3/EL_9/x86_64/ohpc-release-3-1.el9.x86_64.rpm
dnf -y install lmod-ohpc lmod-defaults-gnu12-openmpi4-ohpc gnu12-compilers-ohpc spack-ohpc examples-ohpc \
  automake-ohpc autoconf-ohpc hwloc-ohpc libtool-ohpc \
  openmpi4-pmix-gnu12-ohpc openblas-gnu12-ohpc omb-gnu12-openmpi4-ohpc

dnf clean all
EOR

SHELL ["/bin/sh", "-lc"]
