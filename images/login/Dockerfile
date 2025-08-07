ARG PARENT_IMAGE=scratch
FROM ${PARENT_IMAGE}

RUN <<EOR
# Install socat for https
set -xeuo pipefail
dnf -q makecache --refresh
dnf -q -y group install "Development Tools"
dnf -q -y install libtool gcc-gfortran gcc-toolset-12
dnf -q -y install socat oddjob oddjob-mkhomedir \
    python3 procps-ng git vim-enhanced hostname man-db hwloc iproute
dnf clean all

mv /etc/supervisor/supervisord.conf /etc/supervisord.d/slurm.conf
EOR

COPY files/supervisord.conf /etc/supervisor/supervisord.conf
COPY files/socat.conf /etc/supervisord.d/socat.conf
COPY files/start_socat.sh /usr/local/bin/start_socat.sh
