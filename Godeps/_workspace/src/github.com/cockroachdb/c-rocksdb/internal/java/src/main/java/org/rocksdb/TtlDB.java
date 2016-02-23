// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import java.util.List;

/**
 * Database with TTL support.
 *
 * <p><strong>Use case</strong></p>
 * <p>This API should be used to open the db when key-values inserted are
 * meant to be removed from the db in a non-strict 'ttl' amount of time
 * Therefore, this guarantees that key-values inserted will remain in the
 * db for &gt;= ttl amount of time and the db will make efforts to remove the
 * key-values as soon as possible after ttl seconds of their insertion.
 * </p>
 *
 * <p><strong>Behaviour</strong></p>
 * <p>TTL is accepted in seconds
 * (int32_t)Timestamp(creation) is suffixed to values in Put internally
 * Expired TTL values deleted in compaction only:(Timestamp+ttl&lt;time_now)
 * Get/Iterator may return expired entries(compaction not run on them yet)
 * Different TTL may be used during different Opens
 * </p>
 *
 * <p><strong>Example</strong></p>
 * <ul>
 * <li>Open1 at t=0 with ttl=4 and insert k1,k2, close at t=2</li>
 * <li>Open2 at t=3 with ttl=5. Now k1,k2 should be deleted at t&gt;=5</li>
 * </ul>
 *
 * <p>
 * read_only=true opens in the usual read-only mode. Compactions will not be
 *  triggered(neither manual nor automatic), so no expired entries removed
 * </p>
 *
 * <p><strong>Constraints</strong></p>
 * <p>Not specifying/passing or non-positive TTL behaves
 * like TTL = infinity</p>
 *
 * <p><strong>!!!WARNING!!!</strong></p>
 * <p>Calling DB::Open directly to re-open a db created by this API will get
 * corrupt values(timestamp suffixed) and no ttl effect will be there
 * during the second Open, so use this API consistently to open the db
 * Be careful when passing ttl with a small positive value because the
 * whole database may be deleted in a small amount of time.</p>
 */
public class TtlDB extends RocksDB {

  /**
   * <p>Opens a TtlDB.</p>
   *
   * <p>Database is opened in read-write mode without default TTL.</p>
   *
   * @param options {@link org.rocksdb.Options} instance.
   * @param db_path path to database.
   *
   * @return TtlDB instance.
   *
   * @throws RocksDBException thrown if an error occurs within the native
   *     part of the library.
   */
  public static TtlDB open(final Options options, final String db_path)
      throws RocksDBException {
    return open(options, db_path, 0, false);
  }

  /**
   * <p>Opens a TtlDB.</p>
   *
   * @param options {@link org.rocksdb.Options} instance.
   * @param db_path path to database.
   * @param ttl time to live for new entries.
   * @param readOnly boolean value indicating if database if db is
   *     opened read-only.
   *
   * @return TtlDB instance.
   *
   * @throws RocksDBException thrown if an error occurs within the native
   *     part of the library.
   */
  public static TtlDB open(final Options options, final String db_path,
      final int ttl, final boolean readOnly) throws RocksDBException {
    TtlDB ttldb = new TtlDB();
    ttldb.open(options.nativeHandle_, db_path, ttl, readOnly);
    return ttldb;
  }

  /**
   * <p>Opens a TtlDB.</p>
   *
   * @param options {@link org.rocksdb.Options} instance.
   * @param db_path path to database.
   * @param columnFamilyDescriptors list of column family descriptors
   * @param columnFamilyHandles will be filled with ColumnFamilyHandle instances
   *     on open.
   * @param ttlValues time to live values per column family handle
   * @param readOnly boolean value indicating if database if db is
   *     opened read-only.
   *
   * @return TtlDB instance.
   *
   * @throws RocksDBException thrown if an error occurs within the native
   *     part of the library.
   * @throws java.lang.IllegalArgumentException when there is not a ttl value
   *     per given column family handle.
   */
  public static TtlDB open(final DBOptions options, final String db_path,
      final List<ColumnFamilyDescriptor> columnFamilyDescriptors,
      final List<ColumnFamilyHandle> columnFamilyHandles,
      final List<Integer> ttlValues, final boolean readOnly)
      throws RocksDBException {
    if (columnFamilyDescriptors.size() != ttlValues.size()) {
      throw new IllegalArgumentException("There must be a ttl value per column" +
          "family handle.");
    }
    TtlDB ttlDB = new TtlDB();
    List<Long> cfReferences = ttlDB.openCF(options.nativeHandle_, db_path,
        columnFamilyDescriptors, columnFamilyDescriptors.size(),
        ttlValues, readOnly);
    for (int i=0; i<columnFamilyDescriptors.size(); i++) {
      columnFamilyHandles.add(new ColumnFamilyHandle(ttlDB, cfReferences.get(i)));
    }
    return ttlDB;
  }

  /**
   * <p>Creates a new ttl based column family with a name defined
   * in given ColumnFamilyDescriptor and allocates a
   * ColumnFamilyHandle within an internal structure.</p>
   *
   * <p>The ColumnFamilyHandle is automatically disposed with DB
   * disposal.</p>
   *
   * @param columnFamilyDescriptor column family to be created.
   * @param ttl TTL to set for this column family.
   *
   * @return {@link org.rocksdb.ColumnFamilyHandle} instance.
   *
   * @throws RocksDBException thrown if error happens in underlying
   *    native library.
   */
  public ColumnFamilyHandle createColumnFamilyWithTtl(
      final ColumnFamilyDescriptor columnFamilyDescriptor,
      final int ttl) throws RocksDBException {
    assert(isInitialized());
    return new ColumnFamilyHandle(this,
        createColumnFamilyWithTtl(nativeHandle_,
            columnFamilyDescriptor, ttl));
  }

  /**
   * <p>Close the TtlDB instance and release resource.</p>
   *
   * <p>Internally, TtlDB owns the {@code rocksdb::DB} pointer
   * to its associated {@link org.rocksdb.RocksDB}. The release
   * of that RocksDB pointer is handled in the destructor of the
   * c++ {@code rocksdb::TtlDB} and should be transparent to
   * Java developers.</p>
   */
  @Override public synchronized void close() {
    if (isInitialized()) {
      super.close();
    }
  }

  /**
   * <p>A protected constructor that will be used in the static
   * factory method
   * {@link #open(Options, String, int, boolean)}
   * and
   * {@link #open(DBOptions, String, java.util.List, java.util.List,
   * java.util.List, boolean)}.
   * </p>
   */
  protected TtlDB() {
    super();
  }

  @Override protected void finalize() throws Throwable {
    close();
    super.finalize();
  }

  private native void open(long optionsHandle, String db_path, int ttl,
      boolean readOnly) throws RocksDBException;
  private native List<Long> openCF(long optionsHandle, String db_path,
      List<ColumnFamilyDescriptor> columnFamilyDescriptors,
      int columnFamilyDescriptorsLength, List<Integer> ttlValues,
      boolean readOnly) throws RocksDBException;
  private native long createColumnFamilyWithTtl(long handle,
      ColumnFamilyDescriptor columnFamilyDescriptor, int ttl)
      throws RocksDBException;
}
