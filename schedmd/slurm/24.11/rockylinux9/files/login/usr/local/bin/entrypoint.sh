#!/usr/bin/env bash
# SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

mkdir -p /run/sshd/
chmod 0755 /run/sshd/
mkdir -p /run/dbus/
mkdir -p /run/slurm/
rm -f /var/run/dbus/pid

ssh-keygen -A

export SSHD_OPTIONS="${SSHD_OPTIONS:-""}"
export SACKD_OPTIONS="${SACKD_OPTIONS:-""}"
export SSSD_OPTIONS="${SSSD_OPTIONS:-""}"
export DBUS_OPTIONS="${DBUS_OPTIONS:-""}"

exec supervisord -c /etc/supervisor/supervisord.conf
