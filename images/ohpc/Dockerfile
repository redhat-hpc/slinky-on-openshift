ARG PARENT_IMAGE=scratch
ARG BASE_IMAGE=scratch

FROM ${PARENT_IMAGE} AS base

RUN <<EOR
# Install hpc dev env
set -xeuo pipefail
dnf -q makecache --refresh
dnf -q -y install openssl-devel libevent-devel perl-devel hwloc-devel numactl-devel libcurl-devel python3-devel bzip2-devel \
    python3 python3-pip procps-ng git vim-enhanced hostname man-db hwloc numactl libevent iproute rsync gettext iputils

dnf -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm
# redhat-release >= 9.1 is needed by ohpc-release-3-1.el9.x86_64 but we have 9.0 in stream9
rpm -ivh --nodeps http://repos.openhpc.community/OpenHPC/3/EL_9/x86_64/ohpc-release-3-1.el9.x86_64.rpm
dnf -q -y install lmod-ohpc lmod-defaults-gnu12-openmpi4-ohpc gnu12-compilers-ohpc spack-ohpc examples-ohpc \
  automake-ohpc autoconf-ohpc hwloc-ohpc libtool-ohpc valgrind-ohpc cmake-ohpc \
  pmix-ohpc openmpi4-pmix-gnu12-ohpc openblas-gnu12-ohpc omb-gnu12-openmpi4-ohpc boost-gnu12-openmpi4-ohpc

# get the custom slurm repo for latest slurm
dnf -y copr enable jkincl/openhpc-slurm centos-stream-9-x86_64

# Miscellaneous
dnf -q -y install --setopt='install_weak_deps=False' \
ncurses
# Init System
dnf -q -y install --setopt='install_weak_deps=False' \
supervisor tini

dnf clean all
EOR

SHELL ["/bin/sh", "-lc"]

RUN <<EOR bash -l
# spack install
set -xeuo pipefail

module load spack openblas

# adding spack build cache https://cache.spack.io/tag/v0.23.1/
spack mirror add v0.23.1 https://binaries.spack.io/v0.23.1
spack buildcache keys --install --trust

spack compiler find --scope system
spack external find --scope system --all
spack external find --scope system openmpi
spack external find --scope system openblas

# choosing Haswell-level compatibility (go with v4 for AVX512)
spack install hpl target=x86_64_v3
spack module lmod refresh -y
EOR

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

FROM ${BASE_IMAGE} AS slurmctld

USER root
WORKDIR /tmp/

# Ref: https://slurm.schedmd.com/quickstart_admin.html#pkg_install
RUN <<EOR
# Install Slurm Packages
set -xeuo pipefail
dnf -q -y install --setopt='install_weak_deps=False' \
  slurm-slurmctld-ohpc \
  slurm-ohpc \
&& dnf clean all
EOR

COPY files/slurmctld/ /

EXPOSE 6817/tcp
ENTRYPOINT ["entrypoint.sh"]

FROM ${BASE_IMAGE} AS slurmd

USER root
WORKDIR /tmp/

# Ref: https://slurm.schedmd.com/quickstart_admin.html#pkg_install
RUN <<EOR
# Install Slurm Packages
set -xeuo pipefail
dnf -q -y install --setopt='install_weak_deps=False' \
  slurm-slurmd-ohpc \
  slurm-ohpc \
&& dnf clean all
# Configure
mkdir -p /var/spool/slurmd/
cp -v /etc/nsswitch.conf{,.bak}
sed -i -E "s/^passwd:[[:space:]]+/&slurm /g" /etc/nsswitch.conf
sed -i -E "s/^group:[[:space:]]+/&slurm /g" /etc/nsswitch.conf
EOR

COPY files/slurmd/ /

EXPOSE 6818/tcp
ENTRYPOINT ["entrypoint.sh"]

FROM slurmd AS slurmd-cuda

RUN <<EOR bash -l
# Install CUDA from Spack
set -xeuo pipefail

module load spack
spack install \
  cuda target=x86_64_v3 \
  cudnn target=x86_64_v3
EOR

RUN <<EOR bash -l
# Install NCCL from Spack
set -xeuo pipefail

module load spack
spack install \
  nccl cuda_arch=75 target=x86_64_v3 \
  nccl-tests cuda_arch=75 arch=x86_64_v3
EOR

RUN <<EOR bash -l
# Install PyTorch from Spack
set -xeuo pipefail

module load spack
spack install py-torch target=x86_64_v3
spack module lmod refresh -y
EOR

FROM ${BASE_IMAGE} AS slurmdbd

USER root
WORKDIR /tmp/

RUN <<EOR
# Install Slurm Packages
set -xeuo pipefail
dnf -q -y install --setopt='install_weak_deps=False' \
  slurm-slurmdbd-ohpc \
  slurm-ohpc \
&& dnf clean all
EOR

EXPOSE 6819/tcp
ENTRYPOINT ["tini", "-g", "--", "slurmdbd", "-D"]

FROM ${BASE_IMAGE} AS slurmrestd

USER root
WORKDIR /tmp/

RUN <<EOR
# Install Slurm Packages
set -xeuo pipefail
dnf -q -y install --setopt='install_weak_deps=False' \
  slurm-ohpc-slurmrestd \
  slurm-ohpc \
&& dnf clean all
EOR

EXPOSE 6820/tcp
ENTRYPOINT ["tini", "-g", "--", "slurmrestd"]
CMD ["0.0.0.0:6820"]

FROM ${BASE_IMAGE} AS sackd

USER root
WORKDIR /tmp/

ADD https://dl.k8s.io/release/stable.txt kubernetes_stable.txt

# TODO: change to OCP
# Ref: https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/#install-using-native-package-management
RUN <<EOR
# Install kubectl
set -xeuo pipefail
VERSION="$(cat kubernetes_stable.txt | grep -Eo '^v[0-9]+\.[0-9]+')"
cat <<EOF | tee /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://pkgs.k8s.io/core:/stable:/${VERSION}/rpm/
enabled=1
gpgcheck=1
gpgkey=https://pkgs.k8s.io/core:/stable:/${VERSION}/rpm/repodata/repomd.xml.key
EOF
dnf -q -y install --setopt='install_weak_deps=False' kubectl
rm *.txt && dnf clean all
EOR

RUN <<EOR
# Install Slurm Packages
set -xeuo pipefail
dnf -q -y install --setopt='install_weak_deps=False' \
  slurm-sackd-ohpc \
  slurm-ohpc \
&& dnf clean all

# Prepare Directories
mkdir -p /run/slurm/
EOR

COPY files/sackd/ /

ENTRYPOINT ["entrypoint.sh"]

FROM sackd AS login

USER root
WORKDIR /tmp/

# Ref: https://slurm.schedmd.com/quickstart_admin.html#pkg_install
RUN <<EOR
# Install Packages
set -xeuo pipefail
dnf -q -y install --setopt='install_weak_deps=False' \
  openssh-server \
  dbus-daemon \
  oddjob-mkhomedir \
  authselect sssd sssd-ad sssd-ldap sssd-dbus \
  socat
rm *.rpm && dnf clean all
# Configure
mkdir -p /etc/authselect
authselect select sssd with-mkhomedir --force
rm -f /etc/ssh/ssh_host_*
EOR

COPY files/login/ /

EXPOSE 22/tcp
ENTRYPOINT ["entrypoint.sh"]
