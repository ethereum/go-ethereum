# Report: Concurrent Block Flushing in Tracer Processing System

## Section 1: Introduction

Currently, the tracer processing system operates in a linear fashion, requiring the complete processing of a block's code before proceeding to the next one. As a result, computationally expensive operations—such as `proto.Marshal` and base64 encoding, which are essential for flushing a block to the firehose—can introduce significant delays. To mitigate this bottleneck, a potential solution is to leverage concurrency by introducing goroutines. These heavy operations can be offloaded to a separate channel, enabling asynchronous processing. This approach allows the main execution flow to begin processing subsequent blocks while previous ones are being flushed, thereby improving overall throughput and reducing latency. The goal of this report is to outline the solution and provide benchmarking results.

## Section 2: Background

`proto.Marshal`, part of Protocol Buffers (protobuf), serializes structured data into a compact binary format. While efficient in output size, the process of traversing and encoding complex data structures—especially those with deeply nested or repeated fields—can be CPU-intensive. Additionally, memory allocations during marshaling can introduce further overhead.

Similarly, base64 encoding, which transforms binary data into an ASCII string format for transmission or storage, involves non-trivial byte-wise transformations and increases the data size. This added computational cost becomes significant when processing large blocks or high-throughput workloads.

Together, these operations introduce latency in a linear processing pipeline.

## Section 3: Method

### 3.1 Proposed Solution

A critical point in the tracer processing system is the `OnBlockEnd` hook, which invokes the `printBlockToFirehose` method. This method includes computationally expensive operations such as `proto.Marshal` and base64 encoding, which are necessary to serialize and flush the block data to the firehose.

To address this, the current solution introduces a worker queue mechanism. Specifically, a single goroutine backed by a channel is used to enqueue and process `printBlockToFirehose` tasks asynchronously. This decouples the expensive flush operations from the main block processing path, allowing the tracer to begin handling the next block immediately after `OnBlockEnd` is invoked. This behavior is controlled by the `FirehoseConfig.ConcurrencyBlockFlushing` flag: when set to `true`, the asynchronous flushing mode is enabled; when set to `false`, the system falls back to the default linear execution of `printBlockToFirehose`.

As part of future work, this model can be extended from a single worker goroutine to multiple concurrent workers. This could further improve throughput by increasing parallelism. However, such an enhancement must address critical challenges, including proper synchronization of shared resources like `output.Buffer` and maintaining the strict block ordering requirement—i.e., block N must be flushed before block N+1 to preserve data consistency.

### 3.2 Validation and Performance Metrics

The implementation was validated at multiple levels to ensure correctness, configurability, and performance improvements of the concurrent block flushing mechanism.

1.  **Unit-Level Validation**

    To confirm the functional correctness of the concurrent flushing implementation, a unit test was written that creates and processes 1,000 blocks, flushing each to an `InternalTestingBuffer`. The results were then compared against expected outputs to verify equivalence. The test confirms that the output produced by the concurrent mechanism matches that of the original linear method, thereby validating correctness at the unit level.
2.  **Integration Testing on Battlefield-Ethereum**

    To verify integration within the battlefield-ethereum environment, the feature was exposed via a new configuration flag: `CONCURRENT_BLOCK_FLUSHING`. This flag determines whether the system uses the default sequential method or the new concurrent implementation. The system can be toggled between these modes with the following commands:

    Original (linear flushing):

    ```bash
    ./scripts/run_firehose_geth_dev.sh 3.0 prague
    ```

    Concurrent flushing enabled:

    ```bash
    CONCURRENT_BLOCK_FLUSHING=true ./scripts/run_firehose_geth_dev.sh 3.0 prague
    ```

    Behavioral differences were observed through log output. In the concurrent mode, log lines such as:

    ```
     "Closing channel: flushing the remaining blocks to firehose"
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
    --vmtrace.jsonconfig='{"concurrentBlockFlushing":<true|false>}' \
    --synctarget=0x7ae82cb3e60f13272a59319a4b617022228227258e18e0c5e7404236d773d2a3 \
    --syncmode=full --holesky --datadir=./geth --db.engine=pebble \
    --state.scheme=path --port=30305 --authrpc.jwtsecret=jwt.txt \
    --authrpc.addr=0.0.0.0 --authrpc.port=9551 --authrpc.vhosts="*" \
    --http --http.addr=0.0.0.0 --http.api=eth,net,web3 --http.port=9545 \
    --http.vhosts="*" --port=40303 --ws.port=9546 --ipcpath=/tmp/geth.ipc > /dev/null
    ```

    The benchmark was conducted in two phases:

    * With `concurrentBlockFlushing: false`, the node was synced to block 10,000, the data directory (`./geth`) was removed, and then resynced up to block 100,000.
    * The same steps were repeated with `concurrentBlockFlushing: true`.

    Note: A channel with a buffer of 100 was created to allow the tasks to queue without blocking the producer. \
    The `time` command outputs wall-clock time and system/user CPU usage upon completion, providing a baseline for comparing performance between the linear and concurrent implementations. This methodology enables a controlled, reproducible environment for evaluating the effectiveness of the concurrent block flushing feature.

## Section 4: Analysis

The specifications of the operating system used for testing are as follows:

Model: Macbook Air \
Processor: Apple M1 chip \
Memory: 8 GB

### 4.1 Results

The following outlines the results for metric three:

**Until Block 10000**

**No concurrency**

Run 1: 71.49s user 15.56s system 56% cpu 2:34.28 total\
Run 2: 69.77s user 15.60s system 54% cpu 2:37.73 total\
Run 3: 68.81s user 15.12s system 53% cpu 2:38.18 total

**Concurrency**

Run 1: 69.61s user 14.97s system 55% cpu 2:32.85 total\
Run 2: 67.94s user 15.13s system 54% cpu 2:33.59 total\
Run 3: 68.60s user 16.37s system 48% cpu 2:54.22 total (Not sure what happened here)

**Until Block 100000**

**No concurrency**

364.19s user 172.14s system 34% cpu 26:13.33 total

**Concurrency**

358.42s user 171.21s system 35% cpu 25:11.97 total

**Table 1: Result comparison with and without concurrency**

|                 | No Concurrency | Concurrency        |
| :-------------- | :------------- | :----------------- |
| **Block 10 000** |                |                    |
| user            | 70.02s         | 68.72s             |
| system          | 15.43s         | 15.49s             |
| cpu             | 54.33%         | 52.33%             |
| total           | 2:36.73        | 2:40.22 (because of last run) |
| **Block 100 000**|                |                    |
| user            | 364.19s        | 358.42s            |
| system          | 172.14s        | 171.21s            |
| cpu             | 34%            | 35%                |
| total           | 26:13.33       | 25:11.97           |

### 4.2 Discussion

Run 3 with concurrency seems to be an outlier. Without it, the general trend would be that every block saves around 0.0006 second, or 0.6 millisecond. \
User seems slightly lower, whereas system and cpu are relatively the same.

## Section 5: Conclusion

The implementation of a single goroutine seems to lead to a decrease in the total time by a factor of 0.6 millisecond per block.
