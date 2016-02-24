// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
package org.rocksdb;

import org.junit.ClassRule;
import org.junit.Test;

import static org.assertj.core.api.Assertions.assertThat;

public class SliceTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Test
  public void slice() {
    Slice slice = null;
    Slice otherSlice = null;
    Slice thirdSlice = null;
    try {
      slice = new Slice("testSlice");
      assertThat(slice.empty()).isFalse();
      assertThat(slice.size()).isEqualTo(9);
      assertThat(slice.data()).isEqualTo("testSlice".getBytes());

      otherSlice = new Slice("otherSlice".getBytes());
      assertThat(otherSlice.data()).isEqualTo("otherSlice".getBytes());

      thirdSlice = new Slice("otherSlice".getBytes(), 5);
      assertThat(thirdSlice.data()).isEqualTo("Slice".getBytes());
    } finally {
      if (slice != null) {
        slice.dispose();
      }
      if (otherSlice != null) {
        otherSlice.dispose();
      }
      if (thirdSlice != null) {
        thirdSlice.dispose();
      }
    }
  }

  @Test
  public void sliceEquals() {
    Slice slice = null;
    Slice slice2 = null;
    try {
      slice = new Slice("abc");
      slice2 = new Slice("abc");
      assertThat(slice.equals(slice2)).isTrue();
      assertThat(slice.hashCode() == slice2.hashCode()).isTrue();
    } finally {
      if (slice != null) {
        slice.dispose();
      }
      if (slice2 != null) {
        slice2.dispose();
      }
    }
  }


  @Test
  public void sliceStartWith() {
    Slice slice = null;
    Slice match = null;
    Slice noMatch = null;
    try {
      slice = new Slice("matchpoint");
      match = new Slice("mat");
      noMatch = new Slice("nomatch");

      //assertThat(slice.startsWith(match)).isTrue();
      assertThat(slice.startsWith(noMatch)).isFalse();
    } finally {
      if (slice != null) {
        slice.dispose();
      }
      if (match != null) {
        match.dispose();
      }
      if (noMatch != null) {
        noMatch.dispose();
      }
    }
  }

  @Test
  public void sliceToString() {
    Slice slice = null;
    try {
      slice = new Slice("stringTest");
      assertThat(slice.toString()).isEqualTo("stringTest");
      assertThat(slice.toString(true)).isNotEqualTo("");
    } finally {
      if (slice != null) {
        slice.dispose();
      }
    }
  }
}
