#!/bin/bash

#SBATCH --job-name=hpl
#SBATCH --nodes=3
#SBATCH --exclusive

eval `spack load --sh hpl`

# export OMPI_MCA_btl_tcp_if_exclude=eth0,lo

echo "start $(date)"
mpirun --display-allocation --map-by=node -np 300 xhpl
echo "end $(date)"
