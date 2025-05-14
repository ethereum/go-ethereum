# Report 2: Concurrent Block Flushing in Tracer Processing System

## Section 1: Introduction

Currently, the tracer processing system operates in a linear fashion, requiring the complete processing of a block's code before proceeding to the next one. As a result, computationally expensive operations—such as `proto.Marshal` and base64 encoding, which are essential for flushing a block to the firehose—can introduce significant delays. To mitigate this bottleneck, a potential solution is to leverage concurrency by introducing goroutines. These heavy operations can be offloaded to a separate channel, enabling asynchronous processing. This approach allows the main execution flow to begin processing subsequent blocks while previous ones are being flushed, thereby improving overall throughput and reducing latency. The goal of this report is to outline the solution and provide benchmarking results.

## Section 2: Background

`proto.Marshal`, part of Protocol Buffers (protobuf), serializes structured data into a compact binary format. While efficient in output size, the process of traversing and encoding complex data structures—especially those with deeply nested or repeated fields—can be CPU-intensive. Additionally, memory allocations during marshaling can introduce further overhead.

Similarly, base64 encoding, which transforms binary data into an ASCII string format for transmission or storage, involves non-trivial byte-wise transformations and increases the data size. This added computational cost becomes significant when processing large blocks or high-throughput workloads.

Together, these operations introduce latency in a linear processing pipeline.

## Section 3: Method

### 3.1 Proposed Solution

A critical point in the tracer processing system is the `OnBlockEnd` hook, which invokes the `printBlockToFirehose` method. This method includes computationally expensive operations such as `proto.Marshal` and base64 encoding, which are necessary to serialize and flush the block data to the firehose.


To address this, the proposed solution introduces a concurrency mechanism. Specifically, the user specifies a number of worker goroutines that will be concurrently working through the `FirehoseConfig.ConcurrencyBlockFlushing` configuration. There are three layers to this process:

1. A worker queue channel with a buffer of 100 is created. All `printBlockToFirehose` tasks are enqueued while waiting for a worker to take and process it.
2. A number of workers specified by the configuration will be taking and processing these tasks asynchronously until the data is in a byte format and ready to be sent to stdout. The goroutine will then send that data, along with the block number, to a second channel.
3. A final channel is created to store pairs of (block number, [byte]). This channel allows linear flushing by only allowing the current expected block number to be flushed, while storing the remaining blocks until it is their turn. To achieve this, a block number is stored globally the first time `OnBlockEnd` is called. Therefore, the channel will know the first block number expected to be flushed and will increment the expected number by one each time.

### 3.2 Validation and Performance Metrics

The implementation was validated at multiple levels to ensure correctness, configurability, and performance improvements of the concurrent block flushing mechanism.

1.  **Unit-Level Validation**

    To confirm the functional correctness of the concurrent flushing implementation, a unit test was written that creates and processes 1,000 blocks, flushing each to an `InternalTestingBuffer`. The results were then compared against expected outputs to verify equivalence. The test confirms that the output produced by the concurrent mechanism matches that of the original linear method, thereby validating correctness at the unit level.
2.  **Integration Testing on Battlefield-Ethereum**

    To verify integration within the battlefield-ethereum environment, the feature was exposed via a new configuration flag: `CONCURRENT_BLOCK_FLUSHING`. This flag determines the number of workers that will be processing tasks concurrently. \

    Original (linear flushing):

    ```bash
    ./scripts/run_firehose_geth_dev.sh 3.0 prague
    ```

    Concurrent flushing enabled:

    ```bash
    CONCURRENT_BLOCK_FLUSHING=1 ./scripts/run_firehose_geth_dev.sh 3.0 prague
    ```

    Behavioral differences were observed through log output. In the concurrent mode, log lines such as:

    ```
     "Firehose closing, flushing queued blocks to standard output"
    ```

    appear when the program is interrupted (e.g., via Ctrl + C), indicating that the concurrent flushing logic and cleanup path are active. These lines are absent in the linear configuration, confirming that the switch is functioning as intended.

    Furthermore, when running the integration test suite using:

    ```bash
    pnpm test:fh3.0:geth-dev
    ```

    all tests passed successfully (64 passing), indicating that the concurrent implementation does not introduce regressions in battlefield compatibility.
3.  **Performance Benchmarking**

    To quantify performance differences, benchmarking was conducted using firehose-ethereum. The following command was used for both the baseline and concurrent configurations:

    ```bash
    time geth --vmtrace=firehose \
    --vmtrace.jsonconfig='{"concurrentBlockFlushing":<number_of_workers>}' \
    --synctarget=<last_block_hash> \
    --syncmode=full --holesky --datadir=./geth --db.engine=pebble \
    --state.scheme=path --port=30305 --authrpc.jwtsecret=jwt.txt \
    --authrpc.addr=0.0.0.0 --authrpc.port=9551 --authrpc.vhosts="*" \
    --http --http.addr=0.0.0.0 --http.api=eth,net,web3 --http.port=9545 \
    --http.vhosts="*" --port=40303 --ws.port=9546 --ipcpath=/tmp/geth.ipc > /dev/null
    ```

    The benchmark was conducted in four phases:

    * With `concurrentBlockFlushing: 0`, the node was synced to block 10,000, the data directory (`./geth`) was removed, and then resynced up to block 100,000.
    * The same steps were repeated with `concurrentBlockFlushing: 10`.
    * The same steps were repeated with `concurrentBlockFlushing: 100`.
    * The same steps were repeated with `concurrentBlockFlushing: 1000`.

    The `time` command outputs wall-clock time and system/user CPU usage upon completion, providing a baseline for comparing performance between the linear and concurrent implementations. This methodology enables a controlled, reproducible environment for evaluating the effectiveness of the concurrent block flushing feature.

