{{- /*
SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
SPDX-License-Identifier: Apache-2.0
*/}}

{{/*
Name of the root CA certification.
*/}}
{{- define "slurm-operator.certManager.rootCA" -}}
{{ printf "%s-root-ca" (include "slurm-operator.webhook.name" .) }}
{{- end }}

{{/*
Name of the root issuer.
*/}}
{{- define "slurm-operator.certManager.rootIssuer" -}}
{{ printf "%s-root-issuer" (include "slurm-operator.webhook.name" .) }}
{{- end }}

{{/*
Name of the self signed certification.
*/}}
{{- define "slurm-operator.certManager.selfCert" -}}
{{ printf "%s-self-ca" (include "slurm-operator.webhook.name" .) }}
{{- end }}

{{/*
Name of the self signed issuer.
*/}}
{{- define "slurm-operator.certManager.selfIssuer" -}}
{{ printf "%s-self-issuer" (include "slurm-operator.webhook.name" .) }}
{{- end }}
