# slinky-on-openshift
Pattern for running the [Slurm operator](https://github.com/SlinkyProject/slurm-operator) on OpenShift

Images are built on top of CentOS Stream and are available on [quay.io](https://quay.io/organization/slinky-on-openshift)

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
oc create -f extras/scc.yaml
```


> [!IMPORTANT]
> By default, glauth configuration has commented out default user/password that are easy to guess, modify to suit your use case

Install glauth for simulating LDAP environment

```
oc apply -f extras/glauth.yaml -n slurm
```

Install the Slurm Operator deployment and then deploy Slurm

```
helm upgrade -i -n slurm slurm-operator upstream/slurm-operator/helm/slurm-operator/ --values helm/values-operator.yaml

helm dependency build upstream/slurm-operator/helm/slurm/
helm upgrade -i -n slurm slurm upstream/slurm-operator/helm/slurm/ --values helm/values-slurm.yaml
```

### Add OpenShift Route for SSH

> [!CAUTION]
> Exposing the SSH server in the login pod with easy to guess usernames and passwords is DANGEROUS.
> If you uncommented the simple user1-6 users in glauth configuration then you may be opening up an attack vector to your cluster.

Sometimes it is difficult to open up ports on the cluster for a LoadBalancer or NodePort service. Alternatively we can use the OpenShift Router (L4/L7 LB) to proxy SSH through a TLS tunnel.

> [!NOTE]
> Using TLS to tunnel traffic through the OpenShift Router does reduce the attach surface but it does not remove the threat.

Apply the Route and Service:

```
oc apply -f extras/ssh-route.yaml
oc get route -n slurm
```

Using SSH client and openssl, SSH through OpenShift Route proxy to the login pod. The self signed certificate that openssl complains about is the certificate of the terminating proxy running on the login pod and is fine to disregard.

Specifically, we are not authenticating the connection between the OpenShift router and the backend login pod inside the OpenShift cluster.

```
ssh -o ProxyCommand="openssl s_client -verify_quiet -quiet -connect %h:443 " user1@<route_url>
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
