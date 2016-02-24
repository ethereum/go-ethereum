// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
package org.rocksdb;

/**
 * Instances of this class describe a Backup made by
 * {@link org.rocksdb.BackupableDB}.
 */
public class BackupInfo {

  /**
   * Package private constructor used to create instances
   * of BackupInfo by {@link org.rocksdb.BackupableDB} and
   * {@link org.rocksdb.RestoreBackupableDB}.
   *
   * @param backupId id of backup
   * @param timestamp timestamp of backup
   * @param size size of backup
   * @param numberFiles number of files related to this backup.
   */
  BackupInfo(final int backupId, final long timestamp, final long size,
      final int numberFiles) {
    backupId_ = backupId;
    timestamp_ = timestamp;
    size_ = size;
    numberFiles_ = numberFiles;
  }

  /**
   *
   * @return the backup id.
   */
  public int backupId() {
    return backupId_;
  }

  /**
   *
   * @return the timestamp of the backup.
   */
  public long timestamp() {
    return timestamp_;
  }

  /**
   *
   * @return the size of the backup
   */
  public long size() {
    return size_;
  }

  /**
   *
   * @return the number of files of this backup.
   */
  public int numberFiles() {
    return numberFiles_;
  }

  private int backupId_;
  private long timestamp_;
  private long size_;
  private int numberFiles_;
}
