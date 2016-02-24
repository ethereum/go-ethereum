//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
#pragma once
#include "rocksdb/statistics.h"

#include <vector>
#include <atomic>
#include <string>

#include "util/histogram.h"
#include "util/mutexlock.h"
#include "port/likely.h"


namespace rocksdb {

enum TickersInternal : uint32_t {
  INTERNAL_TICKER_ENUM_START = TICKER_ENUM_MAX,
  INTERNAL_TICKER_ENUM_MAX
};

enum HistogramsInternal : uint32_t {
  INTERNAL_HISTOGRAM_START = HISTOGRAM_ENUM_MAX,
  INTERNAL_HISTOGRAM_ENUM_MAX
};


class StatisticsImpl : public Statistics {
 public:
  StatisticsImpl(std::shared_ptr<Statistics> stats,
                 bool enable_internal_stats);
  virtual ~StatisticsImpl();

  virtual uint64_t getTickerCount(uint32_t ticker_type) const override;
  virtual void histogramData(uint32_t histogram_type,
                             HistogramData* const data) const override;
  std::string getHistogramString(uint32_t histogram_type) const override;

  virtual void setTickerCount(uint32_t ticker_type, uint64_t count) override;
  virtual void recordTick(uint32_t ticker_type, uint64_t count) override;
  virtual void measureTime(uint32_t histogram_type, uint64_t value) override;

  virtual std::string ToString() const override;
  virtual bool HistEnabledForType(uint32_t type) const override;

 private:
  std::shared_ptr<Statistics> stats_shared_;
  Statistics* stats_;
  bool enable_internal_stats_;

  struct Ticker {
    Ticker() : value(uint_fast64_t()) {}

    std::atomic_uint_fast64_t value;
    // Pad the structure to make it size of 64 bytes. A plain array of
    // std::atomic_uint_fast64_t results in huge performance degradataion
    // due to false sharing.
    char padding[64 - sizeof(std::atomic_uint_fast64_t)];
  };

  Ticker tickers_[INTERNAL_TICKER_ENUM_MAX] __attribute__((aligned(64)));
  HistogramImpl histograms_[INTERNAL_HISTOGRAM_ENUM_MAX]
      __attribute__((aligned(64)));
};

// Utility functions
inline void MeasureTime(Statistics* statistics, uint32_t histogram_type,
                        uint64_t value) {
  if (statistics) {
    statistics->measureTime(histogram_type, value);
  }
}

inline void RecordTick(Statistics* statistics, uint32_t ticker_type,
                       uint64_t count = 1) {
  if (statistics) {
    statistics->recordTick(ticker_type, count);
  }
}

inline void SetTickerCount(Statistics* statistics, uint32_t ticker_type,
                           uint64_t count) {
  if (statistics) {
    statistics->setTickerCount(ticker_type, count);
  }
}

}
