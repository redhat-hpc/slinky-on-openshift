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

echo "spack compiler find"
spack compiler find
echo "spack external find slurm"
spack external find slurm
echo "spack install openmpi"
spack install openmpi schedulers=slurm
echo "spack install hpl"
spack install hpl
