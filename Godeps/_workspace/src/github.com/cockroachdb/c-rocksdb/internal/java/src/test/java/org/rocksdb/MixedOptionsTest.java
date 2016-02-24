// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import org.junit.ClassRule;
import org.junit.Test;

import static org.assertj.core.api.Assertions.assertThat;

public class MixedOptionsTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Test
  public void mixedOptionsTest(){
    // Set a table factory and check the names
    ColumnFamilyOptions cfOptions = new ColumnFamilyOptions();
    cfOptions.setTableFormatConfig(new BlockBasedTableConfig().
        setFilter(new BloomFilter()));
    assertThat(cfOptions.tableFactoryName()).isEqualTo(
        "BlockBasedTable");
    cfOptions.setTableFormatConfig(new PlainTableConfig());
    assertThat(cfOptions.tableFactoryName()).isEqualTo("PlainTable");
    // Initialize a dbOptions object from cf options and
    // db options
    DBOptions dbOptions = new DBOptions();
    Options options = new Options(dbOptions, cfOptions);
    assertThat(options.tableFactoryName()).isEqualTo("PlainTable");
    // Free instances
    options.dispose();
    options = null;
    cfOptions.dispose();
    cfOptions = null;
    dbOptions.dispose();
    dbOptions = null;
    System.gc();
    System.runFinalization();
    // Test Optimize for statements
    cfOptions = new ColumnFamilyOptions();
    cfOptions.optimizeUniversalStyleCompaction();
    cfOptions.optimizeLevelStyleCompaction();
    cfOptions.optimizeForPointLookup(1024);
    options = new Options();
    options.optimizeLevelStyleCompaction();
    options.optimizeLevelStyleCompaction(400);
    options.optimizeUniversalStyleCompaction();
    options.optimizeUniversalStyleCompaction(400);
    options.optimizeForPointLookup(1024);
    options.prepareForBulkLoad();
  }
}
