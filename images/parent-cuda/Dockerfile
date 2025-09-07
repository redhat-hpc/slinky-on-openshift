ARG PARENT_IMAGE=scratch
FROM ${PARENT_IMAGE}

RUN <<EOR
# Install hpc dev env
set -xeuo pipefail
dnf config-manager --add-repo https://developer.download.nvidia.com/compute/cuda/repos/rhel9/x86_64/cuda-rhel9.repo
dnf -q -y install cuda-toolkit-12 cudnn libnccl libnccl-devel
dnf clean all
EOR
