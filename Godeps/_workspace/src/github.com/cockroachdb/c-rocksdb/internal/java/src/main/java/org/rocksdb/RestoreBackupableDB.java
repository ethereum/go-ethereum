// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import java.util.List;

/**
 * <p>This class is used to access information about backups and
 * restore from them.</p>
 *
 * <p>Note: {@code dispose()} must be called before this instance
 * become out-of-scope to release the allocated
 * memory in c++.</p>
 *
 */
public class RestoreBackupableDB extends RocksObject {
  /**
   * <p>Construct new estoreBackupableDB instance.</p>
   *
   * @param options {@link org.rocksdb.BackupableDBOptions} instance
   */
  public RestoreBackupableDB(final BackupableDBOptions options) {
    super();
    nativeHandle_ = newRestoreBackupableDB(options.nativeHandle_);
  }

  /**
   * <p>Restore from backup with backup_id.</p>
   *
   * <p><strong>Important</strong>: If options_.share_table_files == true
   * and you restore DB from some backup that is not the latest, and you
   * start creating new backups from the new DB, they will probably
   * fail.</p>
   *
   * <p><strong>Example</strong>: Let's say you have backups 1, 2, 3, 4, 5
   * and you restore 3. If you add new data to the DB and try creating a new
   * backup now, the database will diverge from backups 4 and 5 and the new
   * backup will fail. If you want to create new backup, you will first have
   * to delete backups 4 and 5.</p>
   *
   * @param backupId id pointing to backup
   * @param dbDir database directory to restore to
   * @param walDir directory where wal files are located
   * @param restoreOptions {@link org.rocksdb.RestoreOptions} instance.
   *
   * @throws RocksDBException thrown if error happens in underlying
   *    native library.
   */
  public void restoreDBFromBackup(final long backupId, final String dbDir,
      final String walDir, final RestoreOptions restoreOptions)
      throws RocksDBException {
    assert(isInitialized());
    restoreDBFromBackup0(nativeHandle_, backupId, dbDir, walDir,
        restoreOptions.nativeHandle_);
  }

  /**
   * <p>Restore from the latest backup.</p>
   *
   * @param dbDir database directory to restore to
   * @param walDir directory where wal files are located
   * @param restoreOptions {@link org.rocksdb.RestoreOptions} instance
   *
   * @throws RocksDBException thrown if error happens in underlying
   *    native library.
   */
  public void restoreDBFromLatestBackup(final String dbDir,
      final String walDir, final RestoreOptions restoreOptions)
      throws RocksDBException {
    assert(isInitialized());
    restoreDBFromLatestBackup0(nativeHandle_, dbDir, walDir,
        restoreOptions.nativeHandle_);
  }

  /**
   * <p>Deletes old backups, keeping latest numBackupsToKeep alive.</p>
   *
   * @param numBackupsToKeep of latest backups to keep
   *
   * @throws RocksDBException thrown if error happens in underlying
   *    native library.
   */
  public void purgeOldBackups(final int numBackupsToKeep)
      throws RocksDBException {
    assert(isInitialized());
    purgeOldBackups0(nativeHandle_, numBackupsToKeep);
  }

  /**
   * <p>Deletes a specific backup.</p>
   *
   * @param backupId of backup to delete.
   *
   * @throws RocksDBException thrown if error happens in underlying
   *    native library.
   */
  public void deleteBackup(final int backupId)
      throws RocksDBException {
    assert(isInitialized());
    deleteBackup0(nativeHandle_, backupId);
  }

  /**
   * <p>Returns a list of {@link BackupInfo} instances, which describe
   * already made backups.</p>
   *
   * @return List of {@link BackupInfo} instances.
   */
  public List<BackupInfo> getBackupInfos() {
    assert(isInitialized());
    return getBackupInfo(nativeHandle_);
  }

  /**
   * <p>Returns a list of corrupted backup ids. If there
   * is no corrupted backup the method will return an
   * empty list.</p>
   *
   * @return array of backup ids as int ids.
   */
  public int[] getCorruptedBackups() {
    assert(isInitialized());
    return getCorruptedBackups(nativeHandle_);
  }

  /**
   * <p>Will delete all the files we don't need anymore. It will
   * do the full scan of the files/ directory and delete all the
   * files that are not referenced.</p>
   *
   * @throws RocksDBException thrown if error happens in underlying
   *    native library.
   */
  public void garbageCollect() throws RocksDBException {
    assert(isInitialized());
    garbageCollect(nativeHandle_);
  }

  /**
   * <p>Release the memory allocated for the current instance
   * in the c++ side.</p>
   */
  @Override public synchronized void disposeInternal() {
    dispose(nativeHandle_);
  }

  private native long newRestoreBackupableDB(long options);
  private native void restoreDBFromBackup0(long nativeHandle, long backupId,
      String dbDir, String walDir, long restoreOptions)
      throws RocksDBException;
  private native void restoreDBFromLatestBackup0(long nativeHandle,
      String dbDir, String walDir, long restoreOptions)
      throws RocksDBException;
  private native void purgeOldBackups0(long nativeHandle, int numBackupsToKeep)
      throws RocksDBException;
  private native void deleteBackup0(long nativeHandle, int backupId)
      throws RocksDBException;
  private native List<BackupInfo> getBackupInfo(long handle);
  private native int[] getCorruptedBackups(long handle);
  private native void garbageCollect(long handle)
      throws RocksDBException;
  private native void dispose(long nativeHandle);
}
