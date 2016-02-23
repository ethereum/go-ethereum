// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import java.io.File;
import java.nio.file.Path;

/**
 * <p>BackupableDBOptions to control the behavior of a backupable database.
 * It will be used during the creation of a {@link org.rocksdb.BackupableDB}.
 * </p>
 * <p>Note that dispose() must be called before an Options instance
 * become out-of-scope to release the allocated memory in c++.</p>
 *
 * @see org.rocksdb.BackupableDB
 */
public class BackupableDBOptions extends RocksObject {

  /**
   * <p>BackupableDBOptions constructor.</p>
   *
   * @param path Where to keep the backup files. Has to be different than db name.
   *     Best to set this to {@code db name_ + "/backups"}
   * @throws java.lang.IllegalArgumentException if illegal path is used.
   */
  public BackupableDBOptions(final String path) {
    super();
    File backupPath = path == null ? null : new File(path);
    if (backupPath == null || !backupPath.isDirectory() || !backupPath.canWrite()) {
      throw new IllegalArgumentException("Illegal path provided.");
    }
    newBackupableDBOptions(path);
  }

  /**
   * <p>Returns the path to the BackupableDB directory.</p>
   *
   * @return the path to the BackupableDB directory.
   */
  public String backupDir() {
    assert(isInitialized());
    return backupDir(nativeHandle_);
  }

  /**
   * <p>Share table files between backups.</p>
   *
   * @param shareTableFiles If {@code share_table_files == true}, backup will assume
   *     that table files with same name have the same contents. This enables incremental
   *     backups and avoids unnecessary data copies. If {@code share_table_files == false},
   *     each backup will be on its own and will not share any data with other backups.
   *
   * <p>Default: true</p>
   *
   * @return instance of current BackupableDBOptions.
   */
  public BackupableDBOptions setShareTableFiles(final boolean shareTableFiles) {
    assert(isInitialized());
    setShareTableFiles(nativeHandle_, shareTableFiles);
    return this;
  }

  /**
   * <p>Share table files between backups.</p>
   *
   * @return boolean value indicating if SST files will be shared between
   *     backups.
   */
  public boolean shareTableFiles() {
    assert(isInitialized());
    return shareTableFiles(nativeHandle_);
  }

  /**
   * <p>Set synchronous backups.</p>
   *
   * @param sync If {@code sync == true}, we can guarantee you'll get consistent backup
   *     even on a machine crash/reboot. Backup process is slower with sync enabled.
   *     If {@code sync == false}, we don't guarantee anything on machine reboot.
   *     However,chances are some of the backups are consistent.
   *
   * <p>Default: true</p>
   *
   * @return instance of current BackupableDBOptions.
   */
  public BackupableDBOptions setSync(final boolean sync) {
    assert(isInitialized());
    setSync(nativeHandle_, sync);
    return this;
  }

  /**
   * <p>Are synchronous backups activated.</p>
   *
   * @return boolean value if synchronous backups are configured.
   */
  public boolean sync() {
    assert(isInitialized());
    return sync(nativeHandle_);
  }

  /**
   * <p>Set if old data will be destroyed.</p>
   *
   * @param destroyOldData If true, it will delete whatever backups there are already.
   *
   * <p>Default: false</p>
   *
   * @return instance of current BackupableDBOptions.
   */
  public BackupableDBOptions setDestroyOldData(final boolean destroyOldData) {
    assert(isInitialized());
    setDestroyOldData(nativeHandle_, destroyOldData);
    return this;
  }

  /**
   * <p>Returns if old data will be destroyed will performing new backups.</p>
   *
   * @return boolean value indicating if old data will be destroyed.
   */
  public boolean destroyOldData() {
    assert(isInitialized());
    return destroyOldData(nativeHandle_);
  }

  /**
   * <p>Set if log files shall be persisted.</p>
   *
   * @param backupLogFiles If false, we won't backup log files. This option can be
   *     useful for backing up in-memory databases where log file are persisted,but table
   *     files are in memory.
   *
   * <p>Default: true</p>
   *
   * @return instance of current BackupableDBOptions.
   */
  public BackupableDBOptions setBackupLogFiles(final boolean backupLogFiles) {
    assert(isInitialized());
    setBackupLogFiles(nativeHandle_, backupLogFiles);
    return this;
  }

