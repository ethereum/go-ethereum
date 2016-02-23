//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
#pragma once
#include "rocksdb/perf_context.h"
#include "util/perf_step_timer.h"
#include "util/stop_watch.h"

namespace rocksdb {

#if defined(NPERF_CONTEXT) || defined(IOS_CROSS_COMPILE)

#define PERF_TIMER_GUARD(metric)
#define PERF_TIMER_MEASURE(metric)
#define PERF_TIMER_STOP(metric)
#define PERF_TIMER_START(metric)
#define PERF_COUNTER_ADD(metric, value)

#else

// Stop the timer and update the metric
#define PERF_TIMER_STOP(metric)          \
  perf_step_timer_ ## metric.Stop();

#define PERF_TIMER_START(metric)          \
  perf_step_timer_ ## metric.Start();

// Declare and set start time of the timer
#define PERF_TIMER_GUARD(metric)                                      \
  PerfStepTimer perf_step_timer_ ## metric(&(perf_context.metric));   \
  perf_step_timer_ ## metric.Start();

// Update metric with time elapsed since last START. start time is reset
// to current timestamp.
#define PERF_TIMER_MEASURE(metric)        \
  perf_step_timer_ ## metric.Measure();

// Increase metric value
#define PERF_COUNTER_ADD(metric, value)     \
  perf_context.metric += value;

#endif

}
