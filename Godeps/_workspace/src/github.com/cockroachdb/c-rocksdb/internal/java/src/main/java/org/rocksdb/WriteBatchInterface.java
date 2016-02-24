// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * <p>Defines the interface for a Write Batch which
 * holds a collection of updates to apply atomically to a DB.</p>
 */
public interface WriteBatchInterface {

    /**
     * Returns the number of updates in the batch.
     *
     * @return number of items in WriteBatch
     */
    int count();

    /**
     * <p>Store the mapping "key-&gt;value" in the database.</p>
     *
     * @param key the specified key to be inserted.
     * @param value the value associated with the specified key.
     */
    void put(byte[] key, byte[] value);

    /**
     * <p>Store the mapping "key-&gt;value" within given column
     * family.</p>
     *
     * @param columnFamilyHandle {@link org.rocksdb.ColumnFamilyHandle}
     *     instance
     * @param key the specified key to be inserted.
     * @param value the value associated with the specified key.
     */
    void put(ColumnFamilyHandle columnFamilyHandle,
                    byte[] key, byte[] value);

    /**
     * <p>Merge "value" with the existing value of "key" in the database.
     * "key-&gt;merge(existing, value)"</p>
     *
     * @param key the specified key to be merged.
     * @param value the value to be merged with the current value for
     * the specified key.
     */
    void merge(byte[] key, byte[] value);

    /**
     * <p>Merge "value" with the existing value of "key" in given column family.
     * "key-&gt;merge(existing, value)"</p>
     *
     * @param columnFamilyHandle {@link ColumnFamilyHandle} instance
     * @param key the specified key to be merged.
     * @param value the value to be merged with the current value for
     * the specified key.
     */
    void merge(ColumnFamilyHandle columnFamilyHandle,
                      byte[] key, byte[] value);

    /**
     * <p>If the database contains a mapping for "key", erase it.  Else do nothing.</p>
     *
     * @param key Key to delete within database
     */
    void remove(byte[] key);

    /**
     * <p>If column family contains a mapping for "key", erase it.  Else do nothing.</p>
     *
     * @param columnFamilyHandle {@link ColumnFamilyHandle} instance
     * @param key Key to delete within database
     */
    void remove(ColumnFamilyHandle columnFamilyHandle, byte[] key);

    /**
     * Append a blob of arbitrary size to the records in this batch. The blob will
     * be stored in the transaction log but not in any other file. In particular,
     * it will not be persisted to the SST files. When iterating over this
     * WriteBatch, WriteBatch::Handler::LogData will be called with the contents
     * of the blob as it is encountered. Blobs, puts, deletes, and merges will be
     * encountered in the same order in thich they were inserted. The blob will
     * NOT consume sequence number(s) and will NOT increase the count of the batch
     *
     * Example application: add timestamps to the transaction log for use in
     * replication.
     *
     * @param blob binary object to be inserted
     */
    void putLogData(byte[] blob);

    /**
     * Clear all updates buffered in this batch
     */
    void clear();
}
