// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import org.junit.ClassRule;
import org.junit.Test;

import static org.assertj.core.api.Assertions.assertThat;

public class ComparatorOptionsTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Test
  public void comparatorOptions() {
    final ComparatorOptions copt = new ComparatorOptions();

    assertThat(copt).isNotNull();

    { // UseAdaptiveMutex test
      copt.setUseAdaptiveMutex(true);
      assertThat(copt.useAdaptiveMutex()).isTrue();

      copt.setUseAdaptiveMutex(false);
      assertThat(copt.useAdaptiveMutex()).isFalse();
    }

    copt.dispose();
  }
}
