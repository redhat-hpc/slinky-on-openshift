# slinky-on-openshift
Pattern for running the [Slurm operator](https://github.com/SlinkyProject/slurm-operator) on OpenShift

## Prerequisites

### Shared filesystem

Can use ODF

This creates a 100GB PVC called `user-homearea` and a deployment to manage the filesystem for things like creating home areas

```
oc apply -f extras/homearea.yaml
```

### Building images

Using OpenShift Pipelines

```
oc apply -k build-pipeline
```

## Install

Create the namespace and add a custom SCC to all service accounts in the namespace

```
oc create ns slurm
oc create -f scc.yaml
```

Install the Slurm Operator deployment and then deploy Slurm

```
helm upgrade -i -n slurm slurm-operator helm/slurm-operator/helm/slurm-operator/ --values helm/values-operator.yaml

helm upgrade -i -n slurm slurm helm/slurm-operator/helm/slurm/ --values helm/values-slurm.yaml
```
