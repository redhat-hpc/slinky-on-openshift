# Slurm Container Images

This directory contains methods to build [Slurm] from [SchedMD].

## Build

```bash
cd $VERSION
export DOCKER_BAKE_REGISTRY=<REGISTRY_PREFIX>
docker buildx bake --print
docker buildx bake
docker buildx bake --push
```

<!-- Links -->

[schedmd]: https://www.schedmd.com/
[slurm]: https://slurm.schedmd.com/overview.html
