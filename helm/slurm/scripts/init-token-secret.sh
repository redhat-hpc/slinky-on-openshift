#!/usr/bin/env bash
# SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Assume env contains:
# SLURM_USER - username or UID
# JWT_SECRET - jwt key secret name
# TOKEN_SECRET - token secret name

function token::lookup() {
	if kubectl get secret "$TOKEN_SECRET"; then
		echo "Secret '$TOKEN_SECRET' already exists. Done."
		exit 0
	fi
}

function token::generate() {
	local token

	token="$(scontrol token username="$SLURM_USER" lifespan=infinite)"
	token="${token/SLURM_JWT=/}"
	echo "${token}"
}

function token::save() {
	local token="${1-}"
	local token_encoded
	local jwt_secret_uid

	if [[ -z $token ]]; then
		echo "Input token cannot be empty!"
		return 1
	fi

	token_encoded="$(echo -n "$token" | base64 -w 0)"

	jwt_secret_uid="$(kubectl get secret "$JWT_SECRET" -o jsonpath="{.metadata.uid}")"

	kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: $TOKEN_SECRET
  ownerReferences:
    - apiVersion: v1
      kind: Secret
      name: $JWT_SECRET
      uid: $jwt_secret_uid
      blockOwnerDeletion: true
      controller: true
type: Opaque
data:
  auth-token: $token_encoded
EOF
}

function slurm::wait_for_ping() {
	# wait for controller to be up first
	until scontrol ping >/dev/null; do
		sleep 5
	done
}

function main() {
	token::lookup
	slurm::wait_for_ping
	token::save "$(token::generate)"
}
main
