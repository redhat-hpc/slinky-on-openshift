ARG PARENT_IMAGE=scratch
ARG BASE_IMAGE=scratch

FROM ${PARENT_IMAGE} AS base

RUN <<EOR
# Install hpc dev env
set -xeuo pipefail
dnf -q makecache --refresh
dnf -q -y install openssl-devel libevent-devel perl-devel hwloc-devel numactl-devel \
    python3 python3-pip procps-ng git vim-enhanced hostname man-db hwloc numactl libevent iproute rsync gettext iputils

dnf -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm
# redhat-release >= 9.1 is needed by ohpc-release-3-1.el9.x86_64 but we have 9.0 in stream9
rpm -ivh --nodeps http://repos.openhpc.community/OpenHPC/3/EL_9/x86_64/ohpc-release-3-1.el9.x86_64.rpm
dnf -y install lmod-ohpc lmod-defaults-gnu12-openmpi4-ohpc gnu12-compilers-ohpc spack-ohpc examples-ohpc \
  automake-ohpc autoconf-ohpc hwloc-ohpc libtool-ohpc \
  pmix-ohpc openmpi4-pmix-gnu12-ohpc openblas-gnu12-ohpc omb-gnu12-openmpi4-ohpc

# Miscellaneous
dnf -q -y install --setopt='install_weak_deps=False' \
ncurses
# Init System
dnf -q -y install --setopt='install_weak_deps=False' \
supervisor tini

dnf clean all
EOR

SHELL ["/bin/sh", "-lc"]


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

RUN <<EOR
#mask systemd-machine-id-commit.service - partial fix for https://bugzilla.redhat.com/show_bug.cgi?id=1472439
systemctl mask systemd-remount-fs.service dev-hugepages.mount sys-fs-fuse-connections.mount systemd-logind.service getty.target console-getty.service systemd-udev-trigger.service systemd-udevd.service systemd-random-seed.service systemd-machine-id-commit.service

cat <<EOF > /etc/systemd/system/journal-console.service
[Unit]
Description=Forward journal logs to console
Documentation=man:journalctl(1)
After=systemd-journald.service
Wants=systemd-journald.service

[Service]
Type=simple
ExecStart=/bin/bash -c '/usr/bin/journalctl --follow --no-pager --output=short > /dev/console'
Restart=always
RestartSec=5
TimeoutStartSec=30
TimeoutStopSec=10

# Prevent the service from being killed during shutdown
KillMode=mixed
KillSignal=SIGTERM

# Run with lower priority
Nice=10

[Install]
WantedBy=multi-user.target
EOF

systemctl enable journal-console.service
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

RUN <<EOR
systemctl enable slurmctld.service

mkdir -p /etc/systemd/system/slurmctld.service.d/
cat <<EOF > /etc/systemd/system/slurmctld.service.d/restart-forever.conf
[Unit]
StartLimitIntervalSec=0
StartLimitBurst=0

[Service]
Restart=always
RestartSec=10s
EOF
EOR

EXPOSE 6817/tcp
ENTRYPOINT ["/sbin/init"]

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

RUN <<EOR
systemctl enable slurmd.service

mkdir -p /etc/systemd/system/slurmd.service.d/
cat <<EOF > /etc/systemd/system/slurmd.service.d/restart-forever.conf
[Unit]
StartLimitIntervalSec=0
StartLimitBurst=0

[Service]
Restart=always
RestartSec=10s
EOF
EOR

EXPOSE 6818/tcp
ENTRYPOINT ["/sbin/init"]

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

RUN <<EOR
systemctl enable slurmdbd.service

mkdir -p /etc/systemd/system/slurmdbd.service.d/
cat <<EOF > /etc/systemd/system/slurmdbd.service.d/restart-forever.conf
[Unit]
StartLimitIntervalSec=0
StartLimitBurst=0

[Service]
Restart=always
RestartSec=10s
EOF
EOR

EXPOSE 6819/tcp
ENTRYPOINT ["/sbin/init"]

FROM ${BASE_IMAGE} AS slurmrestd

USER root
WORKDIR /tmp/

RUN <<EOR
# Install Slurm Packages
set -xeuo pipefail
dnf -q -y install --setopt='install_weak_deps=False' \
  slurm-slurmrestd-ohpc \
  slurm-ohpc \
&& dnf clean all
EOR

RUN <<EOR
systemctl enable slurmrestd.service

mkdir -p /etc/systemd/system/slurmrestd.service.d/
cat <<EOF > /etc/systemd/system/slurmrestd.service.d/restart-forever.conf
[Unit]
StartLimitIntervalSec=0
StartLimitBurst=0

[Service]
Restart=always
RestartSec=10s
EOF
EOR

EXPOSE 6820/tcp
ENTRYPOINT ["/sbin/init"]

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

RUN <<EOR
systemctl enable sackd.service

mkdir -p /etc/systemd/system/sackd.service.d/
cat <<EOF > /etc/systemd/system/sackd.service.d/restart-forever.conf
[Unit]
StartLimitIntervalSec=0
StartLimitBurst=0

[Service]
Restart=always
RestartSec=10s
EOF
EOR

ENTRYPOINT ["/sbin/init"]

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

RUN systemctl enable sshd.service

EXPOSE 22/tcp
ENTRYPOINT ["/sbin/init"]