  /**
   * <p>Return information if log files shall be persisted.</p>
   *
   * @return boolean value indicating if log files will be persisted.
   */
  public boolean backupLogFiles() {
    assert(isInitialized());
    return backupLogFiles(nativeHandle_);
  }

  /**
   * <p>Set backup rate limit.</p>
   *
   * @param backupRateLimit Max bytes that can be transferred in a second during backup.
   *     If 0 or negative, then go as fast as you can.
   *
   * <p>Default: 0</p>
   *
   * @return instance of current BackupableDBOptions.
   */
  public BackupableDBOptions setBackupRateLimit(long backupRateLimit) {
    assert(isInitialized());
    backupRateLimit = (backupRateLimit <= 0) ? 0 : backupRateLimit;
    setBackupRateLimit(nativeHandle_, backupRateLimit);
    return this;
  }

  /**
   * <p>Return backup rate limit which described the max bytes that can be transferred in a
   * second during backup.</p>
   *
   * @return numerical value describing the backup transfer limit in bytes per second.
   */
  public long backupRateLimit() {
    assert(isInitialized());
    return backupRateLimit(nativeHandle_);
  }

  /**
   * <p>Set restore rate limit.</p>
   *
   * @param restoreRateLimit Max bytes that can be transferred in a second during restore.
   *     If 0 or negative, then go as fast as you can.
   *
   * <p>Default: 0</p>
   *
   * @return instance of current BackupableDBOptions.
   */
  public BackupableDBOptions setRestoreRateLimit(long restoreRateLimit) {
    assert(isInitialized());
    restoreRateLimit = (restoreRateLimit <= 0) ? 0 : restoreRateLimit;
    setRestoreRateLimit(nativeHandle_, restoreRateLimit);
    return this;
  }

  /**
   * <p>Return restore rate limit which described the max bytes that can be transferred in a
   * second during restore.</p>
   *
   * @return numerical value describing the restore transfer limit in bytes per second.
   */
  public long restoreRateLimit() {
    assert(isInitialized());
    return restoreRateLimit(nativeHandle_);
  }

  /**
   * <p>Only used if share_table_files is set to true. If true, will consider that
   * backups can come from different databases, hence a sst is not uniquely
   * identified by its name, but by the triple (file name, crc32, file length)</p>
   *
   * @param shareFilesWithChecksum boolean value indicating if SST files are stored
   *     using the triple (file name, crc32, file length) and not its name.
   *
   * <p>Note: this is an experimental option, and you'll need to set it manually
   * turn it on only if you know what you're doing*</p>
   *
   * <p>Default: false</p>
   *
   * @return instance of current BackupableDBOptions.
   */
  public BackupableDBOptions setShareFilesWithChecksum(
      final boolean shareFilesWithChecksum) {
    assert(isInitialized());
    setShareFilesWithChecksum(nativeHandle_, shareFilesWithChecksum);
    return this;
  }

  /**
   * <p>Return of share files with checksum is active.</p>
   *
   * @return boolean value indicating if share files with checksum
   *     is active.
   */
  public boolean shareFilesWithChecksum() {
    assert(isInitialized());
    return shareFilesWithChecksum(nativeHandle_);
  }

  /**
   * Release the memory allocated for the current instance
   * in the c++ side.
   */
  @Override protected void disposeInternal() {
    disposeInternal(nativeHandle_);
  }

  private native void newBackupableDBOptions(String path);
  private native String backupDir(long handle);
  private native void setShareTableFiles(long handle, boolean flag);
  private native boolean shareTableFiles(long handle);
  private native void setSync(long handle, boolean flag);
  private native boolean sync(long handle);
  private native void setDestroyOldData(long handle, boolean flag);
  private native boolean destroyOldData(long handle);
  private native void setBackupLogFiles(long handle, boolean flag);
  private native boolean backupLogFiles(long handle);
  private native void setBackupRateLimit(long handle, long rateLimit);
  private native long backupRateLimit(long handle);
  private native void setRestoreRateLimit(long handle, long rateLimit);
  private native long restoreRateLimit(long handle);
  private native void setShareFilesWithChecksum(long handle, boolean flag);
  private native boolean shareFilesWithChecksum(long handle);
  private native void disposeInternal(long handle);
}
