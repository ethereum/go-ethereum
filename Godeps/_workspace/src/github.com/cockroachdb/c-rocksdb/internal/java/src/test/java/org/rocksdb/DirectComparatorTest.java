// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import org.junit.ClassRule;
import org.junit.Rule;
import org.junit.Test;
import org.junit.rules.TemporaryFolder;

import java.io.IOException;
import java.nio.file.FileSystems;

public class DirectComparatorTest {
  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Rule
  public TemporaryFolder dbFolder = new TemporaryFolder();

  @Test
  public void directComparator() throws IOException, RocksDBException {

    final AbstractComparatorTest comparatorTest = new AbstractComparatorTest() {
      @Override
      public AbstractComparator getAscendingIntKeyComparator() {
        return new DirectComparator(new ComparatorOptions()) {

          @Override
          public String name() {
            return "test.AscendingIntKeyDirectComparator";
          }

          @Override
          public int compare(final DirectSlice a, final DirectSlice b) {
            final byte ax[] = new byte[4], bx[] = new byte[4];
            a.data().get(ax);
            b.data().get(bx);
            return compareIntKeys(ax, bx);
          }
        };
      }
    };

    // test the round-tripability of keys written and read with the DirectComparator
    comparatorTest.testRoundtrip(FileSystems.getDefault().getPath(
        dbFolder.getRoot().getAbsolutePath()));
  }
}
