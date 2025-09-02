# slinky-on-openshift
Pattern for running the [Slurm operator](https://github.com/SlinkyProject/slurm-operator) on OpenShift

Images are built on top of CentOS Stream and are available on [quay.io](https://quay.io/organization/slinky-on-openshift)

## Quickstart

### Prerequisites

* `oc` must be installed
* `helm` must be installed

### Install cert-manager

Cert-manager needs to be installed before installing Slinky operator

### Install Slinky, the Slurm Operator

Installing the operator is the [same as upstream](https://github.com/SlinkyProject/slurm-operator/blob/release-0.3/docs/quickstart.md#slurm-operator)

```
helm install slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator --namespace=slinky --create-namespace --version 0.3.1
```

### Install Slurm

Assuming there is a default storage class set:

```
helm install slurm -n slurm oci://quay.io/slinky-on-openshift/slinky-on-openshift --create-namespace
```

This will deploy:

* Base namespace, SecurityContextConstraint, and RoleBindings
* GLAuth for managing LDAP auth
* Expose the login pod SSH server over a TLS route
* Slurm deployment

> [!IMPORTANT]
> By default, glauth configuration has `user1` with a password `user1` which is insecure

### Testing

#### SSH to login pod

With the deployment running, now we can SSH into the login pod. This example uses the *OpenShift Route* to create a TLS tunnel that we can use as a proxy to get into our SSH server. This is a non-standard way of using SSH but it does hide our server from traditional port scanning operations. A production deployment would most likely use a Service with `Type=LoadBalancer` instead. Security by obscurity is not a substitute for strong security but for this demo it is sufficient.

```
SSH_ROUTE=$(oc get route -n slurm slurm-login -o jsonpath={.status.ingress[0].host})
ssh -o StrictHostKeyChecking=no -o ProxyCommand="socat - OPENSSL:%h:443,verify=0" user1@$SSH_ROUTE
```

By default, the password for `user1` is `user1`

#### Alternatively: Use a port-forward for SSH to login pod

In one terminal window, do the port-forward command:

```
oc -n slurm port-forward deploy/slurm-login 2222:22
```

In a second window, ssh:

```
ssh -o StrictHostKeyChecking=no -p 2222 user1@localhost
```

By default, the password for `user1` is `user1`

#### Run a job

We can see that the nodes have checked into the cluster

```
sinfo
```
```
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
debug        up   infinite      2   idle debug-[0-1]
all*         up   infinite      2   idle debug-[0-1]
```

Run a simple command on a node

```
srun -n 1 -t 1:00 hostname
```

## Uninstall Slurm and Slinky

```
helm uninstall slurm -n slurm
helm uninstall slinky -n slinky
```

## Optional: Shared filesystem with NFS

Alternatively to the quickstart, we can deploy Slurm with a shared home area. If you have already deployed Slurm then uninstall the quickstart before installing the example with NFS

### Prerequisite: Storage

The NFS example will consume a RWO volume using the default storage class

### Deploy the NFS CSI provisionier

```
helm repo add nfs-ganesha-server-and-external-provisioner https://kubernetes-sigs.github.io/nfs-ganesha-server-and-external-provisioner/

helm install nfs nfs-ganesha-server-and-external-provisioner/nfs-server-provisioner -n nfs --create-namespace \
  -f https://raw.githubusercontent.com/redhat-na-ssa/slinky-on-openshift/refs/heads/main/helm/values-nfs-provisioner.yaml
```

### Deploy Slurm with a NFS-backed home area

```
helm upgrade -i slurm oci://quay.io/slinky-on-openshift/slinky-on-openshift --reset-values -n slurm \
  -f https://raw.githubusercontent.com/redhat-na-ssa/slinky-on-openshift/refs/heads/main/helm/values-slurm-nfs.yaml
```

When used with SSH, the homeareas should be created automatically on successful login.

## Optional: Enable Autoscaling

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
oc apply -k deploy/keda
```

## Extra: OpenShift Route for SSH

>[!NOTE]
> If you deployed the quickstart then the SSH route has been deployed for you!

> [!CAUTION]
> Exposing the SSH server in the login pod with easy to guess usernames and passwords is DANGEROUS.
> If you uncommented the simple user1-6 users in glauth configuration then you may be opening up an attack vector to your cluster.

Sometimes it is difficult to open up ports on the cluster for a LoadBalancer or NodePort service. Alternatively we can use the OpenShift Router (L4/L7 LB) to proxy SSH through a TLS tunnel.

> [!NOTE]
> Using TLS to tunnel traffic through the OpenShift Router does reduce the attach surface but it does not remove the threat.

```
oc apply -k deploy/ssh
```

Using SSH client and openssl, SSH through OpenShift Route proxy to the login pod. The self signed certificate that openssl complains about is the certificate of the terminating proxy running on the login pod and is fine to disregard.

Specifically, we are not authenticating the connection between the OpenShift router and the backend login pod inside the OpenShift cluster.

```
SSH_ROUTE=$(oc get route -n slurm slurm-login -o jsonpath={.status.ingress[0].host})
ssh -o ProxyCommand="openssl s_client -verify_quiet -connect %h:443 " user1@$SSH_ROUTE
```

## Optional: Building images

By default we can use the images built by GH Actions and hosted on quay.io/slinky-on-openshift

Using OpenShift Pipelines

```
oc apply -k deploy/build-pipeline
```
