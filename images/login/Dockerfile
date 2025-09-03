ARG PARENT_IMAGE=scratch
FROM ${PARENT_IMAGE}

RUN <<EOR
# Install socat for https
set -xeuo pipefail
dnf -q makecache --refresh
dnf -q -y install socat
dnf clean all

mv /etc/supervisor/supervisord.conf /etc/supervisord.d/slurm.conf
EOR

COPY files/supervisord.conf /etc/supervisor/supervisord.conf
COPY files/socat.conf /etc/supervisord.d/socat.conf
COPY files/start_socat.sh /usr/local/bin/start_socat.sh
