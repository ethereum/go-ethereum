// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

public enum HistogramType {
  DB_GET(0),
  DB_WRITE(1),
  COMPACTION_TIME(2),
  TABLE_SYNC_MICROS(3),
  COMPACTION_OUTFILE_SYNC_MICROS(4),
  WAL_FILE_SYNC_MICROS(5),
  MANIFEST_FILE_SYNC_MICROS(6),
  // TIME SPENT IN IO DURING TABLE OPEN
  TABLE_OPEN_IO_MICROS(7),
  DB_MULTIGET(8),
  READ_BLOCK_COMPACTION_MICROS(9),
  READ_BLOCK_GET_MICROS(10),
  WRITE_RAW_BLOCK_MICROS(11),
  STALL_L0_SLOWDOWN_COUNT(12),
  STALL_MEMTABLE_COMPACTION_COUNT(13),
  STALL_L0_NUM_FILES_COUNT(14),
  HARD_RATE_LIMIT_DELAY_COUNT(15),
  SOFT_RATE_LIMIT_DELAY_COUNT(16),
  NUM_FILES_IN_SINGLE_COMPACTION(17),
  DB_SEEK(18),
  WRITE_STALL(19);

  private final int value_;

  private HistogramType(int value) {
    value_ = value;
  }

  public int getValue() {
    return value_;
  }
}
