package org.rocksdb;

/**
 * RocksDB log levels.
 */
public enum InfoLogLevel {
  DEBUG_LEVEL((byte)0),
  INFO_LEVEL((byte)1),
  WARN_LEVEL((byte)2),
  ERROR_LEVEL((byte)3),
  FATAL_LEVEL((byte)4),
  NUM_INFO_LOG_LEVELS((byte)5);

  private final byte value_;

  private InfoLogLevel(byte value) {
    value_ = value;
  }

  /**
   * Returns the byte value of the enumerations value
   *
   * @return byte representation
   */
  public byte getValue() {
    return value_;
  }

  /**
   * Get InfoLogLevel by byte value.
   *
   * @param value byte representation of InfoLogLevel.
   *
   * @return {@link org.rocksdb.InfoLogLevel} instance or null.
   * @throws java.lang.IllegalArgumentException if an invalid
   *     value is provided.
   */
  public static InfoLogLevel getInfoLogLevel(byte value) {
    for (InfoLogLevel infoLogLevel : InfoLogLevel.values()) {
      if (infoLogLevel.getValue() == value){
        return infoLogLevel;
      }
    }
    throw new IllegalArgumentException(
        "Illegal value provided for InfoLogLevel.");
  }
}
