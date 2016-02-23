// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
package org.rocksdb;

/**
 * MemTableConfig is used to config the internal mem-table of a RocksDB.
 * It is required for each memtable to have one such sub-class to allow
 * Java developers to use it.
 *
 * To make a RocksDB to use a specific MemTable format, its associated
 * MemTableConfig should be properly set and passed into Options
 * via Options.setMemTableFactory() and open the db using that Options.
 *
 * @see Options
 */
public abstract class MemTableConfig {
  /**
   * This function should only be called by Options.setMemTableConfig(),
   * which will create a c++ shared-pointer to the c++ MemTableRepFactory
   * that associated with the Java MemTableConfig.
   *
   * @see Options#setMemTableConfig(MemTableConfig)
   *
   * @return native handle address to native memory table instance.
   */
  abstract protected long newMemTableFactoryHandle();
}
