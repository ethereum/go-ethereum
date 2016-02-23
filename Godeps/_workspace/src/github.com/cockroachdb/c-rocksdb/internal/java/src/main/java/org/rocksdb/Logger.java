// Copyright (c) 2015, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * <p>This class provides a custom logger functionality
 * in Java which wraps {@code RocksDB} logging facilities.
 * </p>
 *
 * <p>Using this class RocksDB can log with common
 * Java logging APIs like Log4j or Slf4j without keeping
 * database logs in the filesystem.</p>
 *
 * <strong>Performance</strong>
 * <p>There are certain performance penalties using a Java
 * {@code Logger} implementation within production code.
 * </p>
 *
 * <p>
 * A log level can be set using {@link org.rocksdb.Options} or
 * {@link Logger#setInfoLogLevel(InfoLogLevel)}. The set log level
 * influences the underlying native code. Each log message is
 * checked against the set log level and if the log level is more
 * verbose as the set log level, native allocations will be made
 * and data structures are allocated.
 * </p>
 *
 * <p>Every log message which will be emitted by native code will
 * trigger expensive native to Java transitions. So the preferred
 * setting for production use is either
 * {@link org.rocksdb.InfoLogLevel#ERROR_LEVEL} or
 * {@link org.rocksdb.InfoLogLevel#FATAL_LEVEL}.
 * </p>
 */
public abstract class Logger extends RocksObject {

  /**
   * <p>AbstractLogger constructor.</p>
   *
   * <p><strong>Important:</strong> the log level set within
   * the {@link org.rocksdb.Options} instance will be used as
   * maximum log level of RocksDB.</p>
   *
   * @param options {@link org.rocksdb.Options} instance.
   */
  public Logger(final Options options) {
    createNewLoggerOptions(options.nativeHandle_);
  }

  /**
   * <p>AbstractLogger constructor.</p>
   *
   * <p><strong>Important:</strong> the log level set within
   * the {@link org.rocksdb.DBOptions} instance will be used
   * as maximum log level of RocksDB.</p>
   *
   * @param dboptions {@link org.rocksdb.DBOptions} instance.
   */
  public Logger(final DBOptions dboptions) {
    createNewLoggerDbOptions(dboptions.nativeHandle_);
  }

  /**
   * Set {@link org.rocksdb.InfoLogLevel} to AbstractLogger.
   *
   * @param infoLogLevel {@link org.rocksdb.InfoLogLevel} instance.
   */
  public void setInfoLogLevel(final InfoLogLevel infoLogLevel) {
      setInfoLogLevel(nativeHandle_, infoLogLevel.getValue());
  }

  /**
   * Return the loggers log level.
   *
   * @return {@link org.rocksdb.InfoLogLevel} instance.
   */
  public InfoLogLevel infoLogLevel() {
    return InfoLogLevel.getInfoLogLevel(
        infoLogLevel(nativeHandle_));
  }

  protected abstract void log(InfoLogLevel infoLogLevel,
      String logMsg);

  /**
   * Deletes underlying C++ slice pointer.
   * Note that this function should be called only after all
   * RocksDB instances referencing the slice are closed.
   * Otherwise an undefined behavior will occur.
   */
  @Override
  protected void disposeInternal() {
    assert(isInitialized());
    disposeInternal(nativeHandle_);
  }

  protected native void createNewLoggerOptions(
      long options);
  protected native void createNewLoggerDbOptions(
      long dbOptions);
  protected native void setInfoLogLevel(long handle,
      byte infoLogLevel);
  protected native byte infoLogLevel(long handle);
  private native void disposeInternal(long handle);
}
