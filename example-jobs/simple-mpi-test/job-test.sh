#!/bin/bash
#SBATCH --job-name=mpi_test
#SBATCH --nodes=2
#SBATCH --ntasks-per-node=4
#SBATCH --exclusive
#SBATCH --time=1:00:00

set -x

# compile the C file
mpicc test_mpi.c -o test_mpi

# run compiled test_mpi.c file
srun --mpi=pmix -n 8 ./test_mpi
