// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#pragma once
#include <string>
#include "rocksdb/db.h"

#ifdef _WIN32
// Windows API macro interference
#undef DeleteFile
#endif


namespace rocksdb {

// This class contains APIs to stack rocksdb wrappers.Eg. Stack TTL over base d
class StackableDB : public DB {
 public:
  // StackableDB is the owner of db now!
  explicit StackableDB(DB* db) : db_(db) {}

  ~StackableDB() {
    delete db_;
  }

  virtual DB* GetBaseDB() {
    return db_;
  }

  virtual DB* GetRootDB() override { return db_->GetRootDB(); }

  virtual Status CreateColumnFamily(const ColumnFamilyOptions& options,
                                    const std::string& column_family_name,
                                    ColumnFamilyHandle** handle) override {
    return db_->CreateColumnFamily(options, column_family_name, handle);
  }

  virtual Status DropColumnFamily(ColumnFamilyHandle* column_family) override {
    return db_->DropColumnFamily(column_family);
  }

  using DB::Put;
  virtual Status Put(const WriteOptions& options,
                     ColumnFamilyHandle* column_family, const Slice& key,
                     const Slice& val) override {
    return db_->Put(options, column_family, key, val);
  }

  using DB::Get;
  virtual Status Get(const ReadOptions& options,
                     ColumnFamilyHandle* column_family, const Slice& key,
                     std::string* value) override {
    return db_->Get(options, column_family, key, value);
  }

  using DB::MultiGet;
  virtual std::vector<Status> MultiGet(
      const ReadOptions& options,
      const std::vector<ColumnFamilyHandle*>& column_family,
      const std::vector<Slice>& keys,
      std::vector<std::string>* values) override {
    return db_->MultiGet(options, column_family, keys, values);
  }

  using DB::KeyMayExist;
  virtual bool KeyMayExist(const ReadOptions& options,
                           ColumnFamilyHandle* column_family, const Slice& key,
                           std::string* value,
                           bool* value_found = nullptr) override {
    return db_->KeyMayExist(options, column_family, key, value, value_found);
  }

  using DB::Delete;
  virtual Status Delete(const WriteOptions& wopts,
                        ColumnFamilyHandle* column_family,
                        const Slice& key) override {
    return db_->Delete(wopts, column_family, key);
  }

  using DB::Merge;
  virtual Status Merge(const WriteOptions& options,
                       ColumnFamilyHandle* column_family, const Slice& key,
                       const Slice& value) override {
    return db_->Merge(options, column_family, key, value);
  }


  virtual Status Write(const WriteOptions& opts, WriteBatch* updates)
    override {
      return db_->Write(opts, updates);
  }

  using DB::NewIterator;
  virtual Iterator* NewIterator(const ReadOptions& opts,
                                ColumnFamilyHandle* column_family) override {
    return db_->NewIterator(opts, column_family);
  }

  virtual Status NewIterators(
      const ReadOptions& options,
      const std::vector<ColumnFamilyHandle*>& column_families,
      std::vector<Iterator*>* iterators) override {
    return db_->NewIterators(options, column_families, iterators);
  }


  virtual const Snapshot* GetSnapshot() override {
    return db_->GetSnapshot();
  }

  virtual void ReleaseSnapshot(const Snapshot* snapshot) override {
    return db_->ReleaseSnapshot(snapshot);
  }

  using DB::GetProperty;
  virtual bool GetProperty(ColumnFamilyHandle* column_family,
                           const Slice& property, std::string* value) override {
    return db_->GetProperty(column_family, property, value);
  }

  using DB::GetIntProperty;
  virtual bool GetIntProperty(ColumnFamilyHandle* column_family,
                              const Slice& property, uint64_t* value) override {
    return db_->GetIntProperty(column_family, property, value);
  }

  using DB::GetApproximateSizes;
  virtual void GetApproximateSizes(ColumnFamilyHandle* column_family,
                                   const Range* r, int n, uint64_t* sizes,
                                   bool include_memtable = false) override {
      return db_->GetApproximateSizes(column_family, r, n, sizes);
  }

