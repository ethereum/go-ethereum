//  Copyright (c) 2015, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#ifndef ROCKSDB_LITE
namespace rocksdb {

// A factory of a table property collector that marks a SST
// file as need-compaction when it observe at least "D" deletion
// entries in any "N" consecutive entires.
class CompactOnDeletionCollectorFactory
    : public TablePropertiesCollectorFactory {
 public:
  // A factory of a table property collector that marks a SST
  // file as need-compaction when it observe at least "D" deletion
  // entries in any "N" consecutive entires.
  //
  // @param sliding_window_size "N"
  // @param deletion_trigger "D"
  CompactOnDeletionCollectorFactory(
      size_t sliding_window_size,
      size_t deletion_trigger) :
          sliding_window_size_(sliding_window_size),
          deletion_trigger_(deletion_trigger) {}

  virtual ~CompactOnDeletionCollectorFactory() {}

  virtual TablePropertiesCollector* CreateTablePropertiesCollector() override;

  virtual const char* Name() const override {
    return "CompactOnDeletionCollector";
  }

 private:
  size_t sliding_window_size_;
  size_t deletion_trigger_;
};

class CompactOnDeletionCollector : public TablePropertiesCollector {
 public:
  CompactOnDeletionCollector(
      size_t sliding_window_size,
      size_t deletion_trigger);

  // AddUserKey() will be called when a new key/value pair is inserted into the
  // table.
  // @params key    the user key that is inserted into the table.
  // @params value  the value that is inserted into the table.
  // @params file_size  file size up to now
  virtual Status AddUserKey(const Slice& key, const Slice& value,
                            EntryType type, SequenceNumber seq,
                            uint64_t file_size) override;

  // Finish() will be called when a table has already been built and is ready
  // for writing the properties block.
  // @params properties  User will add their collected statistics to
  // `properties`.
  virtual Status Finish(UserCollectedProperties* properties) override {
    Reset();
    return Status::OK();
  }

  // Return the human-readable properties, where the key is property name and
  // the value is the human-readable form of value.
  virtual UserCollectedProperties GetReadableProperties() const override {
    return UserCollectedProperties();
  }

  // The name of the properties collector can be used for debugging purpose.
  virtual const char* Name() const override {
    return "CompactOnDeletionCollector";
  }

  // EXPERIMENTAL Return whether the output file should be further compacted
  virtual bool NeedCompact() const override {
    return need_compaction_;
  }

  static const int kNumBuckets = 128;

 private:
  void Reset();

  // A ring buffer that used to count the number of deletion entries for every
  // "bucket_size_" keys.
  size_t num_deletions_in_buckets_[kNumBuckets];
  // the number of keys in a bucket
  size_t bucket_size_;

  size_t current_bucket_;
  size_t num_keys_in_current_bucket_;
  size_t num_deletions_in_observation_window_;
  size_t deletion_trigger_;
  // true if the current SST file needs to be compacted.
  bool need_compaction_;
};
}  // namespace rocksdb
#endif  // !ROCKSDB_LITE
