# Slinky on OpenShift with Demo Environment

## Deploy Slinky
```
helm install slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator \
  --namespace=slinky --create-namespace --version 0.3.1
```

## Deploy Slurm
```
helm install slurm -n slurm oci://quay.io/slinky-on-openshift/slinky-on-openshift --create-namespace \
  -f https://raw.githubusercontent.com/redhat-na-ssa/slinky-on-openshift/refs/heads/main/helm/values-slurm-all-in-one.yaml
```

## SSH to login node with Web Terminal
```
ssh -p 2222 user1@slurm-login.slurm.svc
```

# Update the GPU partition
```
oc patch -n slurm nodeset slurm-compute-gpu --type=merge -p '{"spec": {"replicas": 2}}'
```
