// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import java.io.IOException;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

import org.junit.ClassRule;
import org.junit.Test;

import static org.assertj.core.api.Assertions.assertThat;


public class WriteBatchHandlerTest {
  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Test
  public void writeBatchHandler() throws IOException, RocksDBException {
    WriteBatch batch = null;
    CapturingWriteBatchHandler handler = null;
    try {
      // setup test data
      final List<Tuple<Action, Tuple<byte[], byte[]>>> testEvents = new ArrayList<>();
      testEvents.add(new Tuple<>(Action.DELETE,
          new Tuple<byte[], byte[]>("k0".getBytes(), null)));
      testEvents.add(new Tuple<>(Action.PUT,
          new Tuple<>("k1".getBytes(), "v1".getBytes())));
      testEvents.add(new Tuple<>(Action.PUT,
          new Tuple<>("k2".getBytes(), "v2".getBytes())));
      testEvents.add(new Tuple<>(Action.PUT,
          new Tuple<>("k3".getBytes(), "v3".getBytes())));
      testEvents.add(new Tuple<>(Action.LOG,
          new Tuple<byte[], byte[]>(null, "log1".getBytes())));
      testEvents.add(new Tuple<>(Action.MERGE,
          new Tuple<>("k2".getBytes(), "v22".getBytes())));
      testEvents.add(new Tuple<>(Action.DELETE,
          new Tuple<byte[], byte[]>("k3".getBytes(), null)));

      // load test data to the write batch
      batch = new WriteBatch();
      for (final Tuple<Action, Tuple<byte[], byte[]>> testEvent : testEvents) {
        final Tuple<byte[], byte[]> data = testEvent.value;
        switch (testEvent.key) {

          case PUT:
            batch.put(data.key, data.value);
            break;

          case MERGE:
            batch.merge(data.key, data.value);
            break;

          case DELETE:
            batch.remove(data.key);
            break;

          case LOG:
            batch.putLogData(data.value);
            break;
        }
      }

      // attempt to read test data back from the WriteBatch by iterating with a handler
      handler = new CapturingWriteBatchHandler();
      batch.iterate(handler);

      // compare the results to the test data
      final List<Tuple<Action, Tuple<byte[], byte[]>>> actualEvents = handler.getEvents();
      assertThat(testEvents.size()).isSameAs(actualEvents.size());

      for (int i = 0; i < testEvents.size(); i++) {
        assertThat(equals(testEvents.get(i), actualEvents.get(i))).isTrue();
      }
    } finally {
      if (handler != null) {
        handler.dispose();
      }
      if (batch != null) {
        batch.dispose();
      }
    }
  }

  private static boolean equals(final Tuple<Action, Tuple<byte[], byte[]>> expected,
                                final Tuple<Action, Tuple<byte[], byte[]>> actual) {
    if (!expected.key.equals(actual.key)) {
      return false;
    }

    final Tuple<byte[], byte[]> expectedData = expected.value;
    final Tuple<byte[], byte[]> actualData = actual.value;

    return equals(expectedData.key, actualData.key)
        && equals(expectedData.value, actualData.value);
  }

  private static boolean equals(byte[] expected, byte[] actual) {
    if (expected != null) {
      return Arrays.equals(expected, actual);
    } else {
      return actual == null;
    }
  }

  private static class Tuple<K, V> {
    public final K key;
    public final V value;

    public Tuple(final K key, final V value) {
      this.key = key;
      this.value = value;
    }
  }

  /**
   * Enumeration of Write Batch
   * event actions
   */
  private enum Action {
    PUT,
    MERGE,
    DELETE,
    LOG
  }

  /**
   * A simple WriteBatch Handler which adds a record
   * of each event that it receives to a list
   */
  private static class CapturingWriteBatchHandler extends WriteBatch.Handler {

    private final List<Tuple<Action, Tuple<byte[], byte[]>>> events = new ArrayList<>();

    /**
     * Returns a copy of the current events list
     *
     * @return a list of the events which have happened upto now
     */
    public List<Tuple<Action, Tuple<byte[], byte[]>>> getEvents() {
      return new ArrayList<>(events);
    }

    @Override
    public void put(final byte[] key, final byte[] value) {
      events.add(new Tuple<>(Action.PUT, new Tuple<>(key, value)));
    }

    @Override
    public void merge(final byte[] key, final byte[] value) {
      events.add(new Tuple<>(Action.MERGE, new Tuple<>(key, value)));
    }

    @Override
    public void delete(final byte[] key) {
      events.add(new Tuple<>(Action.DELETE, new Tuple<byte[], byte[]>(key, null)));
    }

    @Override
    public void logData(final byte[] blob) {
      events.add(new Tuple<>(Action.LOG, new Tuple<byte[], byte[]>(null, blob)));
    }
  }
}
