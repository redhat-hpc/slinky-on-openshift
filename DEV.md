# Dev docs

## Pulling subtrees

```
git subtree pull --prefix slurm-operator git@github.com:kincl/slurm-operator.git match-resources --squash
```

```
git subtree add --squash --prefix=upstream/containers https://github.com/SlinkyProject/containers.git main
```
