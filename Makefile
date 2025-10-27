##@ General

# VERSION defines the project version.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the build target (e.g make build VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.4.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= podman

# BUNDLE_IMG defines the image:tag used for the bundle.
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# CATALOG_IMG defines the image:tag used for the catalog.
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
IMAGE_TAG_BASE ?= quay.io/slinky-on-openshift/slurm-operator

# Bundle and catalog variables
BUNDLE_DIR ?= bundle
CATALOG_DIR ?= catalog
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) --package slurm-operator --channels=alpha --default-channel=alpha

# JSON templates using define with compact formatting
define CATALOGSOURCE_JSON
{"apiVersion":"operators.coreos.com/v1alpha1","kind":"CatalogSource","metadata":{"name":"slinky-catalog","namespace":"openshift-marketplace"},"spec":{"sourceType":"grpc","image":"$(CATALOG_IMG)","displayName":"Slinky Operators","publisher":"Slinky"}}
endef

define SUBSCRIPTION_JSON
{"apiVersion":"operators.coreos.com/v1alpha1","kind":"Subscription","metadata":{"name":"slurm-operator","namespace":"openshift-operators"},"spec":{"channel":"alpha","name":"slurm-operator","source":"slinky-catalog","sourceNamespace":"openshift-marketplace"}}
endef




# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv "$$(echo "$(1)" | sed "s/-$(3)$$//")" $(1) ;\
}
endef

## Tool Binaries
OC ?= oc
OPERATOR_SDK ?= $(LOCALBIN)/operator-sdk-$(OPERATOR_SDK_VERSION)
OPM ?= $(LOCALBIN)/opm-$(OPM_VERSION)
KUSTOMIZE ?= $(LOCALBIN)/kustomize-$(KUSTOMIZE_VERSION)

## Tool Versions
OC_VERSION ?= v4.19.11
OPERATOR_SDK_VERSION ?= v1.39.2
OPM_VERSION ?= v1.44.0
KUSTOMIZE_VERSION ?= v5.4.2

.PHONY: oc
oc: ## Download oc locally if necessary.
ifeq (,$(wildcard $(OC)))
ifeq (, $(shell which oc 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OC)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OC) https://mirror.openshift.com/pub/openshift-v4/$${ARCH}/clients/ocp/$(OC_VERSION)openshift-client-$${OS}.tar.gz ;\
	chmod +x $(OC) ;\
	}
else
OC = $(shell which oc)
endif
endif

.PHONY: operator-sdk
operator-sdk: ## Download operator-sdk locally if necessary.
ifeq (,$(wildcard $(OPERATOR_SDK)))
ifeq (, $(shell which operator-sdk 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPERATOR_SDK)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_$${OS}_$${ARCH} ;\
	chmod +x $(OPERATOR_SDK) ;\
	}
else
OPERATOR_SDK = $(shell which operator-sdk)
endif
endif

