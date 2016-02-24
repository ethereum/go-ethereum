// Copyright (c) 2013, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
// Copyright (c) 2013 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#ifndef STORAGE_ROCKSDB_INCLUDE_COMPACTION_FILTER_H_
#define STORAGE_ROCKSDB_INCLUDE_COMPACTION_FILTER_H_

#include <memory>
#include <string>
#include <vector>

namespace rocksdb {

class Slice;
class SliceTransform;

// Context information of a compaction run
struct CompactionFilterContext {
  // Does this compaction run include all data files
  bool is_full_compaction;
  // Is this compaction requested by the client (true),
  // or is it occurring as an automatic compaction process
  bool is_manual_compaction;
};

// CompactionFilter allows an application to modify/delete a key-value at
// the time of compaction.

class CompactionFilter {
 public:
  // Context information of a compaction run
  struct Context {
    // Does this compaction run include all data files
    bool is_full_compaction;
    // Is this compaction requested by the client (true),
    // or is it occurring as an automatic compaction process
    bool is_manual_compaction;
  };

  virtual ~CompactionFilter() {}

  // The compaction process invokes this
  // method for kv that is being compacted. A return value
  // of false indicates that the kv should be preserved in the
  // output of this compaction run and a return value of true
  // indicates that this key-value should be removed from the
  // output of the compaction.  The application can inspect
  // the existing value of the key and make decision based on it.
  //
  // When the value is to be preserved, the application has the option
  // to modify the existing_value and pass it back through new_value.
  // value_changed needs to be set to true in this case.
  //
  // If multithreaded compaction is being used *and* a single CompactionFilter
  // instance was supplied via Options::compaction_filter, this method may be
  // called from different threads concurrently.  The application must ensure
  // that the call is thread-safe.
  //
  // If the CompactionFilter was created by a factory, then it will only ever
  // be used by a single thread that is doing the compaction run, and this
  // call does not need to be thread-safe.  However, multiple filters may be
  // in existence and operating concurrently.
  virtual bool Filter(int level,
                      const Slice& key,
                      const Slice& existing_value,
                      std::string* new_value,
                      bool* value_changed) const = 0;

  // Returns a name that identifies this compaction filter.
  // The name will be printed to LOG file on start up for diagnosis.
  virtual const char* Name() const = 0;
};

// CompactionFilterV2 that buffers kv pairs sharing the same prefix and let
// application layer to make individual decisions for all the kv pairs in the
// buffer.
class CompactionFilterV2 {
 public:
  virtual ~CompactionFilterV2() {}

  // The compaction process invokes this method for all the kv pairs
  // sharing the same prefix. It is a "roll-up" version of CompactionFilter.
  //
  // Each entry in the return vector indicates if the corresponding kv should
  // be preserved in the output of this compaction run. The application can
  // inspect the existing values of the keys and make decision based on it.
  //
  // When a value is to be preserved, the application has the option
  // to modify the entry in existing_values and pass it back through an entry
  // in new_values. A corresponding values_changed entry needs to be set to
  // true in this case. Note that the new_values vector contains only changed
  // values, i.e. new_values.size() <= values_changed.size().
  //
  typedef std::vector<Slice> SliceVector;
  virtual std::vector<bool> Filter(int level,
                                   const SliceVector& keys,
                                   const SliceVector& existing_values,
                                   std::vector<std::string>* new_values,
                                   std::vector<bool>* values_changed)
    const = 0;

  // Returns a name that identifies this compaction filter.
  // The name will be printed to LOG file on start up for diagnosis.
  virtual const char* Name() const = 0;
};

// Each compaction will create a new CompactionFilter allowing the
// application to know about different compactions
class CompactionFilterFactory {
 public:
  virtual ~CompactionFilterFactory() { }

  virtual std::unique_ptr<CompactionFilter> CreateCompactionFilter(
      const CompactionFilter::Context& context) = 0;

  // Returns a name that identifies this compaction filter factory.
  virtual const char* Name() const = 0;
};

// Default implementation of CompactionFilterFactory which does not
// return any filter
class DefaultCompactionFilterFactory : public CompactionFilterFactory {
 public:
  virtual std::unique_ptr<CompactionFilter> CreateCompactionFilter(
      const CompactionFilter::Context& context) override {
    return std::unique_ptr<CompactionFilter>(nullptr);
  }

  virtual const char* Name() const override {
    return "DefaultCompactionFilterFactory";
  }
};

// Each compaction will create a new CompactionFilterV2
//
// CompactionFilterFactoryV2 enables application to specify a prefix and use
// CompactionFilterV2 to filter kv-pairs in batches. Each batch contains all
// the kv-pairs sharing the same prefix.
//
// This is useful for applications that require grouping kv-pairs in
// compaction filter to make a purge/no-purge decision. For example, if the
// key prefix is user id and the rest of key represents the type of value.
// This batching filter will come in handy if the application's compaction
// filter requires knowledge of all types of values for any user id.
//
class CompactionFilterFactoryV2 {
 public:
  // NOTE: CompactionFilterFactoryV2 will not delete prefix_extractor
  explicit CompactionFilterFactoryV2(const SliceTransform* prefix_extractor)
    : prefix_extractor_(prefix_extractor) { }

  virtual ~CompactionFilterFactoryV2() { }

  virtual std::unique_ptr<CompactionFilterV2> CreateCompactionFilterV2(
    const CompactionFilterContext& context) = 0;

  // Returns a name that identifies this compaction filter factory.
  virtual const char* Name() const = 0;

  const SliceTransform* GetPrefixExtractor() const {
    return prefix_extractor_;
  }

  void SetPrefixExtractor(const SliceTransform* prefix_extractor) {
    prefix_extractor_ = prefix_extractor;
  }

 private:
  // Prefix extractor for compaction filter v2
  // Keys sharing the same prefix will be buffered internally.
  // Client can implement a Filter callback function to operate on the buffer
  const SliceTransform* prefix_extractor_;
};

// Default implementation of CompactionFilterFactoryV2 which does not
// return any filter
class DefaultCompactionFilterFactoryV2 : public CompactionFilterFactoryV2 {
 public:
  explicit DefaultCompactionFilterFactoryV2()
      : CompactionFilterFactoryV2(nullptr) { }

  virtual std::unique_ptr<CompactionFilterV2>
  CreateCompactionFilterV2(
      const CompactionFilterContext& context) override {
    return std::unique_ptr<CompactionFilterV2>(nullptr);
  }

  virtual const char* Name() const override {
    return "DefaultCompactionFilterFactoryV2";
  }
};

}  // namespace rocksdb

#endif  // STORAGE_ROCKSDB_INCLUDE_COMPACTION_FILTER_H_
