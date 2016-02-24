package org.rocksdb;

import org.junit.ClassRule;
import org.junit.Rule;
import org.junit.Test;
import org.junit.rules.TemporaryFolder;

import java.io.IOException;

import static java.nio.file.Files.readAllBytes;
import static java.nio.file.Paths.get;
import static org.assertj.core.api.Assertions.assertThat;

public class InfoLogLevelTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Rule
  public TemporaryFolder dbFolder = new TemporaryFolder();

  @Test
  public void testInfoLogLevel() throws RocksDBException,
      IOException {
    RocksDB db = null;
    try {
      db = RocksDB.open(dbFolder.getRoot().getAbsolutePath());
      db.put("key".getBytes(), "value".getBytes());
      assertThat(getLogContents()).isNotEmpty();
    } finally {
      if (db != null) {
        db.close();
      }
    }
  }

  @Test
     public void testFatalLogLevel() throws RocksDBException,
      IOException {
    RocksDB db = null;
    Options options = null;
    try {
      options = new Options().
          setCreateIfMissing(true).
          setInfoLogLevel(InfoLogLevel.FATAL_LEVEL);
      assertThat(options.infoLogLevel()).
          isEqualTo(InfoLogLevel.FATAL_LEVEL);
      db = RocksDB.open(options,
          dbFolder.getRoot().getAbsolutePath());
      db.put("key".getBytes(), "value".getBytes());
      // As InfoLogLevel is set to FATAL_LEVEL, here we expect the log
      // content to be empty.
      assertThat(getLogContents()).isEmpty();
    } finally {
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test
  public void testFatalLogLevelWithDBOptions()
      throws RocksDBException, IOException {
    RocksDB db = null;
    Options options = null;
    DBOptions dbOptions = null;
    try {
      dbOptions = new DBOptions().
          setInfoLogLevel(InfoLogLevel.FATAL_LEVEL);
      options = new Options(dbOptions,
          new ColumnFamilyOptions()).
          setCreateIfMissing(true);
      assertThat(dbOptions.infoLogLevel()).
          isEqualTo(InfoLogLevel.FATAL_LEVEL);
      assertThat(options.infoLogLevel()).
          isEqualTo(InfoLogLevel.FATAL_LEVEL);
      db = RocksDB.open(options,
          dbFolder.getRoot().getAbsolutePath());
      db.put("key".getBytes(), "value".getBytes());
      assertThat(getLogContents()).isEmpty();
    } finally {
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
      if (dbOptions != null) {
        dbOptions.dispose();
      }
    }
  }

  @Test(expected = IllegalArgumentException.class)
  public void failIfIllegalByteValueProvided() {
    InfoLogLevel.getInfoLogLevel((byte)-1);
  }

  @Test
  public void valueOf() {
    assertThat(InfoLogLevel.valueOf("DEBUG_LEVEL")).
        isEqualTo(InfoLogLevel.DEBUG_LEVEL);
  }

  /**
   * Read LOG file contents into String.
   *
   * @return LOG file contents as String.
   * @throws IOException if file is not found.
   */
  private String getLogContents() throws IOException {
    return new String(readAllBytes(get(
        dbFolder.getRoot().getAbsolutePath()+ "/LOG")));
  }
}
