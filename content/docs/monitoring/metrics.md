---
title: Metrics
sort_key: G
---


Geth includes a variety of optional metrics that can be reported to the user. However, metrics are disabled by default to save on the computational overhead for the average user. Users that choose to see more detailed metrics can enable them using the `--metrics` flag when starting Geth. Some metrics are classed as especially expensive and are only enabled when the `--metrics.expensive` flag is supplied. For example, per-packet network traffic data is considered expensive.

The goal of the Geth metrics system is that - similar to logs - arbitrary metric collections can be added to any part of the code without requiring fancy constructs to analyze them (counter variables, public interfaces, crossing over the APIs, console hooks, etc). Instead, metrics should be "updated" whenever and wherever needed and be automatically collected, surfaced through the APIs, queryable and visualizable for analysis.


## Metric types

Geth's metrics can be classified into four types: meters, timers, counters and guages.

### Meters

Analogous to physical meters (electricity, water, etc), Geth's meters are capable of measuring the *amount* of "things" that pass through and at the *rate* at which they do. A meter doesn't have a specific unit of measure (byte, block, malloc, etc), it just counts arbitrary *events*. At any point in time a meter can report:

* *Total number of events* that passed through the meter
* *Mean throughput rate* of the meter since startup (events / second)
* *Weighted throughput rate* in the last *1*, *5* and *15* minutes (events / second)
  ("weighted" means that recent seconds count more that in older ones*)


### Timers

Timers are extensions of *meters*, the *duration* of an event is collected alongside a log of its occurrence. Similarly to meters, a timer can also measure arbitrary events but each requires a duration to be assigned individually. In addition generating all of the meter report types, a timer also reports:

* *Percentiles (5, 20, 50, 80, 95)*, reporting that some percentage of the events took less than the reported time to execute (*e.g. Percentile 20 = 1.5s would mean that 20% of the measured events took less time than 1.5 seconds to execute; inherently 80%(=100%-20%) took more that 1.5s*)
* Percentile 5: minimum durations (this is as fast as it gets)
* Percentile 50: well behaved samples (boring, just to give an idea)
* Percentile 80: general performance (these should be optimised)
* Percentile 95: worst case outliers (rare, just handle gracefully)

### Counters: 

A counter is a single int64 value that can be incremented and decremented. The current value of the counter can be queried.

### Gauges: 

A gauge is a single int64 value. Its value can increment and decrement - as with a counter - but can also be set arbitrarily.


## Querying metrics

Geth collects metrics if the `--metrics` flag is provided at startup. Those metrics are available via an HTTP server if the `--metrics.addr` flag is also provided. By default the metrics are served at `127.0.0.1:6060/debug/metrics` but a custom IP address can be provided. A custom port can also be provided to the `--metrics.port` flag. More computationally expensive metrics are toggled on or off by providing or omitting the `--metrics.expensive` flag. For example, to serve all metrics at the default address and port:

```
geth <other commands> --metrics --metrics.addr 127.0.0.1 --metrics.expensive
```

Navigating the browser to the given metrics address displays all the available metrics in the form
of JSON data that looks similar to:

```
chain/account/commits.50-percentile:        374072
chain/account/commits.75-percentile:        830356
chain/account/commits.95-percentile:        1783005.3999976
chain/account/commits.99-percentile:        3991806
chain/account/commits.99.999-percentile:    3991806
chain/account/commits.count:                43
chain/account/commits.fifteen-minute:       0.029134344092314267
chain/account/commits.five-minute:          0.029134344092314267

...

```

Any developer is free to add, remove or modify the available metrics as they see fit. The precise list of available metrics is always available by opening the metrics server in the browser.

Geth also supports dumping metrics directly into an influx database. In order to activate this, the `--metrics.influxdb` flag must be provided at startup. The API endpoint,username, password and other influxdb tags can also be provided. The available tags are:

```
--metrics.influxdb.endpoint value      InfluxDB API endpoint to report metrics to (default: "http://localhost:8086")
--metrics.influxdb.database value      InfluxDB database name to push reported metrics to (default: "geth")
--metrics.influxdb.username value      Username to authorize access to the database (default: "test")
--metrics.influxdb.password value      Password to authorize access to the database (default: "test")
--metrics.influxdb.tags value          Comma-separated InfluxDB tags (key/values) attached to all measurements (default: "host=localhost")
--metrics.influxdbv2                   Enable metrics export/push to an external InfluxDB v2 database
--metrics.influxdb.token value         Token to authorize access to the database (v2 only) (default: "test")
--metrics.influxdb.bucket value        InfluxDB bucket name to push reported metrics to (v2 only) (default: "geth")
--metrics.influxdb.organization value  InfluxDB organization name (v2 only) (default: "geth")
```

## Creating and updating metrics

Metrics can be added easily in the Geth source code:

```go
meter := metrics.NewMeter("system/memory/allocs")
timer := metrics.NewTimer("chain/inserts")
```

In order to use the same meter from two different packages without creating dependency cycles, the metrics can be created using `NewOrRegisteredX()` functions. This creates a new meter if no meter with this name is available or returns the existing meter.

```go
meter := metrics.NewOrRegisteredMeter("system/memory/allocs")
timer := metrics.NewOrRegisteredTimer("chain/inserts")
```

The name given to the metric can be any arbitrary string. However, since Geth assumes it to be some meaningful sub-system hierarchy, it should be named accordingly. 

Metrics can then be updated:

```go
meter.Mark(n) // Record the occurrence of `n` events

timer.Update(duration)  // Record an event that took `duration`
timer.UpdateSince(time) // Record an event that started at `time`
timer.Time(function)    // Measure and record the execution of `function`
```

## Summary

Geth can be configured to report metrics to an HTTP server or database. These functions are disabled by default but can be configured by passing the appropriate commands on startup. Users can easily create custom metrics by adding them to the Geth source code, following the instructions on this page. 
