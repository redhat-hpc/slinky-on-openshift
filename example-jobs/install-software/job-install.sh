#!/bin/bash
#SBATCH --job-name=spack
#SBATCH --nodes=1
#SBATCH --exclusive

cd ~

if [ ! -d spack ]; then
    git clone --depth=2 --branch=releases/v1.0 https://github.com/spack/spack.git spack
fi
echo "source ~/spack/share/spack/setup-env.sh" > ~/.bash_profile

source ~/.bash_profile

set +x

spack compiler find
spack compiler find /opt/rh/gcc-toolset-12/root
spack external find --all
spack install openmpi schedulers=slurm
spack install hpl