.PHONY: opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (, $(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/$(OPM_VERSION)/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
ifeq (,$(wildcard $(KUSTOMIZE)))
ifeq (, $(shell which kustomize 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(KUSTOMIZE)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(KUSTOMIZE).tar.gz https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/$(KUSTOMIZE_VERSION)/kustomize_$(KUSTOMIZE_VERSION)_$${OS}_$${ARCH}.tar.gz ;\
	tar -xzf $(KUSTOMIZE).tar.gz -C $(dir $(KUSTOMIZE)) ;\
	rm $(KUSTOMIZE).tar.gz ;\
	chmod +x $(KUSTOMIZE) ;\
	}
else
KUSTOMIZE = $(shell which kustomize)
endif
endif

##@ Bundle

.PHONY: bundle
bundle: operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	$(OC) kustomize --enable-helm config/manifests | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS) --verbose --version $(VERSION)
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	$(CONTAINER_TOOL) build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(CONTAINER_TOOL) push $(BUNDLE_IMG)

##@ Build with OCP

.PHONY: bundle-build-ocp
bundle-build-ocp: oc bundle ## Build bundle with OpenShift BuildConfig using internal registry
	$(eval IMAGE_TAG_BASE := image-registry.openshift-image-registry.svc:5000/default/slurm-operator)
	$(eval BUNDLE_IMG := $(IMAGE_TAG_BASE)-bundle:v$(VERSION))
	$(OC) new-build --name slurm-operator-bundle --binary --to $(BUNDLE_IMG) || true
	$(OC) patch bc/slurm-operator-bundle -p '{"spec":{"strategy":{"dockerStrategy":{"dockerfilePath":"bundle.Dockerfile"}}}}'
	$(OC) start-build slurm-operator-bundle --from-dir=. --follow
	$(OC) policy add-role-to-group system:image-puller system:authenticated || true


.PHONY: catalog-build-ocp
catalog-build-ocp: oc catalog-build-local ## Build catalog with OpenShift BuildConfig using internal registry
	$(eval IMAGE_TAG_BASE := image-registry.openshift-image-registry.svc:5000/default/slurm-operator)
	$(eval CATALOG_IMG := $(IMAGE_TAG_BASE)-catalog:v$(VERSION))
	$(OC) new-build --name slurm-operator-catalog --binary --to $(CATALOG_IMG) || true
	$(OC) patch bc/slurm-operator-catalog -p '{"spec":{"strategy":{"dockerStrategy":{"dockerfilePath":"catalog.Dockerfile"}}}}'
	$(OC) start-build slurm-operator-catalog --from-dir=. --follow
	$(OC) policy add-role-to-group system:image-puller system:authenticated || true

.PHONY: build-ocp-all
build-ocp-all: bundle-build-ocp catalog-build-ocp ## build bundle and catalog with OCP

.PHONY: clean-ocp-builds
clean-ocp-builds: ## Clean up OpenShift BuildConfigs
	$(OC) delete bc/slurm-operator-bundle --ignore-not-found=true
	$(OC) delete bc/slurm-operator-catalog --ignore-not-found=true

.PHONY: clean-ocp-images
clean-ocp-images: ## Clean up OpenShift ImageStreams
	$(OC) delete is/slurm-operator-bundle --ignore-not-found=true
	$(OC) delete is/slurm-operator-catalog --ignore-not-found=true

.PHONY: clean-ocp-all
clean-ocp-all: clean-ocp-builds clean-ocp-images ## Clean up OpenShift configs

##@ Catalog
.PHONY: catalog-build-local
catalog-local: opm bundle ## Build a catalog image from local bundle directory.
	rm -rf $(CATALOG_DIR) $(CATALOG_DIR).Dockerfile
	mkdir -p $(CATALOG_DIR)
	$(OPM) init slurm-operator --default-channel=alpha --description=README.md --output=yaml > $(CATALOG_DIR)/index.yaml
	echo "---" >> $(CATALOG_DIR)/index.yaml
	$(OPM) render $(BUNDLE_DIR) --output=yaml >> $(CATALOG_DIR)/index.yaml
	echo "---" >> $(CATALOG_DIR)/index.yaml
	echo "schema: olm.channel" >> $(CATALOG_DIR)/index.yaml
	echo "package: slurm-operator" >> $(CATALOG_DIR)/index.yaml
	echo "name: alpha" >> $(CATALOG_DIR)/index.yaml
	echo "entries:" >> $(CATALOG_DIR)/index.yaml
	echo "- name: slurm-operator.v$(VERSION)" >> $(CATALOG_DIR)/index.yaml
	$(OPM) validate $(CATALOG_DIR)
	$(OPM) generate dockerfile $(CATALOG_DIR)

.PHONY: catalog-build-local
catalog-build-local: opm bundle catalog-local ## Build a catalog image from local bundle directory.
	$(CONTAINER_TOOL) build -f $(CATALOG_DIR).Dockerfile -t $(CATALOG_IMG) .

.PHONY: catalog-build
catalog-build: opm bundle-push ## Build a catalog image from pushed bundle image.
	rm -rf $(CATALOG_DIR) $(CATALOG_DIR).Dockerfile
	mkdir -p $(CATALOG_DIR)
	echo "Slurm operator for managing Slurm clusters on OpenShift" > $(CATALOG_DIR)/description.md
	$(OPM) init slurm-operator --default-channel=alpha --description=$(CATALOG_DIR)/description.md --output=yaml > $(CATALOG_DIR)/index.yaml
	echo "---" >> $(CATALOG_DIR)/index.yaml
	$(OPM) render $(BUNDLE_IMG) --output=yaml >> $(CATALOG_DIR)/index.yaml
	echo "---" >> $(CATALOG_DIR)/index.yaml
	echo "schema: olm.channel" >> $(CATALOG_DIR)/index.yaml
	echo "package: slurm-operator" >> $(CATALOG_DIR)/index.yaml
	echo "name: alpha" >> $(CATALOG_DIR)/index.yaml
	echo "entries:" >> $(CATALOG_DIR)/index.yaml
	echo "- name: slurm-operator.v$(VERSION)" >> $(CATALOG_DIR)/index.yaml
	$(OPM) validate $(CATALOG_DIR)
	$(OPM) generate dockerfile $(CATALOG_DIR)
	$(CONTAINER_TOOL) build -f $(CATALOG_DIR).Dockerfile -t $(CATALOG_IMG) .

.PHONY: catalog-push
catalog-push: ## Push the catalog image.
	$(CONTAINER_TOOL) push $(CATALOG_IMG)

.PHONY: catalog-deploy
catalog-deploy: ## Deploy the catalog to the cluster.
	@echo "Creating CatalogSource..."
	@echo '$(CATALOGSOURCE_JSON)' | oc apply -f -

.PHONY: catalog-clean
catalog-clean: ## Remove the catalog from the cluster.
	@echo "Removing CatalogSource..."
	@oc delete catalogsource slinky-catalog -n openshift-marketplace --ignore-not-found=true

.PHONY: bundle-all
bundle-all: bundle bundle-build bundle-push ## Generate, build and push the bundle image.

.PHONY: catalog-all-local
catalog-all-local: catalog-build-local catalog-push ## Build and push the catalog image from local bundle.

.PHONY: catalog-all
catalog-all: catalog-build catalog-push ## Build and push the catalog image from registry bundle.

.PHONY: operator-install
operator-install: ## Create subscription to install the operator from catalog.
	@echo "Creating Subscription..."
	@echo '$(SUBSCRIPTION_JSON)' | oc apply -f -

.PHONY: operator-uninstall
operator-uninstall: ## Uninstall the operator.
	@echo "Removing Subscription..."
	@oc delete subscription slurm-operator -n openshift-operators --ignore-not-found=true
	@echo "Removing ClusterServiceVersion..."
	@oc delete csv -l operators.coreos.com/slurm-operator.openshift-operators --ignore-not-found=true

##@ Cleanup

.PHONY: bundle-clean-fs
bundle-clean-fs: ## Clean up bundle directory.
	rm -rf bundle bundle.Dockerfile

.PHONY: catalog-clean-fs
catalog-clean-fs: ## Clean up catalog directory.
	rm -rf $(CATALOG_DIR)

.PHONY: clean-all
clean-all: bundle-clean-fs catalog-clean-fs ## Clean up all generated directories.
	rm -rf $(LOCALBIN)
