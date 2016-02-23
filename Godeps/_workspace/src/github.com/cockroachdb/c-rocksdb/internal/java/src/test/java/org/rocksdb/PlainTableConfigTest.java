// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import org.junit.ClassRule;
import org.junit.Test;

import static org.assertj.core.api.Assertions.assertThat;

public class PlainTableConfigTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Test
  public void keySize() {
    PlainTableConfig plainTableConfig = new PlainTableConfig();
    plainTableConfig.setKeySize(5);
    assertThat(plainTableConfig.keySize()).
        isEqualTo(5);
  }

  @Test
  public void bloomBitsPerKey() {
    PlainTableConfig plainTableConfig = new PlainTableConfig();
    plainTableConfig.setBloomBitsPerKey(11);
    assertThat(plainTableConfig.bloomBitsPerKey()).
        isEqualTo(11);
  }

  @Test
  public void hashTableRatio() {
    PlainTableConfig plainTableConfig = new PlainTableConfig();
    plainTableConfig.setHashTableRatio(0.95);
    assertThat(plainTableConfig.hashTableRatio()).
        isEqualTo(0.95);
  }

  @Test
  public void indexSparseness() {
    PlainTableConfig plainTableConfig = new PlainTableConfig();
    plainTableConfig.setIndexSparseness(18);
    assertThat(plainTableConfig.indexSparseness()).
        isEqualTo(18);
  }

  @Test
  public void hugePageTlbSize() {
    PlainTableConfig plainTableConfig = new PlainTableConfig();
    plainTableConfig.setHugePageTlbSize(1);
    assertThat(plainTableConfig.hugePageTlbSize()).
        isEqualTo(1);
  }

  @Test
  public void encodingType() {
    PlainTableConfig plainTableConfig = new PlainTableConfig();
    plainTableConfig.setEncodingType(EncodingType.kPrefix);
    assertThat(plainTableConfig.encodingType()).isEqualTo(
        EncodingType.kPrefix);
  }

  @Test
  public void fullScanMode() {
    PlainTableConfig plainTableConfig = new PlainTableConfig();
    plainTableConfig.setFullScanMode(true);
    assertThat(plainTableConfig.fullScanMode()).isTrue();  }

  @Test
  public void storeIndexInFile() {
    PlainTableConfig plainTableConfig = new PlainTableConfig();
    plainTableConfig.setStoreIndexInFile(true);
    assertThat(plainTableConfig.storeIndexInFile()).
        isTrue();
  }

  @Test
  public void plainTableConfig() {
    Options opt = null;
    try {
      opt = new Options();
      PlainTableConfig plainTableConfig = new PlainTableConfig();
      opt.setTableFormatConfig(plainTableConfig);
      assertThat(opt.tableFactoryName()).isEqualTo("PlainTable");
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }
}
