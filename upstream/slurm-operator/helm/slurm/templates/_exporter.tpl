{{- /*
SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
SPDX-License-Identifier: Apache-2.0
*/}}

{{/*
Define exporter User
*/}}
{{- define "exporter.user" -}}
{{- print "exporter" -}}
{{- end }}

{{/*
Define slurm UID
*/}}
{{- define "exporter.uid" -}}
{{- print "402" -}}
{{- end }}
