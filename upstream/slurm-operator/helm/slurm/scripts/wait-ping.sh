#!/usr/bin/env bash
# SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Assume env contains:
# HOST - Network host

DELAY="1s"

until ping -c1 "${HOST}"; do
	sleep "${DELAY}"
done
