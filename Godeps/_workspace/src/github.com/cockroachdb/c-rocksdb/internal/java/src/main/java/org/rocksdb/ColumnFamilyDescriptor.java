// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * <p>Describes a column family with a
 * name and respective Options.</p>
 */
public class ColumnFamilyDescriptor {

  /**
   * <p>Creates a new Column Family using a name and default
   * options,</p>
   *
   * @param columnFamilyName name of column family.
   * @since 3.10.0
   */
  public ColumnFamilyDescriptor(final byte[] columnFamilyName) {
    this(columnFamilyName, new ColumnFamilyOptions());
  }

  /**
   * <p>Creates a new Column Family using a name and custom
   * options.</p>
   *
   * @param columnFamilyName name of column family.
   * @param columnFamilyOptions options to be used with
   *     column family.
   * @since 3.10.0
   */
  public ColumnFamilyDescriptor(final byte[] columnFamilyName,
      final ColumnFamilyOptions columnFamilyOptions) {
    columnFamilyName_ = columnFamilyName;
    columnFamilyOptions_ = columnFamilyOptions;
  }

  /**
   * Retrieve name of column family.
   *
   * @return column family name.
   * @since 3.10.0
   */
  public byte[] columnFamilyName() {
    return columnFamilyName_;
  }

  /**
   * Retrieve assigned options instance.
   *
   * @return Options instance assigned to this instance.
   */
  public ColumnFamilyOptions columnFamilyOptions() {
    return columnFamilyOptions_;
  }

  private final byte[] columnFamilyName_;
  private final ColumnFamilyOptions columnFamilyOptions_;
}