  using DB::CompactRange;
  virtual Status CompactRange(const CompactRangeOptions& options,
                              ColumnFamilyHandle* column_family,
                              const Slice* begin, const Slice* end) override {
    return db_->CompactRange(options, column_family, begin, end);
  }

  using DB::CompactFiles;
  virtual Status CompactFiles(
      const CompactionOptions& compact_options,
      ColumnFamilyHandle* column_family,
      const std::vector<std::string>& input_file_names,
      const int output_level, const int output_path_id = -1) override {
    return db_->CompactFiles(
        compact_options, column_family, input_file_names,
        output_level, output_path_id);
  }

  using DB::NumberLevels;
  virtual int NumberLevels(ColumnFamilyHandle* column_family) override {
    return db_->NumberLevels(column_family);
  }

  using DB::MaxMemCompactionLevel;
  virtual int MaxMemCompactionLevel(ColumnFamilyHandle* column_family)
      override {
    return db_->MaxMemCompactionLevel(column_family);
  }

  using DB::Level0StopWriteTrigger;
  virtual int Level0StopWriteTrigger(ColumnFamilyHandle* column_family)
      override {
    return db_->Level0StopWriteTrigger(column_family);
  }

  virtual const std::string& GetName() const override {
    return db_->GetName();
  }

  virtual Env* GetEnv() const override {
    return db_->GetEnv();
  }

  using DB::GetOptions;
  virtual const Options& GetOptions(ColumnFamilyHandle* column_family) const
      override {
    return db_->GetOptions(column_family);
  }

  using DB::GetDBOptions;
  virtual const DBOptions& GetDBOptions() const override {
    return db_->GetDBOptions();
  }

  using DB::Flush;
  virtual Status Flush(const FlushOptions& fopts,
                       ColumnFamilyHandle* column_family) override {
    return db_->Flush(fopts, column_family);
  }

  virtual Status SyncWAL() override {
    return db_->SyncWAL();
  }

#ifndef ROCKSDB_LITE

  virtual Status DisableFileDeletions() override {
    return db_->DisableFileDeletions();
  }

  virtual Status EnableFileDeletions(bool force) override {
    return db_->EnableFileDeletions(force);
  }

  virtual void GetLiveFilesMetaData(
      std::vector<LiveFileMetaData>* metadata) override {
    db_->GetLiveFilesMetaData(metadata);
  }

  virtual void GetColumnFamilyMetaData(
      ColumnFamilyHandle *column_family,
      ColumnFamilyMetaData* cf_meta) override {
    db_->GetColumnFamilyMetaData(column_family, cf_meta);
  }

#endif  // ROCKSDB_LITE

  virtual Status GetLiveFiles(std::vector<std::string>& vec, uint64_t* mfs,
                              bool flush_memtable = true) override {
      return db_->GetLiveFiles(vec, mfs, flush_memtable);
  }

  virtual SequenceNumber GetLatestSequenceNumber() const override {
    return db_->GetLatestSequenceNumber();
  }

  virtual Status GetSortedWalFiles(VectorLogPtr& files) override {
    return db_->GetSortedWalFiles(files);
  }

  virtual Status DeleteFile(std::string name) override {
    return db_->DeleteFile(name);
  }

  virtual Status GetDbIdentity(std::string& identity) const override {
    return db_->GetDbIdentity(identity);
  }

  using DB::SetOptions;
  virtual Status SetOptions(
    const std::unordered_map<std::string, std::string>& new_options) override {
    return db_->SetOptions(new_options);
  }

  using DB::GetPropertiesOfAllTables;
  virtual Status GetPropertiesOfAllTables(
      ColumnFamilyHandle* column_family,
      TablePropertiesCollection* props) override {
    return db_->GetPropertiesOfAllTables(column_family, props);
  }

  virtual Status GetUpdatesSince(
      SequenceNumber seq_number, unique_ptr<TransactionLogIterator>* iter,
      const TransactionLogIterator::ReadOptions& read_options) override {
    return db_->GetUpdatesSince(seq_number, iter, read_options);
  }

  virtual ColumnFamilyHandle* DefaultColumnFamily() const override {
    return db_->DefaultColumnFamily();
  }

 protected:
  DB* db_;
};

} //  namespace rocksdb
