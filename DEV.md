# Dev docs

## Pulling subtrees

```
git subtree pull --squash --prefix=upstream/slurm-operator git@github.com:kincl/slurm-operator.git match-resources
# or
git subtree pull --squash --prefix=upstream/slurm-operator https://github.com/SlinkyProject/slurm-operator.git v0.3.0
```

```
git subtree pull --squash --prefix=upstream/containers https://github.com/SlinkyProject/containers.git main
```

## Diffing helm values

```
diff -u upstream/slurm-operator/helm/slurm-operator/values.yaml helm/values-operator.yaml

diff -u upstream/slurm-operator/helm/slurm/values.yaml helm/values-slurm.yaml
```
