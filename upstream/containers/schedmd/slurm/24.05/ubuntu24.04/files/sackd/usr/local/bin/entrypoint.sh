#!/usr/bin/env bash
# SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
# SPDX-License-Identifier: Apache-2.0

set -xeuo pipefail

mkdir -p /run/slurm/

export SACKD_OPTIONS="${SACKD_OPTIONS:-} $*"

exec supervisord -c /etc/supervisor/supervisord.conf
