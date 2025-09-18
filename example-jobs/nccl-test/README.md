# NCCL Test - AllReduce Example

This directory contains examples for running NCCL communication tests on OpenShift using Slinky.

## AllReduce Operation

The AllReduce operation is one of the fundamental collective communication patterns in distributed deep learning. It combines values from all processes and distributes the result back to all processes.

### Visualization of AllReduce

Below is a representation of a 4-node AllReduce operation:

```
                     ┌─────┐     ┌─────┐     ┌─────┐     ┌─────┐
                     │GPU 0│     │GPU 1│     │GPU 2│     │GPU 3│
                     └──┬──┘     └──┬──┘     └──┬──┘     └──┬──┘
                        │           │           │           │
Initial values:         │5          │2          │9          │7
                        │           │           │           │
                     ┌──┼───────────┼───────────┼───────────┼──┐
                     │  │           │           │           │  │
 AllReduce           │  ▼           ▼           ▼           ▼  │
 (sum operation)     │        Distributed Reduction            │
                     │  ┌───────────┐   ┌───────────┐          │
                     │  │           │   │           │          │
                     └──┼───────────┼───┼───────────┼──────────┘
                        │           │   │           │
Final values:           │23         │23 │23         │23
                        │           │   │           │
                        ▼           ▼   ▼           ▼
                     ┌─────┐     ┌─────┐     ┌─────┐     ┌─────┐
                     │GPU 0│     │GPU 1│     │GPU 2│     │GPU 3│
                     └─────┘     └─────┘     └─────┘     └─────┘
```

## Running the NCCL AllReduce Test

You can run the NCCL AllReduce test using the included job script:

```bash
./job-allreduce.sh
```

This will execute the NCCL test on your OpenShift cluster using Slinky.

## NCCL Test Parameters

The NCCL test provides several parameters to customize your tests:

- `nThreads`: Number of threads per process
- `minBytes` and `maxBytes`: Range of message sizes to test
- `numGpus`: Number of GPUs to use
- `check`: Enable correctness checking
- `warmup`: Number of warmup iterations

## Performance Considerations

When running NCCL tests, consider:

1. Network topology (e.g., NVLink, InfiniBand)
2. GPU placement within nodes
3. Buffer sizes and communication patterns
4. Network congestion and other workloads

## Interpreting Results

The NCCL test outputs bandwidth measurements for each message size. Higher bandwidth indicates better performance. The results are typically reported in GB/s.

Example output format:
```
#                                                       out-of-place                       in-place
#       size         count      type   redop     time   algbw   busbw  error     time   algbw   busbw  error
#        (B)    (elements)                       (us)  (GB/s)  (GB/s)            (us)  (GB/s)  (GB/s)
         8192          2048     float     sum    34.9    0.23    0.35  5e-07    35.0    0.23    0.35  5e-07
        16384          4096     float     sum    38.3    0.43    0.64  5e-07    38.4    0.43    0.64  5e-07
        32768          8192     float     sum    41.6    0.79    1.18  5e-07    41.8    0.78    1.18  5e-07
```
