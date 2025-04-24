# syntax=docker/dockerfile:1
# SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
# SPDX-License-Identifier: Apache-2.0

################################################################################
FROM golang:1 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY api/ api/
COPY cmd/ cmd/
COPY internal/ internal/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/manager/main.go

################################################################################

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot

# OCI Annotations
# https://github.com/opencontainers/image-spec/blob/v1.0/annotations.md
LABEL org.opencontainers.image.authors="slinky@schedmd.com" \
      org.opencontainers.image.title="Slurm Operator" \
      org.opencontainers.image.description="Kubernetes Operator for Slurm" \
      org.opencontainers.image.documentation="https://github.com/SlinkyProject/slurm-operator" \
      org.opencontainers.image.license="Apache-2.0" \
      org.opencontainers.image.vendor="SchedMD LLC." \
      org.opencontainers.image.version="v0.2.0" \
      org.opencontainers.image.source="https://github.com/SlinkyProject/slurm-operator"

# HasRequiredLabel requirement from Red Hat OpenShift Software Certification
# https://access.redhat.com/documentation/en-us/red_hat_software_certification/2024/html/red_hat_openshift_software_certification_policy_guide/assembly-requirements-for-container-images_openshift-sw-cert-policy-introduction#con-image-metadata-requirements_openshift-sw-cert-policy-container-images
LABEL name="Slurm Operator" \
      summary="Kubernetes Operator for Slurm " \
      description="Kubernetes Operator for Slurm" \
      vendor="SchedMD LLC." \
      version="v0.2.0" \
      release="https://github.com/SlinkyProject/slurm-operator"

WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
