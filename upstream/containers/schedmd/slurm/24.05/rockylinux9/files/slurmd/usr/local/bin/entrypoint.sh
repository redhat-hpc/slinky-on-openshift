#!/usr/bin/env bash
# SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Additional arguments to pass to slurmd.
export SLURMD_OPTIONS="${SLURMD_OPTIONS:-} $*"

function main() {
	mkdir -p /run/slurm/
	mkdir -p /var/spool/slurmd/

	exec supervisord -c /etc/supervisor/supervisord.conf
}
main
