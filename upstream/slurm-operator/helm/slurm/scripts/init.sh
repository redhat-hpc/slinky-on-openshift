#!/usr/bin/env bash
# SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Assume env contains:
# SLURM_USER - username or UID

function init::common() {
	local dir

	dir=/var/spool/slurmd
	mkdir -p "$dir"
	chown -v "${SLURM_USER}:${SLURM_USER}" "$dir"
	chmod -v 700 "$dir"

	dir=/var/spool/slurmctld
	mkdir -p "$dir"
	chown -v "${SLURM_USER}:${SLURM_USER}" "$dir"
	chmod -v 700 "$dir"
}

function init::slurm() {
	SLURM_MOUNT=/mnt/slurm
	SLURM_DIR=/mnt/etc/slurm

	# Workaround to ephemeral volumes not supporting securityContext
	# https://github.com/kubernetes/kubernetes/issues/81089

	# Copy Slurm config files, secrets, and scripts
	mkdir -p "$SLURM_DIR"
	find "${SLURM_MOUNT}" -type f -name "*.conf" -print0 | xargs -0r cp -vt "${SLURM_DIR}"
	find "${SLURM_MOUNT}" -type f -name "*.key" -print0 | xargs -0r cp -vt "${SLURM_DIR}"
	find "${SLURM_MOUNT}" -type f -regextype posix-extended -regex "^.*/(pro|epi)log-.*$" -print0 | xargs -0r cp -vt "${SLURM_DIR}"
	find "${SLURM_MOUNT}" -type f -regextype posix-extended -regex "^.*/(pro|epi)log-.*$" -print0 | xargs -0r cp -vt "${SLURM_DIR}"

	# Set general permissions and ownership
	find "${SLURM_DIR}" -type f -print0 | xargs -0r chown -v "${SLURM_USER}:${SLURM_USER}"
	find "${SLURM_DIR}" -type f -name "*.conf" -print0 | xargs -0r chmod -v 644
	find "${SLURM_DIR}" -type f -name "*.key" -print0 | xargs -0r chmod -v 600
	find "${SLURM_DIR}" -type f -regextype posix-extended -regex "^.*/(pro|epi)log-.*$" -print0 | xargs -0r chown -v "${SLURM_USER}:${SLURM_USER}"
	find "${SLURM_DIR}" -type f -regextype posix-extended -regex "^.*/(pro|epi)log-.*$" -print0 | xargs -0r chmod -v 755

	# Inject secrets into certain config files
	local dbd_conf="slurmdbd.conf"
	if [[ -f "${SLURM_MOUNT}/${dbd_conf}" ]]; then
		echo "Injecting secrets from environment into: ${dbd_conf}"
		rm -f "${SLURM_DIR}/${dbd_conf}"
		envsubst <"${SLURM_MOUNT}/${dbd_conf}" >"${SLURM_DIR}/${dbd_conf}"
		chown -v "${SLURM_USER}:${SLURM_USER}" "${SLURM_DIR}/${dbd_conf}"
		chmod -v 600 "${SLURM_DIR}/${dbd_conf}"
	fi

	# Display Slurm directory files
	ls -lAF "${SLURM_DIR}"
}

function main() {
	init::common
	init::slurm
}
main
