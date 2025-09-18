#!/bin/bash
#SBATCH --job-name=hpl
#SBATCH --nodes=2
#SBATCH --ntasks=4
#SBATCH --exclusive
#SBATCH --time=1:00:00

module load spack hpl

# export OMPI_MCA_btl_tcp_if_exclude=eth0,lo

echo "start $(date)"
srun --mpi=pmix -n 8 xhpl
echo "end $(date)"
