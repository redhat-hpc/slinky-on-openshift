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

Install glauth for simulating LDAP environment

```
oc apply -f extras/glauth.yaml
```

Install the Slurm Operator deployment and then deploy Slurm

```
helm upgrade -i -n slurm slurm-operator slurm-operator/helm/slurm-operator/ --values helm/values-operator.yaml

helm dependency build slurm-operator/helm/slurm/
helm upgrade -i -n slurm slurm slurm-operator/helm/slurm/ --values helm/values-slurm.yaml
```

## Enable Autoscaling (optional)

https://github.com/SlinkyProject/slurm-operator/blob/main/docs/autoscaling.md

### Install OCP Custom Metrics Autoscaler
- add KedaController

### Enable user workload monitoring

https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html/monitoring/configuring-user-workload-monitoring#enabling-monitoring-for-user-defined-projects_preparing-to-configure-the-monitoring-stack-uwm

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-monitoring-config
  namespace: openshift-monitoring
data:
  config.yaml: |
    enableUserWorkload: true
```

https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html/nodes/automatically-scaling-pods-with-the-custom-metrics-autoscaler-operator#nodes-cma-autoscaling-custom-prometheus-config_nodes-cma-autoscaling-custom-trigger

https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html/monitoring/accessing-metrics#viewing-a-list-of-available-metrics_accessing-metrics-as-a-developer

### Apply scaling objects

```
oc apply -f keda-objects.yaml
```
