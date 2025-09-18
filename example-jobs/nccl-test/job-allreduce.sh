#!/bin/bash
#SBATCH -p gpu
#SBATCH -N 2
#SBATCH -n 4
#SBATCH --time=1:00:00

module load nccl-tests

srun --mpi=pmix -N 2 -n 2 all_reduce_perf -g 1 -b 8 -e 1G -f 2
