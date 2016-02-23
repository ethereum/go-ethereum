package org.rocksdb;

import org.junit.rules.ExternalResource;

/**
 * Resource to trigger garbage collection after each test
 * run.
 */
public class RocksMemoryResource extends ExternalResource {

  static {
    RocksDB.loadLibrary();
  }

  @Override
  protected void after() {
    System.gc();
    System.runFinalization();
  }
}
