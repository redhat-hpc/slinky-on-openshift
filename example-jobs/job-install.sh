#!/bin/bash
#SBATCH --job-name=spack
#SBATCH --nodes=1
#SBATCH --exclusive

cd ~

if [ ! -d spack ]; then
    curl -LO https://github.com/spack/spack/archive/refs/tags/v0.23.1.tar.gz
    tar -xf v0.23.1.tar.gz
    rm -rf v0.23.1.tar.gz
    mv spack-0.23.1/ spack
fi
echo "source ~/spack/share/spack/setup-env.sh" > ~/.bash_profile

source ~/.bash_profile

set +x

spack compiler find
spack compiler find /opt/rh/gcc-toolset-12/root
spack external find slurm
spack install openmpi schedulers=slurm
spack install hpl
