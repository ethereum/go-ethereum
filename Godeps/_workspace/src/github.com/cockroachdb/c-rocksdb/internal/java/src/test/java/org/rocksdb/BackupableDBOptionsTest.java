// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import org.junit.ClassRule;
import org.junit.Rule;
import org.junit.Test;
import org.junit.rules.ExpectedException;

import java.util.Random;

import static org.assertj.core.api.Assertions.assertThat;

public class BackupableDBOptionsTest {

  private final static String ARBITRARY_PATH = "/tmp";

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Rule
  public ExpectedException exception = ExpectedException.none();

  public static final Random rand = PlatformRandomHelper.
      getPlatformSpecificRandomFactory();

  @Test
  public void backupDir() {
    BackupableDBOptions backupableDBOptions = null;
    try {
      backupableDBOptions = new BackupableDBOptions(ARBITRARY_PATH);
      assertThat(backupableDBOptions.backupDir()).
          isEqualTo(ARBITRARY_PATH);
    } finally {
      if (backupableDBOptions != null) {
        backupableDBOptions.dispose();
      }
    }
  }

  @Test
  public void shareTableFiles() {
    BackupableDBOptions backupableDBOptions = null;
    try {
      backupableDBOptions = new BackupableDBOptions(ARBITRARY_PATH);
      boolean value = rand.nextBoolean();
      backupableDBOptions.setShareTableFiles(value);
      assertThat(backupableDBOptions.shareTableFiles()).
          isEqualTo(value);
    } finally {
      if (backupableDBOptions != null) {
        backupableDBOptions.dispose();
      }
    }
  }

  @Test
  public void sync() {
    BackupableDBOptions backupableDBOptions = null;
    try {
      backupableDBOptions = new BackupableDBOptions(ARBITRARY_PATH);
      boolean value = rand.nextBoolean();
      backupableDBOptions.setSync(value);
      assertThat(backupableDBOptions.sync()).isEqualTo(value);
    } finally {
      if (backupableDBOptions != null) {
        backupableDBOptions.dispose();
      }
    }
  }

  @Test
  public void destroyOldData() {
    BackupableDBOptions backupableDBOptions = null;
    try {
      backupableDBOptions = new BackupableDBOptions(ARBITRARY_PATH);
      boolean value = rand.nextBoolean();
      backupableDBOptions.setDestroyOldData(value);
      assertThat(backupableDBOptions.destroyOldData()).
          isEqualTo(value);
    } finally {
      if (backupableDBOptions != null) {
        backupableDBOptions.dispose();
      }
    }
  }

  @Test
  public void backupLogFiles() {
    BackupableDBOptions backupableDBOptions = null;
    try {
      backupableDBOptions = new BackupableDBOptions(ARBITRARY_PATH);
      boolean value = rand.nextBoolean();
      backupableDBOptions.setBackupLogFiles(value);
      assertThat(backupableDBOptions.backupLogFiles()).
          isEqualTo(value);
    } finally {
      if (backupableDBOptions != null) {
        backupableDBOptions.dispose();
      }
    }
  }

  @Test
  public void backupRateLimit() {
    BackupableDBOptions backupableDBOptions = null;
    try {
      backupableDBOptions = new BackupableDBOptions(ARBITRARY_PATH);
      long value = Math.abs(rand.nextLong());
      backupableDBOptions.setBackupRateLimit(value);
      assertThat(backupableDBOptions.backupRateLimit()).
          isEqualTo(value);
      // negative will be mapped to 0
      backupableDBOptions.setBackupRateLimit(-1);
      assertThat(backupableDBOptions.backupRateLimit()).
          isEqualTo(0);
    } finally {
      if (backupableDBOptions != null) {
        backupableDBOptions.dispose();
      }
    }
  }

  @Test
  public void restoreRateLimit() {
    BackupableDBOptions backupableDBOptions = null;
    try {
      backupableDBOptions = new BackupableDBOptions(ARBITRARY_PATH);
      long value = Math.abs(rand.nextLong());
      backupableDBOptions.setRestoreRateLimit(value);
      assertThat(backupableDBOptions.restoreRateLimit()).
          isEqualTo(value);
      // negative will be mapped to 0
      backupableDBOptions.setRestoreRateLimit(-1);
      assertThat(backupableDBOptions.restoreRateLimit()).
          isEqualTo(0);
    } finally {
      if (backupableDBOptions != null) {
        backupableDBOptions.dispose();
      }
    }
  }

  @Test
  public void shareFilesWithChecksum() {
    BackupableDBOptions backupableDBOptions = null;
    try {
      backupableDBOptions = new BackupableDBOptions(ARBITRARY_PATH);
      boolean value = rand.nextBoolean();
      backupableDBOptions.setShareFilesWithChecksum(value);
      assertThat(backupableDBOptions.shareFilesWithChecksum()).
          isEqualTo(value);
    } finally {
      if (backupableDBOptions != null) {
        backupableDBOptions.dispose();
      }
    }
  }

  @Test
  public void failBackupDirIsNull() {
    exception.expect(IllegalArgumentException.class);
    new BackupableDBOptions(null);
  }

  @Test
  public void failBackupDirIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.backupDir();
  }

  @Test
  public void failSetShareTableFilesIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.setShareTableFiles(true);
  }

  @Test
  public void failShareTableFilesIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.shareTableFiles();
  }

  @Test
  public void failSetSyncIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.setSync(true);
  }

  @Test
  public void failSyncIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.sync();
  }

  @Test
  public void failSetDestroyOldDataIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.setDestroyOldData(true);
  }

  @Test
  public void failDestroyOldDataIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.destroyOldData();
  }

  @Test
  public void failSetBackupLogFilesIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.setBackupLogFiles(true);
  }

  @Test
  public void failBackupLogFilesIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.backupLogFiles();
  }

  @Test
  public void failSetBackupRateLimitIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.setBackupRateLimit(1);
  }

  @Test
  public void failBackupRateLimitIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.backupRateLimit();
  }

  @Test
  public void failSetRestoreRateLimitIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.setRestoreRateLimit(1);
  }

  @Test
  public void failRestoreRateLimitIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.restoreRateLimit();
  }

  @Test
  public void failSetShareFilesWithChecksumIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.setShareFilesWithChecksum(true);
  }

  @Test
  public void failShareFilesWithChecksumIfDisposed(){
    BackupableDBOptions options = setupUninitializedBackupableDBOptions(
        exception);
    options.shareFilesWithChecksum();
  }

  private BackupableDBOptions setupUninitializedBackupableDBOptions(
      ExpectedException exception) {
    BackupableDBOptions backupableDBOptions =
        new BackupableDBOptions(ARBITRARY_PATH);
    backupableDBOptions.dispose();
    exception.expect(AssertionError.class);
    return backupableDBOptions;
  }
}
