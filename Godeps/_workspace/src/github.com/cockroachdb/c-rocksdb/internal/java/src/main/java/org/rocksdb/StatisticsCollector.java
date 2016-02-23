// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import java.util.List;
import java.util.concurrent.Executors;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.TimeUnit;

/**
 * <p>Helper class to collect DB statistics periodically at a period specified in
 * constructor. Callback function (provided in constructor) is called with
 * every statistics collection.</p>
 *
 * <p>Caller should call start() to start statistics collection. Shutdown() should
 * be called to stop stats collection and should be called before statistics (
 * provided in constructor) reference has been disposed.</p>
 */
public class StatisticsCollector {
  private final List<StatsCollectorInput> _statsCollectorInputList;
  private final ExecutorService _executorService;
  private final int _statsCollectionInterval;
  private volatile boolean _isRunning = true;

  /**
   * Constructor for statistics collector.
   *
   * @param statsCollectorInputList List of statistics collector input.
   * @param statsCollectionIntervalInMilliSeconds Statistics collection time
   *        period (specified in milliseconds).
   */
  public StatisticsCollector(
      final List<StatsCollectorInput> statsCollectorInputList,
      final int statsCollectionIntervalInMilliSeconds) {
    _statsCollectorInputList = statsCollectorInputList;
    _statsCollectionInterval = statsCollectionIntervalInMilliSeconds;

    _executorService = Executors.newSingleThreadExecutor();
  }

  public void start() {
    _executorService.submit(collectStatistics());
  }

  /**
   * Shuts down statistics collector.
   *
   * @param shutdownTimeout Time in milli-seconds to wait for shutdown before
   *        killing the collection process.
   * @throws java.lang.InterruptedException thrown if Threads are interrupted.
   */
  public void shutDown(final int shutdownTimeout) throws InterruptedException {
    _isRunning = false;

    _executorService.shutdownNow();
    // Wait for collectStatistics runnable to finish so that disposal of
    // statistics does not cause any exceptions to be thrown.
    _executorService.awaitTermination(shutdownTimeout, TimeUnit.MILLISECONDS);
  }

  private Runnable collectStatistics() {
    return new Runnable() {

      @Override
      public void run() {
        while (_isRunning) {
          try {
            if(Thread.currentThread().isInterrupted()) {
              break;
            }
            for(StatsCollectorInput statsCollectorInput :
                _statsCollectorInputList) {
              Statistics statistics = statsCollectorInput.getStatistics();
              StatisticsCollectorCallback statsCallback =
                  statsCollectorInput.getCallback();

                // Collect ticker data
              for(TickerType ticker : TickerType.values()) {
                long tickerValue = statistics.getTickerCount(ticker);
                statsCallback.tickerCallback(ticker, tickerValue);
              }

              // Collect histogram data
              for(HistogramType histogramType : HistogramType.values()) {
                HistogramData histogramData =
                    statistics.geHistogramData(histogramType);
                statsCallback.histogramCallback(histogramType, histogramData);
              }

              Thread.sleep(_statsCollectionInterval);
            }
          }
          catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            break;
          }
          catch (Exception e) {
            throw new RuntimeException("Error while calculating statistics", e);
          }
        }
      }
    };
  }
}
