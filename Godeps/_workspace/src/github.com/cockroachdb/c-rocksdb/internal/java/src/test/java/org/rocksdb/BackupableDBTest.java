// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import org.junit.ClassRule;
import org.junit.Rule;
import org.junit.Test;
import org.junit.rules.TemporaryFolder;

import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;

public class BackupableDBTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Rule
  public TemporaryFolder dbFolder = new TemporaryFolder();

  @Rule
  public TemporaryFolder backupFolder = new TemporaryFolder();

  @Test
  public void backupDb() throws RocksDBException {
    Options opt = null;
    BackupableDBOptions bopt = null;
    BackupableDB bdb = null;
    try {
      opt = new Options().setCreateIfMissing(true);
      bopt = new BackupableDBOptions(
          backupFolder.getRoot().getAbsolutePath());
      assertThat(bopt.backupDir()).isEqualTo(
          backupFolder.getRoot().getAbsolutePath());
      // Open empty database.
      bdb = BackupableDB.open(opt, bopt,
          dbFolder.getRoot().getAbsolutePath());
      // Fill database with some test values
      prepareDatabase(bdb);
      // Create two backups
      bdb.createNewBackup(false);
      bdb.createNewBackup(true);
      verifyNumberOfValidBackups(bdb, 2);
    } finally {
      if (bdb != null) {
        bdb.close();
      }
      if (bopt != null) {
        bopt.dispose();
      }
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void deleteBackup() throws RocksDBException {
    Options opt = null;
    BackupableDBOptions bopt = null;
    BackupableDB bdb = null;
    try {
      opt = new Options().setCreateIfMissing(true);
      bopt = new BackupableDBOptions(
          backupFolder.getRoot().getAbsolutePath());
      assertThat(bopt.backupDir()).isEqualTo(
          backupFolder.getRoot().getAbsolutePath());
      // Open empty database.
      bdb = BackupableDB.open(opt, bopt,
          dbFolder.getRoot().getAbsolutePath());
      // Fill database with some test values
      prepareDatabase(bdb);
      // Create two backups
      bdb.createNewBackup(false);
      bdb.createNewBackup(true);
      List<BackupInfo> backupInfo =
          verifyNumberOfValidBackups(bdb, 2);
      // Delete the first backup
      bdb.deleteBackup(backupInfo.get(0).backupId());
      List<BackupInfo> newBackupInfo =
          verifyNumberOfValidBackups(bdb, 1);
      // The second backup must remain.
      assertThat(newBackupInfo.get(0).backupId()).
          isEqualTo(backupInfo.get(1).backupId());
    } finally {
      if (bdb != null) {
        bdb.close();
      }
      if (bopt != null) {
        bopt.dispose();
      }
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void deleteBackupWithRestoreBackupableDB()
      throws RocksDBException {
    Options opt = null;
    BackupableDBOptions bopt = null;
    BackupableDB bdb = null;
    RestoreBackupableDB rdb = null;
    try {
      opt = new Options().setCreateIfMissing(true);
      bopt = new BackupableDBOptions(
          backupFolder.getRoot().getAbsolutePath());
      assertThat(bopt.backupDir()).isEqualTo(
          backupFolder.getRoot().getAbsolutePath());
      // Open empty database.
      bdb = BackupableDB.open(opt, bopt,
          dbFolder.getRoot().getAbsolutePath());
      // Fill database with some test values
      prepareDatabase(bdb);
      // Create two backups
      bdb.createNewBackup(false);
      bdb.createNewBackup(true);
      List<BackupInfo> backupInfo =
          verifyNumberOfValidBackups(bdb, 2);
      // init RestoreBackupableDB
      rdb = new RestoreBackupableDB(bopt);
      // Delete the first backup
      rdb.deleteBackup(backupInfo.get(0).backupId());
      // Fetch backup info using RestoreBackupableDB
      List<BackupInfo> newBackupInfo = verifyNumberOfValidBackups(rdb, 1);
      // The second backup must remain.
      assertThat(newBackupInfo.get(0).backupId()).
          isEqualTo(backupInfo.get(1).backupId());
    } finally {
      if (bdb != null) {
        bdb.close();
      }
      if (rdb != null) {
        rdb.dispose();
      }
      if (bopt != null) {
        bopt.dispose();
      }
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void purgeOldBackups() throws RocksDBException {
    Options opt = null;
    BackupableDBOptions bopt = null;
    BackupableDB bdb = null;
    try {
      opt = new Options().setCreateIfMissing(true);
      bopt = new BackupableDBOptions(
          backupFolder.getRoot().getAbsolutePath());
      assertThat(bopt.backupDir()).isEqualTo(
          backupFolder.getRoot().getAbsolutePath());
      // Open empty database.
      bdb = BackupableDB.open(opt, bopt,
          dbFolder.getRoot().getAbsolutePath());
      // Fill database with some test values
      prepareDatabase(bdb);
      // Create two backups
      bdb.createNewBackup(false);
      bdb.createNewBackup(true);
      bdb.createNewBackup(true);
      bdb.createNewBackup(true);
      List<BackupInfo> backupInfo =
          verifyNumberOfValidBackups(bdb, 4);
      // Delete everything except the latest backup
      bdb.purgeOldBackups(1);
      List<BackupInfo> newBackupInfo =
          verifyNumberOfValidBackups(bdb, 1);
      // The latest backup must remain.
      assertThat(newBackupInfo.get(0).backupId()).
          isEqualTo(backupInfo.get(3).backupId());
    } finally {
      if (bdb != null) {
        bdb.close();
      }
      if (bopt != null) {
        bopt.dispose();
      }
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void purgeOldBackupsWithRestoreBackupableDb()
      throws RocksDBException {
    Options opt = null;
    BackupableDBOptions bopt = null;
    BackupableDB bdb = null;
    RestoreBackupableDB rdb = null;
    try {
      opt = new Options().setCreateIfMissing(true);
      bopt = new BackupableDBOptions(
          backupFolder.getRoot().getAbsolutePath());
      assertThat(bopt.backupDir()).isEqualTo(
          backupFolder.getRoot().getAbsolutePath());
      // Open empty database.
      bdb = BackupableDB.open(opt, bopt,
          dbFolder.getRoot().getAbsolutePath());
      // Fill database with some test values
      prepareDatabase(bdb);
      // Create two backups
      bdb.createNewBackup(false);
      bdb.createNewBackup(true);
      bdb.createNewBackup(true);
      bdb.createNewBackup(true);
      List<BackupInfo> infos = verifyNumberOfValidBackups(bdb, 4);
      assertThat(infos.get(1).size()).
          isEqualTo(infos.get(2).size());
      assertThat(infos.get(1).numberFiles()).
          isEqualTo(infos.get(2).numberFiles());
      long maxTimeBeforePurge = Long.MIN_VALUE;
      for (BackupInfo backupInfo : infos) {
        if (maxTimeBeforePurge < backupInfo.timestamp()) {
          maxTimeBeforePurge = backupInfo.timestamp();
        }
      }
      // init RestoreBackupableDB
      rdb = new RestoreBackupableDB(bopt);
      // the same number of backups must
      // exist using RestoreBackupableDB.
      verifyNumberOfValidBackups(rdb, 4);
      rdb.purgeOldBackups(1);
      infos = verifyNumberOfValidBackups(rdb, 1);
      assertThat(infos.get(0).timestamp()).
          isEqualTo(maxTimeBeforePurge);
    } finally {
      if (bdb != null) {
        bdb.close();
      }
      if (rdb != null) {
        rdb.dispose();
      }
      if (bopt != null) {
        bopt.dispose();
      }
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void restoreLatestBackup()
      throws RocksDBException {
    Options opt = null;
    BackupableDBOptions bopt = null;
    BackupableDB bdb = null;
    RestoreBackupableDB rdb = null;
    try {
      opt = new Options().setCreateIfMissing(true);
      bopt = new BackupableDBOptions(
          backupFolder.getRoot().getAbsolutePath());
      assertThat(bopt.backupDir()).isEqualTo(
          backupFolder.getRoot().getAbsolutePath());
      // Open empty database.
      bdb = BackupableDB.open(opt, bopt,
          dbFolder.getRoot().getAbsolutePath());
      // Fill database with some test values
      prepareDatabase(bdb);
      bdb.createNewBackup(true);
      verifyNumberOfValidBackups(bdb, 1);
      bdb.put("key1".getBytes(), "valueV2".getBytes());
      bdb.put("key2".getBytes(), "valueV2".getBytes());
      bdb.createNewBackup(true);
      verifyNumberOfValidBackups(bdb, 2);
      bdb.put("key1".getBytes(), "valueV3".getBytes());
      bdb.put("key2".getBytes(), "valueV3".getBytes());
      assertThat(new String(bdb.get("key1".getBytes()))).endsWith("V3");
      assertThat(new String(bdb.get("key2".getBytes()))).endsWith("V3");
      bdb.close();

      // init RestoreBackupableDB
      rdb = new RestoreBackupableDB(bopt);
      verifyNumberOfValidBackups(rdb, 2);
      // restore db from latest backup
      rdb.restoreDBFromLatestBackup(dbFolder.getRoot().getAbsolutePath(),
          dbFolder.getRoot().getAbsolutePath(),
          new RestoreOptions(false));
      // Open database again.
      bdb = BackupableDB.open(opt, bopt,
          dbFolder.getRoot().getAbsolutePath());
      // Values must have suffix V2 because of restoring latest backup.
      assertThat(new String(bdb.get("key1".getBytes()))).endsWith("V2");
      assertThat(new String(bdb.get("key2".getBytes()))).endsWith("V2");
    } finally {
      if (bdb != null) {
        bdb.close();
      }
      if (rdb != null) {
        rdb.dispose();
      }
      if (bopt != null) {
        bopt.dispose();
      }
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void restoreFromBackup()
      throws RocksDBException {
    Options opt = null;
    BackupableDBOptions bopt = null;
    BackupableDB bdb = null;
    RestoreBackupableDB rdb = null;
    try {
      opt = new Options().setCreateIfMissing(true);
      bopt = new BackupableDBOptions(
          backupFolder.getRoot().getAbsolutePath());
      assertThat(bopt.backupDir()).isEqualTo(
          backupFolder.getRoot().getAbsolutePath());
      // Open empty database.
      bdb = BackupableDB.open(opt, bopt,
          dbFolder.getRoot().getAbsolutePath());
      // Fill database with some test values
      prepareDatabase(bdb);
      bdb.createNewBackup(true);
      verifyNumberOfValidBackups(bdb, 1);
      bdb.put("key1".getBytes(), "valueV2".getBytes());
      bdb.put("key2".getBytes(), "valueV2".getBytes());
      bdb.createNewBackup(true);
      verifyNumberOfValidBackups(bdb, 2);
      bdb.put("key1".getBytes(), "valueV3".getBytes());
      bdb.put("key2".getBytes(), "valueV3".getBytes());
      assertThat(new String(bdb.get("key1".getBytes()))).endsWith("V3");
      assertThat(new String(bdb.get("key2".getBytes()))).endsWith("V3");
      bdb.close();

      // init RestoreBackupableDB
      rdb = new RestoreBackupableDB(bopt);
      List<BackupInfo> backupInfo = verifyNumberOfValidBackups(rdb, 2);
      // restore db from first backup
      rdb.restoreDBFromBackup(backupInfo.get(0).backupId(),
          dbFolder.getRoot().getAbsolutePath(),
          dbFolder.getRoot().getAbsolutePath(),
          new RestoreOptions(false));
      // Open database again.
      bdb = BackupableDB.open(opt, bopt,
          dbFolder.getRoot().getAbsolutePath());
      // Values must have suffix V2 because of restoring latest backup.
      assertThat(new String(bdb.get("key1".getBytes()))).endsWith("V1");
      assertThat(new String(bdb.get("key2".getBytes()))).endsWith("V1");
    } finally {
      if (bdb != null) {
        bdb.close();
      }
      if (rdb != null) {
        rdb.dispose();
      }
      if (bopt != null) {
        bopt.dispose();
      }
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  /**
   * Verify backups.
   *
   * @param bdb {@link BackupableDB} instance.
   * @param expectedNumberOfBackups numerical value
   * @throws RocksDBException thrown if an error occurs within the native
   *     part of the library.
   */
  private List<BackupInfo> verifyNumberOfValidBackups(BackupableDB bdb,
     int expectedNumberOfBackups) throws RocksDBException {
    // Verify that backups exist
    assertThat(bdb.getCorruptedBackups().length).
        isEqualTo(0);
    bdb.garbageCollect();
    List<BackupInfo> backupInfo = bdb.getBackupInfos();
    assertThat(backupInfo.size()).
        isEqualTo(expectedNumberOfBackups);
    return backupInfo;
  }

  /**
   * Verify backups.
   *
   * @param rdb {@link RestoreBackupableDB} instance.
   * @param expectedNumberOfBackups numerical value
   * @throws RocksDBException thrown if an error occurs within the native
   *     part of the library.
   */
  private List<BackupInfo> verifyNumberOfValidBackups(
      RestoreBackupableDB rdb, int expectedNumberOfBackups)
      throws RocksDBException {
    // Verify that backups exist
    assertThat(rdb.getCorruptedBackups().length).
        isEqualTo(0);
    rdb.garbageCollect();
    List<BackupInfo> backupInfo = rdb.getBackupInfos();
    assertThat(backupInfo.size()).
        isEqualTo(expectedNumberOfBackups);
    return backupInfo;
  }

  /**
   * Fill database with some test values.
   *
   * @param db {@link RocksDB} instance.
   * @throws RocksDBException thrown if an error occurs within the native
   *     part of the library.
   */
  private void prepareDatabase(RocksDB db)
      throws RocksDBException {
    db.put("key1".getBytes(), "valueV1".getBytes());
    db.put("key2".getBytes(), "valueV1".getBytes());
  }
}
