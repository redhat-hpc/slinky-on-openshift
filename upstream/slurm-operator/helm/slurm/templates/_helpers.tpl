{{- /*
SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
SPDX-License-Identifier: Apache-2.0
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "slurm.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "slurm.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "slurm.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Allow the release namespace to be overridden
*/}}
{{- define "slurm.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "slurm.labels" -}}
helm.sh/chart: {{ include "slurm.chart" . }}
app.kubernetes.io/part-of: slurm
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Common imagePullPolicy
*/}}
{{- define "slurm.imagePullPolicy" -}}
{{ .Values.imagePullPolicy | default "IfNotPresent" }}
{{- end }}

{{/*
Common imagePullSecrets
*/}}
{{- define "slurm.imagePullSecrets" -}}
{{- with .Values.imagePullSecrets -}}
imagePullSecrets:
  {{- . | toYaml | nindent 2 }}
{{- end }}
{{- end }}

{{/*
Determine slurm image repository
*/}}
{{- define "slurm.image.repository" -}}
{{- print "ghcr.io/slinkyproject" -}}
{{- end }}

{{/*
Define image tag
*/}}
{{- define "slurm.image.tag" -}}
{{- printf "%s-ubuntu24.04" .Chart.AppVersion -}}
{{- end }}

{{/*
Parse resources object and convert units.
Ref: https://github.com/helm/helm/issues/11376#issuecomment-1256831105
*/}}
{{- define "resource-quantity" -}}
{{- $value := . -}}
{{- $unit := 1.0 -}}
{{- if typeIs "string" . -}}
  {{- $base2 := dict "Ki" 0x1p10 "Mi" 0x1p20 "Gi" 0x1p30 "Ti" 0x1p40 "Pi" 0x1p50 "Ei" 0x1p60 -}}
  {{- $base10 := dict "m" 1e-3 "k" 1e3 "M" 1e6 "G" 1e9 "T" 1e12 "P" 1e15 "E" 1e18 -}}
  {{- range $k, $v := merge $base2 $base10 -}}
    {{- if hasSuffix $k $ -}}
      {{- $value = trimSuffix $k $ -}}
      {{- $unit = $v -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- mulf (float64 $value) $unit -}}
{{- end -}}

{{/*
Expand a map in a uniform method.
Handles map[string]string and map[string][]string
*/}}
{{- define "expand-map" -}}
{{- range $key, $val := . -}}
  {{- if $val -}}
    {{- printf "%s=%s\n" $key ($val | join ",") -}}
  {{- end -}}
{{- end -}}
{{- end -}}
