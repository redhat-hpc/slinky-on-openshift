#!/bin/bash
#SBATCH --job-name=mpi_test
#SBATCH --nodes=2
#SBATCH --ntasks-per-node=4
#SBATCH --exclusive

# load Open MPI module
eval `spack load --sh openmpi`

set -x

#mpirun -np 4 bash -c 'echo "Rank: $OMPI_COMM_WORLD_RANK, Hostname: $HOSTNAME"'
# compile the C file
mpicc test_mpi.c -o test_mpi

# run compiled test_mpi.c file
# srun -v -n 12 ./test_mpi

# export OMPI_MCA_btl_tcp_if_exclude=eth0,lo
# mpirun -np 12 -mca btl_base_verbose 100 ./test_mpi
mpirun -np 8 ./test_mpi
