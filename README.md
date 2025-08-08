# slinky-on-openshift
Pattern for running the [Slurm operator](https://github.com/SlinkyProject/slurm-operator) on OpenShift

Images are built on top of CentOS Stream and are available on [quay.io](https://quay.io/organization/slinky-on-openshift)

## Quickstart

### Install cert-manager

Cert-manager needs to be installed before installing Slinky operator

### Install Prerequisites, Slurm Operator, and Slurm

This uses the helm integration with `kustomize` built into the OpenShift Client, `helm` must be installed

```
oc kustomize --enable-helm https://github.com/redhat-na-ssa/slinky-on-openshift/deploy/overlays/quickstart?ref=main | oc apply --server-side -f -
```

This will deploy:

* Base namespace, SecurityContextConstraint, and RoleBindings
* GLAuth for managing LDAP auth
* Expose the login pod SSH server over a TLS route
* Slinky (slurm-operator and CustomResourceDefinitions)
* Slurm deployment

> [!IMPORTANT]
> By default, glauth configuration has `user1` with a password `user1` which is insecure

### Testing

#### SSH to login pod

With the deployment running, now we can SSH into the login pod. This example uses the *OpenShift Route* to create a TLS tunnel that we can use as a proxy to get into our SSH server. This is a non-standard way of using SSH but it does hide our server from traditional port scanning operations. A production deployment would most likely use a Service with `Type=LoadBalancer` instead. Security by obscurity is not a substitute for strong security but for this demo it is sufficient.

```
SSH_ROUTE=$(oc get route -n slurm slurm-login -o jsonpath={.status.ingress[0].host})
ssh -o ProxyCommand="openssl s_client -verify_quiet -connect %h:443 " user1@$SSH_ROUTE
```

>[!INFO]
> or socat: `ssh -o UserKnownHostsFile=/dev/null -o ProxyCommand="socat - OPENSSL:%h:443,verify=0" user1@$SSH_ROUTE`

By default, the password for `user1` is `user1`

The output will include a reference to a insecure and self-signed certificate which can be safely ignored.

```
depth=0 C=US, ST=State, L=City, O=Organization, OU=Unit, CN=localhost
verify error:num=18:self-signed certificate
```

#### Alternatively: Use a port-forward for SSH to login pod

In one terminal window, do the port-forward command:

```
oc -n slurm port-forward deploy/slurm-login 2222:22
```

In a second window, ssh:

```
ssh -o UserKnownHostsFile=/dev/null -p 2222 user1@localhost
```

By default, the password for `user1` is `user1`

>[!NOTE]
> Setting `UserKnownHostsFile` just prevents SSH from caching the server key as localhost, the security of authenticating the server is handled by the port-forward command to the OpenShift API server

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

## Optional: Shared filesystem

Optionally can add a shared filesystem that is mounted at `/home` in the containers

This creates a 100GB PVC called `user-homearea` and a deployment to manage the filesystem for things like creating home areas

```
oc apply -k deploy/homearea
```

> [!NOTE]
> This deploys a PVC with cephfs (which requires OpenShift Data Foundations being deployed and running in the cluster) which may not be for all use cases.

We can deploy a shared filesystem with cephfs and mount it on the login and compute pods:

```
helm upgrade -i -n slurm slurm upstream/slurm-operator/helm/slurm/ --values helm/values-slurm-with-homearea.yaml
```

When used with SSH, the homeareas should be created automatically on successful login.

Other shared homeareas can be used as well, such as NFS which works great if you only have RWO block access: https://github.com/naps-product-sa/openshift-batch/tree/main/storage/simple-nfs


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