## Section 4: Analysis

The specifications of the operating system used for testing are as follows:

Model: Macbook Air \
Processor: Apple M1 chip \
Memory: 8 GB

### 4.1 Results

The following outlines the results for metric three:

### Until Block 10,000

#### No Concurrency

- Run 1: 84.43s user, 15.27s system, 61% CPU, 2:41.39 total
- Run 2: 86.76s user, 15.93s system, 64% CPU, 2:40.36 total
- Run 3: 78.30s user, 14.62s system, 55% CPU, 2:47.51 total
- Run 4: 78.29s user, 13.72s system, 55% CPU, 2:44.45 total
- Run 5: 83.29s user, 13.37s system, 59% CPU, 2:43.47 total

#### Concurrency (10 Goroutines)

- Run 1: 68.14s user, 13.85s system, 51% CPU, 2:39.27 total
- Run 2: 70.81s user, 14.10s system, 52% CPU, 2:40.37 total
- Run 3: 64.79s user, 13.65s system, 49% CPU, 2:39.22 total
- Run 4: 64.34s user, 15.18s system, 48% CPU, 2:43.25 total
- Run 5: 70.21s user, 15.58s system, 51% CPU, 2:47.34 total

#### Concurrency (100 Goroutines)

- Run 1: 71.19s user, 16.06s system, 53% CPU, 2:42.77 total
- Run 2: 66.74s user, 12.12s system, 51% CPU, 2:34.47 total
- Run 3: 72.53s user, 16.11s system, 56% CPU, 2:37.50 total
- Run 4: 70.99s user, 14.02s system, 56% CPU, 2:29.46 total
- Run 5: 65.86s user, 13.66s system, 54% CPU, 2:24.97 total

#### Concurrency (1000 Goroutines)

- Run 1: 68.44s user, 14.65s system, 54% CPU, 2:33.83 total
- Run 2: 72.79s user, 15.06s system, 55% CPU, 2:38.76 total
- Run 3: 65.60s user, 13.19s system, 53% CPU, 2:28.36 total
- Run 4: 71.82s user, 15.05s system, 54% CPU, 2:38.10 total
- Run 5: 71.36s user, 14.87s system, 56% CPU, 2:33.32 total

---

### Until Block 100,000

#### No Concurrency

- 331.03s user, 165.76s system, 32% CPU, 25:28.66 total

#### Concurrency (10 Goroutines)

- 383.44s user, 183.39s system, 37% CPU, 25:22.84 total

#### Concurrency (100 Goroutines)

- 399.57s user, 175.51s system, 37% CPU, 25:28.03 total

#### Concurrency (1000 Goroutines)

- 346.83s user, 165.96s system, 34% CPU, 25:02.78 total

---

### Table 1: Result Comparison with and Without Concurrency

#### Block 10,000 Summary

| Goroutines | User Time | System Time | CPU   | Total Time |
|------------|-----------|-------------|-------|------------|
| 0          | 82.214s   | 14.582s      | 58.8% | 2:43.44    |
| 10         | 67.666s   | 14.270s      | 50.2% | 2:41.89    |
| 100        | 69.462s   | 14.394s      | 54.0% | 2:33.83    |
| 1000       | 69.990s   | 14.564s      | 54.4% | 2:34.47    |

#### Block 100,000 Summary

| Goroutines | User Time | System Time | CPU   | Total Time  |
|------------|-----------|-------------|--------|-------------|
| 0          | 331.03s   | 165.76s      | 32.0%  | 25:28.66    |
| 10         | 383.44s   | 183.39s      | 37.0%  | 25:22.84    |
| 100        | 399.57s   | 175.51s      | 37.0%  | 25:28.03    |
| 1000       | 346.83s   | 165.96s      | 34.0%  | 25:02.78    |

---

### 4.2 Discussion

The four parameters analyzed are **user**, **system**, **CPU**, and **total**.

- **System** and **CPU** times do not vary significantly with the implementation of goroutines.
- **User** and **Total** time are generally lower with the use of concurrency for 10,000 blocks.
- However, performance improvement for 100,000 blocks is less conclusive.

This could be due to:
- Single-run variance
- Varying system conditions during execution
- Larger processing overhead over a long duration

More runs and broader statistical analysis may be required to solidify findings.

## Section 5: Conclusion

The introduction of concurrency to the block flushing process demonstrates performance improvements for smaller workloads (up to 10,000 blocks), particularly in reducing user and total processing time. While results for larger workloads (100,000 blocks) show less consistent gains.
