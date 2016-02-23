// Copyright (c) 2015, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include "utilities/transactions/transaction_base.h"

#include "db/column_family.h"
#include "rocksdb/comparator.h"
#include "rocksdb/db.h"
#include "rocksdb/status.h"
#include "util/string_util.h"

namespace rocksdb {

TransactionBaseImpl::TransactionBaseImpl(DB* db,
                                         const WriteOptions& write_options)
    : db_(db),
      write_options_(write_options),
      cmp_(GetColumnFamilyUserComparator(db->DefaultColumnFamily())),
      write_batch_(new WriteBatchWithIndex(cmp_, 0, true)),
      start_time_(db_->GetEnv()->NowMicros()) {}

TransactionBaseImpl::~TransactionBaseImpl() {}

void TransactionBaseImpl::Clear() {
  save_points_.reset(nullptr);
  write_batch_->Clear();
  tracked_keys_.clear();
  num_puts_ = 0;
  num_deletes_ = 0;
  num_merges_ = 0;
}

void TransactionBaseImpl::SetSnapshot() {
  snapshot_.reset(new ManagedSnapshot(db_));
}

Status TransactionBaseImpl::TryLock(ColumnFamilyHandle* column_family,
                                    const SliceParts& key, bool untracked) {
  size_t key_size = 0;
  for (int i = 0; i < key.num_parts; ++i) {
    key_size += key.parts[i].size();
  }

  std::string str;
  str.reserve(key_size);

  for (int i = 0; i < key.num_parts; ++i) {
    str.append(key.parts[i].data(), key.parts[i].size());
  }

  return TryLock(column_family, str, untracked);
}

void TransactionBaseImpl::SetSavePoint() {
  if (save_points_ == nullptr) {
    save_points_.reset(new std::stack<TransactionBaseImpl::SavePoint>());
  }
  save_points_->emplace(snapshot_, num_puts_, num_deletes_, num_merges_);
  write_batch_->SetSavePoint();
}

Status TransactionBaseImpl::RollbackToSavePoint() {
  if (save_points_ != nullptr && save_points_->size() > 0) {
    // Restore saved SavePoint
    TransactionBaseImpl::SavePoint& save_point = save_points_->top();
    snapshot_ = save_point.snapshot_;
    num_puts_ = save_point.num_puts_;
    num_deletes_ = save_point.num_deletes_;
    num_merges_ = save_point.num_merges_;

    // Rollback batch
    Status s = write_batch_->RollbackToSavePoint();
    assert(s.ok());

    // Rollback any keys that were tracked since the last savepoint
    const TransactionKeyMap* key_map = GetTrackedKeysSinceSavePoint();
    assert(key_map);
    for (auto& key_map_iter : *key_map) {
      uint32_t column_family_id = key_map_iter.first;
      auto& keys = key_map_iter.second;

      for (auto& key_iter : keys) {
        const std::string& key = key_iter.first;
        tracked_keys_[column_family_id].erase(key);
      }
    }

    save_points_->pop();

    return s;
  } else {
    assert(write_batch_->RollbackToSavePoint().IsNotFound());
    return Status::NotFound();
  }
}

Status TransactionBaseImpl::Get(const ReadOptions& read_options,
                                ColumnFamilyHandle* column_family,
                                const Slice& key, std::string* value) {
  return write_batch_->GetFromBatchAndDB(db_, read_options, column_family, key,
                                         value);
}

Status TransactionBaseImpl::GetForUpdate(const ReadOptions& read_options,
                                         ColumnFamilyHandle* column_family,
                                         const Slice& key, std::string* value) {
  Status s = TryLock(column_family, key);

  if (s.ok() && value != nullptr) {
    s = Get(read_options, column_family, key, value);
  }
  return s;
}

std::vector<Status> TransactionBaseImpl::MultiGet(
    const ReadOptions& read_options,
    const std::vector<ColumnFamilyHandle*>& column_family,
    const std::vector<Slice>& keys, std::vector<std::string>* values) {
  size_t num_keys = keys.size();
  values->resize(num_keys);

  std::vector<Status> stat_list(num_keys);
  for (size_t i = 0; i < num_keys; ++i) {
    std::string* value = values ? &(*values)[i] : nullptr;
    stat_list[i] = Get(read_options, column_family[i], keys[i], value);
  }

  return stat_list;
}

std::vector<Status> TransactionBaseImpl::MultiGetForUpdate(
    const ReadOptions& read_options,
    const std::vector<ColumnFamilyHandle*>& column_family,
    const std::vector<Slice>& keys, std::vector<std::string>* values) {
  // Regardless of whether the MultiGet succeeded, track these keys.
  size_t num_keys = keys.size();
  values->resize(num_keys);

  // Lock all keys
  for (size_t i = 0; i < num_keys; ++i) {
    Status s = TryLock(column_family[i], keys[i]);
    if (!s.ok()) {
      // Fail entire multiget if we cannot lock all keys
      return std::vector<Status>(num_keys, s);
    }
  }

  // TODO(agiardullo): optimize multiget?
  std::vector<Status> stat_list(num_keys);
  for (size_t i = 0; i < num_keys; ++i) {
    std::string* value = values ? &(*values)[i] : nullptr;
    stat_list[i] = Get(read_options, column_family[i], keys[i], value);
  }

  return stat_list;
}

Iterator* TransactionBaseImpl::GetIterator(const ReadOptions& read_options) {
  Iterator* db_iter = db_->NewIterator(read_options);
  assert(db_iter);

  return write_batch_->NewIteratorWithBase(db_iter);
}

Iterator* TransactionBaseImpl::GetIterator(const ReadOptions& read_options,
                                           ColumnFamilyHandle* column_family) {
  Iterator* db_iter = db_->NewIterator(read_options, column_family);
  assert(db_iter);

  return write_batch_->NewIteratorWithBase(column_family, db_iter);
}

Status TransactionBaseImpl::Put(ColumnFamilyHandle* column_family,
                                const Slice& key, const Slice& value) {
  Status s = TryLock(column_family, key);

  if (s.ok()) {
    write_batch_->Put(column_family, key, value);
    num_puts_++;
  }

  return s;
}

Status TransactionBaseImpl::Put(ColumnFamilyHandle* column_family,
                                const SliceParts& key,
                                const SliceParts& value) {
  Status s = TryLock(column_family, key);

  if (s.ok()) {
    write_batch_->Put(column_family, key, value);
    num_puts_++;
  }

  return s;
}

Status TransactionBaseImpl::Merge(ColumnFamilyHandle* column_family,
                                  const Slice& key, const Slice& value) {
  Status s = TryLock(column_family, key);

  if (s.ok()) {
    write_batch_->Merge(column_family, key, value);
    num_merges_++;
  }

  return s;
}

Status TransactionBaseImpl::Delete(ColumnFamilyHandle* column_family,
                                   const Slice& key) {
  Status s = TryLock(column_family, key);

  if (s.ok()) {
    write_batch_->Delete(column_family, key);
    num_deletes_++;
  }

  return s;
}

Status TransactionBaseImpl::Delete(ColumnFamilyHandle* column_family,
                                   const SliceParts& key) {
  Status s = TryLock(column_family, key);

  if (s.ok()) {
    write_batch_->Delete(column_family, key);
    num_deletes_++;
  }

  return s;
}

Status TransactionBaseImpl::PutUntracked(ColumnFamilyHandle* column_family,
                                         const Slice& key, const Slice& value) {
  bool untracked = true;
  Status s = TryLock(column_family, key, untracked);

  if (s.ok()) {
    write_batch_->Put(column_family, key, value);
    num_puts_++;
  }

  return s;
}

Status TransactionBaseImpl::PutUntracked(ColumnFamilyHandle* column_family,
                                         const SliceParts& key,
                                         const SliceParts& value) {
  bool untracked = true;
  Status s = TryLock(column_family, key, untracked);

  if (s.ok()) {
    write_batch_->Put(column_family, key, value);
    num_puts_++;
  }

  return s;
}

Status TransactionBaseImpl::MergeUntracked(ColumnFamilyHandle* column_family,
                                           const Slice& key,
                                           const Slice& value) {
  bool untracked = true;
  Status s = TryLock(column_family, key, untracked);

  if (s.ok()) {
    write_batch_->Merge(column_family, key, value);
    num_merges_++;
  }

  return s;
}

Status TransactionBaseImpl::DeleteUntracked(ColumnFamilyHandle* column_family,
                                            const Slice& key) {
  bool untracked = true;
  Status s = TryLock(column_family, key, untracked);

  if (s.ok()) {
    write_batch_->Delete(column_family, key);
    num_deletes_++;
  }

  return s;
}

Status TransactionBaseImpl::DeleteUntracked(ColumnFamilyHandle* column_family,
                                            const SliceParts& key) {
  bool untracked = true;
  Status s = TryLock(column_family, key, untracked);

  if (s.ok()) {
    write_batch_->Delete(column_family, key);
    num_deletes_++;
  }

  return s;
}

void TransactionBaseImpl::PutLogData(const Slice& blob) {
  write_batch_->PutLogData(blob);
}

WriteBatchWithIndex* TransactionBaseImpl::GetWriteBatch() {
  return write_batch_.get();
}

uint64_t TransactionBaseImpl::GetElapsedTime() const {
  return (db_->GetEnv()->NowMicros() - start_time_) / 1000;
}

uint64_t TransactionBaseImpl::GetNumPuts() const { return num_puts_; }

uint64_t TransactionBaseImpl::GetNumDeletes() const { return num_deletes_; }

uint64_t TransactionBaseImpl::GetNumMerges() const { return num_merges_; }

uint64_t TransactionBaseImpl::GetNumKeys() const {
  uint64_t count = 0;

  // sum up locked keys in all column families
  for (const auto& key_map_iter : tracked_keys_) {
    const auto& keys = key_map_iter.second;
    count += keys.size();
  }

  return count;
}

void TransactionBaseImpl::TrackKey(uint32_t cfh_id, const std::string& key,
                                   SequenceNumber seq) {
  auto iter = tracked_keys_[cfh_id].find(key);
  if (iter == tracked_keys_[cfh_id].end()) {
    tracked_keys_[cfh_id].insert({key, seq});

    if (save_points_ != nullptr && !save_points_->empty()) {
      // Aren't tracking this key, add it.
      save_points_->top().new_keys_[cfh_id][key] = seq;
    }
  } else if (seq < iter->second) {
    // Now tracking this key with an earlier sequence number
    iter->second = seq;
  }
}

const TransactionKeyMap* TransactionBaseImpl::GetTrackedKeysSinceSavePoint() {
  if (save_points_ != nullptr && !save_points_->empty()) {
    return &save_points_->top().new_keys_;
  }

  return nullptr;
}

}  // namespace rocksdb

#endif  // ROCKSDB_LITE
