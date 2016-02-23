//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
package org.rocksdb;

import org.junit.ClassRule;
import org.junit.Rule;
import org.junit.Test;
import org.junit.rules.TemporaryFolder;

import java.io.UnsupportedEncodingException;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * This class mimics the db/write_batch_test.cc
 * in the c++ rocksdb library.
 *
 * Not ported yet:
 *
 * Continue();
 * PutGatherSlices();
 */
public class WriteBatchTest {
  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Rule
  public TemporaryFolder dbFolder = new TemporaryFolder();

  @Test
  public void emptyWriteBatch() {
    WriteBatch batch = new WriteBatch();
    assertThat(batch.count()).isEqualTo(0);
  }

  @Test
  public void multipleBatchOperations()
      throws UnsupportedEncodingException {
    WriteBatch batch =  new WriteBatch();
    batch.put("foo".getBytes("US-ASCII"), "bar".getBytes("US-ASCII"));
    batch.remove("box".getBytes("US-ASCII"));
    batch.put("baz".getBytes("US-ASCII"), "boo".getBytes("US-ASCII"));
    WriteBatchTestInternalHelper.setSequence(batch, 100);
    assertThat(WriteBatchTestInternalHelper.sequence(batch)).
        isNotNull().
        isEqualTo(100);
    assertThat(batch.count()).isEqualTo(3);
    assertThat(new String(getContents(batch), "US-ASCII")).
        isEqualTo("Put(baz, boo)@102" +
                  "Delete(box)@101" +
                  "Put(foo, bar)@100");
  }

  @Test
  public void testAppendOperation()
      throws UnsupportedEncodingException {
    WriteBatch b1 = new WriteBatch();
    WriteBatch b2 = new WriteBatch();
    WriteBatchTestInternalHelper.setSequence(b1, 200);
    WriteBatchTestInternalHelper.setSequence(b2, 300);
    WriteBatchTestInternalHelper.append(b1, b2);
    assertThat(getContents(b1).length).isEqualTo(0);
    assertThat(b1.count()).isEqualTo(0);
    b2.put("a".getBytes("US-ASCII"), "va".getBytes("US-ASCII"));
    WriteBatchTestInternalHelper.append(b1, b2);
    assertThat("Put(a, va)@200".equals(new String(getContents(b1), "US-ASCII")));
    assertThat(b1.count()).isEqualTo(1);
    b2.clear();
    b2.put("b".getBytes("US-ASCII"), "vb".getBytes("US-ASCII"));
    WriteBatchTestInternalHelper.append(b1, b2);
    assertThat(("Put(a, va)@200" +
            "Put(b, vb)@201")
                .equals(new String(getContents(b1), "US-ASCII")));
    assertThat(b1.count()).isEqualTo(2);
    b2.remove("foo".getBytes("US-ASCII"));
    WriteBatchTestInternalHelper.append(b1, b2);
    assertThat(("Put(a, va)@200" +
        "Put(b, vb)@202" +
        "Put(b, vb)@201" +
        "Delete(foo)@203")
        .equals(new String(getContents(b1), "US-ASCII")));
    assertThat(b1.count()).isEqualTo(4);
  }

  @Test
  public void blobOperation()
      throws UnsupportedEncodingException {
    WriteBatch batch = new WriteBatch();
    batch.put("k1".getBytes("US-ASCII"), "v1".getBytes("US-ASCII"));
    batch.put("k2".getBytes("US-ASCII"), "v2".getBytes("US-ASCII"));
    batch.put("k3".getBytes("US-ASCII"), "v3".getBytes("US-ASCII"));
    batch.putLogData("blob1".getBytes("US-ASCII"));
    batch.remove("k2".getBytes("US-ASCII"));
    batch.putLogData("blob2".getBytes("US-ASCII"));
    batch.merge("foo".getBytes("US-ASCII"), "bar".getBytes("US-ASCII"));
    assertThat(batch.count()).isEqualTo(5);
    assertThat(("Merge(foo, bar)@4" +
            "Put(k1, v1)@0" +
            "Delete(k2)@3" +
            "Put(k2, v2)@1" +
            "Put(k3, v3)@2")
               .equals(new String(getContents(batch), "US-ASCII")));
  }

  static native byte[] getContents(WriteBatch batch);
}

/**
 * Package-private class which provides java api to access
 * c++ WriteBatchInternal.
 */
class WriteBatchTestInternalHelper {
  static native void setSequence(WriteBatch batch, long sn);
  static native long sequence(WriteBatch batch);
  static native void append(WriteBatch b1, WriteBatch b2);
}
