#!/usr/bin/env bash
# SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

mkdir -p /run/slurmctld/

export SLURMCTLD_OPTIONS="${SLURMCTLD_OPTIONS:-} $*"

exec supervisord -c /etc/supervisor/supervisord.conf
