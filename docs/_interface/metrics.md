---
title: Metrics
sort_key: C
---

## Meters and Timers

Note, metrics collection is disabled by default in order not to incur reporting overhead for the average user. The flag `--metrics` must therefore be used to enable the basic metrics, and the flag `--metrics.expensive` can be used to enable certain metrics that are deemed 'expensive', from a resource-consumption perspective. Examples of expensive metrics is per-packet network traffic data.

The goal of the Geth metrics system is that - similar to logs - we should be able to add arbitrary metric collection to any part of the code without requiring fancy constructs to analyze them (counter variables, public interfaces, crossing over the APIs, console hooks, etc). Instead, we should just "update" metrics whenever and wherever needed, and have them automatically collected, surfaced through the APIs, queryable and visualizable for analysis.

To that extent, Geth currently implement two types of metrics:
 * **Meters**: Analogous to physical meters (electricity, water, etc), they are capable of measuring the *amount* of "things" that pass through and at the *rate* at which they do that. A meter doesn't have a specific unit of measure (byte, block, malloc, etc), it just counts arbitrary *events*. At any point in time it can report:
   * *Total number of events* that passed through the meter
   * *Mean throughput rate* of the meter since startup (events / second)
   * *Weighted throughput rate* in the last *1*, *5* and *15* minutes (events / second)
     * (*"weighted" means that recent seconds count more that in older ones*)
 * **Timers**: Extension of *meters*, where not only the occurrence of some event is measured, its *duration* is also collected. Similarly to meters, a timer can also measure arbitrary events, but each requires a duration to be assigned individually. Beside **all** the reports a meter can generate, a timer has additionally:
   * *Percentiles (5, 20, 50, 80, 95)*, reporting that some percentage of the events took less than the reported time to execute (*e.g. Percentile 20 = 1.5s would mean that 20% of the measured events took less time than 1.5 seconds to execute; inherently 80%(=100%-20%) took more that 1.5s*)
     * Percentile 5: minimum durations (this is as fast as it gets)
     * Percentile 50: well behaved samples (boring, just to give an idea)
     * Percentile 80: general performance (these should be optimised)
     * Percentile 95: worst case outliers (rare, just handle gracefully)
 * **Counters**: A counter holds a single int64 value that can be incremented and decremented. The current value of the counter can be queried.
 * **Gauges**: A gauge measures a single int64 value. Additionally to increment and decrement the value, as with a counter, the gauge can be set arbitrarely.

## Creating and updating metrics

Metrics can be added easily in the code:

```go
meter := metrics.NewMeter("system/memory/allocs")
timer := metrics.NewTimer("chain/inserts")
```

In order to use the same meter from two different packages without creating dependency cycles, the metrics can be created using `NewOrRegisteredX()` functions.
This creates a new meter if no meter with this name is available or returns the existing meter.

```go
meter := metrics.NewOrRegisteredMeter("system/memory/allocs")
timer := metrics.NewOrRegisteredTimer("chain/inserts")
```

The name can be any arbitrary string, however since Geth assumes it to be some meaningful sub-system hierarchy, please name accordingly. Metrics can then be updated equally simply:

```go
meter.Mark(n) // Record the occurrence of `n` events

timer.Update(duration)  // Record an event that took `duration`
timer.UpdateSince(time) // Record an event that started at `time`
timer.Time(function)    // Measure and record the execution of `function`
```

## Querying metrics

Geth exposes all collected metrics at `127.0.0.1:6060/debug/metrics`. 
For collecting metrics you need to add the `--metrics` flag. In order to start the metric server you need to specify the `--metrics.addr` flag. 

Geth also supports dumping metrics directly into an influx database. In order to activate this, you need to specify the `--metrics.influxdb` flag. You can specify the API endpoint as well as password and username and other influxdb tags.


## Available metrics

Metrics are a debugging tool, with every developer being free to add, remove or modify them as seen fit. As they can change between commits, the exactly available ones can be queried by opening `127.0.0.1:6060/debug/metrics` in your browser. A few however may warrant longevity, so feel free to add to the below list if you feel it's worth a more general audience:

```
 * system/
    * memory/
        * allocs: number of memory allocations made
        * used: amount of memory currently used
        * held: memory allocated on the heap
        * pauses: garbage collector pauses
        * frees: number of memory allocations freed
    * cpu/
        * sysload: time spent by CPU on all processes
        * syswait: time spent waiting on disk i/o
        * procload: time spent by CPU on this process
        * threads: number of threads
        * goroutines: number of goroutines
    * disk/
        * readcount: number of read operations
        * readdata: total number of bytes read
        * readbytes: counter of bytes read
        * writecount: number of write operations
        * writedata: total number of bytes written
        * writebytes: counter of bytes written

* rpc/
    * requests: number of requests
    * success: number of successful requests
    * failure: number of failed requests

* chain/
    * reorg/
        * drop: blocks dropped by reorg
        * add: blocks added by reorg
    * head/
        * block: currently newest block
        * header: currently newest header
        * receipt: currently newest receipt

* p2p/
    * peers: number of connected peers
    * ingress: inbound traffic in bytes
    * egress: outbound traffic in bytes

* txpool/
    * pending: currently pending transactions
    * local: number of transactions send from this node
```
