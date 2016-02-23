// Copyright (c) 2015, Facebook, Inc.  All rights reserved.
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

public class BackupEngineTest {

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
    RocksDB db = null;
    try {
      opt = new Options().setCreateIfMissing(true);
      // Open empty database.
      db = RocksDB.open(opt,
          dbFolder.getRoot().getAbsolutePath());
      // Fill database with some test values
      prepareDatabase(db);
      // Create two backups
      BackupableDBOptions bopt = null;
      try {
        bopt = new BackupableDBOptions(
          backupFolder.getRoot().getAbsolutePath());
        try(final BackupEngine be = BackupEngine.open(opt.getEnv(), bopt)) {
          be.createNewBackup(db, false);
          be.createNewBackup(db, true);
          verifyNumberOfValidBackups(be, 2);
        }
      } finally {
        if(bopt != null) {
          bopt.dispose();
        }
      }
    } finally {
      if (db != null) {
        db.close();
      }
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void deleteBackup() throws RocksDBException {
    Options opt = null;
    RocksDB db = null;
    try {
      opt = new Options().setCreateIfMissing(true);
      // Open empty database.
      db = RocksDB.open(opt,
          dbFolder.getRoot().getAbsolutePath());
      // Fill database with some test values
      prepareDatabase(db);
      // Create two backups
      BackupableDBOptions bopt = null;
      try {
        bopt = new BackupableDBOptions(
            backupFolder.getRoot().getAbsolutePath());
        try(final BackupEngine be = BackupEngine.open(opt.getEnv(), bopt)) {
          be.createNewBackup(db, false);
          be.createNewBackup(db, true);
          final List<BackupInfo> backupInfo =
              verifyNumberOfValidBackups(be, 2);
          // Delete the first backup
          be.deleteBackup(backupInfo.get(0).backupId());
          final List<BackupInfo> newBackupInfo =
              verifyNumberOfValidBackups(be, 1);

          // The second backup must remain.
          assertThat(newBackupInfo.get(0).backupId()).
              isEqualTo(backupInfo.get(1).backupId());
        }
      } finally {
        if(bopt != null) {
          bopt.dispose();
        }
      }
    } finally {
      if (db != null) {
        db.close();
      }
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void purgeOldBackups() throws RocksDBException {
    Options opt = null;
    RocksDB db = null;
    try {
      opt = new Options().setCreateIfMissing(true);
      // Open empty database.
      db = RocksDB.open(opt,
          dbFolder.getRoot().getAbsolutePath());
      // Fill database with some test values
      prepareDatabase(db);
      // Create four backups
      BackupableDBOptions bopt = null;
      try {
        bopt = new BackupableDBOptions(
            backupFolder.getRoot().getAbsolutePath());
        try(final BackupEngine be = BackupEngine.open(opt.getEnv(), bopt)) {
          be.createNewBackup(db, false);
          be.createNewBackup(db, true);
          be.createNewBackup(db, true);
          be.createNewBackup(db, true);
          final List<BackupInfo> backupInfo =
              verifyNumberOfValidBackups(be, 4);
          // Delete everything except the latest backup
          be.purgeOldBackups(1);
          final List<BackupInfo> newBackupInfo =
              verifyNumberOfValidBackups(be, 1);
          // The latest backup must remain.
          assertThat(newBackupInfo.get(0).backupId()).
              isEqualTo(backupInfo.get(3).backupId());
        }
      } finally {
        if(bopt != null) {
          bopt.dispose();
        }
      }
    } finally {
      if (db != null) {
        db.close();
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
    RocksDB db = null;
    try {
      opt = new Options().setCreateIfMissing(true);
      // Open empty database.
      db = RocksDB.open(opt,
          dbFolder.getRoot().getAbsolutePath());
      // Fill database with some test values
      prepareDatabase(db);
      BackupableDBOptions bopt = null;
      try {
        bopt = new BackupableDBOptions(
            backupFolder.getRoot().getAbsolutePath());
        try (final BackupEngine be = BackupEngine.open(opt.getEnv(), bopt)) {
          be.createNewBackup(db, true);
          verifyNumberOfValidBackups(be, 1);
          db.put("key1".getBytes(), "valueV2".getBytes());
          db.put("key2".getBytes(), "valueV2".getBytes());
          be.createNewBackup(db, true);
          verifyNumberOfValidBackups(be, 2);
          db.put("key1".getBytes(), "valueV3".getBytes());
          db.put("key2".getBytes(), "valueV3".getBytes());
          assertThat(new String(db.get("key1".getBytes()))).endsWith("V3");
          assertThat(new String(db.get("key2".getBytes()))).endsWith("V3");

          db.close();

          verifyNumberOfValidBackups(be, 2);
          // restore db from latest backup
          be.restoreDbFromLatestBackup(dbFolder.getRoot().getAbsolutePath(),
              dbFolder.getRoot().getAbsolutePath(),
              new RestoreOptions(false));
          // Open database again.
          db = RocksDB.open(opt,
              dbFolder.getRoot().getAbsolutePath());
          // Values must have suffix V2 because of restoring latest backup.
          assertThat(new String(db.get("key1".getBytes()))).endsWith("V2");
          assertThat(new String(db.get("key2".getBytes()))).endsWith("V2");
        }
      } finally {
        if(bopt != null) {
          bopt.dispose();
        }
      }
    } finally {
      if (db != null) {
        db.close();
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
    RocksDB db = null;
    try {
      opt = new Options().setCreateIfMissing(true);
      // Open empty database.
      db = RocksDB.open(opt,
          dbFolder.getRoot().getAbsolutePath());
      // Fill database with some test values
      prepareDatabase(db);
      BackupableDBOptions bopt = null;
      try {
        bopt = new BackupableDBOptions(
            backupFolder.getRoot().getAbsolutePath());
        try (final BackupEngine be = BackupEngine.open(opt.getEnv(), bopt)) {
          be.createNewBackup(db, true);
          verifyNumberOfValidBackups(be, 1);
          db.put("key1".getBytes(), "valueV2".getBytes());
          db.put("key2".getBytes(), "valueV2".getBytes());
          be.createNewBackup(db, true);
          verifyNumberOfValidBackups(be, 2);
          db.put("key1".getBytes(), "valueV3".getBytes());
          db.put("key2".getBytes(), "valueV3".getBytes());
          assertThat(new String(db.get("key1".getBytes()))).endsWith("V3");
          assertThat(new String(db.get("key2".getBytes()))).endsWith("V3");

          //close the database
          db.close();

          //restore the backup
          List<BackupInfo> backupInfo = verifyNumberOfValidBackups(be, 2);
          // restore db from first backup
          be.restoreDbFromBackup(backupInfo.get(0).backupId(),
              dbFolder.getRoot().getAbsolutePath(),
              dbFolder.getRoot().getAbsolutePath(),
              new RestoreOptions(false));
          // Open database again.
          db = RocksDB.open(opt,
              dbFolder.getRoot().getAbsolutePath());
          // Values must have suffix V2 because of restoring latest backup.
          assertThat(new String(db.get("key1".getBytes()))).endsWith("V1");
          assertThat(new String(db.get("key2".getBytes()))).endsWith("V1");
        }
      } finally {
        if(bopt != null) {
          bopt.dispose();
        }
      }
    } finally {
      if (db != null) {
        db.close();
      }
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  /**
   * Verify backups.
   *
   * @param be {@link BackupEngine} instance.
   * @param expectedNumberOfBackups numerical value
   * @throws RocksDBException thrown if an error occurs within the native
   *     part of the library.
   */
  private List<BackupInfo> verifyNumberOfValidBackups(final BackupEngine be,
      final int expectedNumberOfBackups) throws RocksDBException {
    // Verify that backups exist
    assertThat(be.getCorruptedBackups().length).
        isEqualTo(0);
    be.garbageCollect();
    final List<BackupInfo> backupInfo = be.getBackupInfo();
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
  private void prepareDatabase(final RocksDB db)
      throws RocksDBException {
    db.put("key1".getBytes(), "valueV1".getBytes());
    db.put("key2".getBytes(), "valueV1".getBytes());
  }
}
