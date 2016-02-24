//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include <limits>
#include <string>
#include <utility>

#include "db/column_family.h"
#include "db/db_impl.h"
#include "db/db_iter.h"
#include "db/dbformat.h"
#include "db/managed_iterator.h"
#include "rocksdb/env.h"
#include "rocksdb/slice.h"
#include "rocksdb/slice_transform.h"
#include "table/merger.h"
#include "util/xfunc.h"

namespace rocksdb {

namespace {
// Helper class that locks a mutex on construction and unlocks the mutex when
// the destructor of the MutexLock object is invoked.
//
// Typical usage:
//
//   void MyClass::MyMethod() {
//     MILock l(&mu_);       // mu_ is an instance variable
//     ... some complex code, possibly with multiple return paths ...
//   }

class MILock {
 public:
  explicit MILock(std::mutex* mu, ManagedIterator* mi) : mu_(mu), mi_(mi) {
    this->mu_->lock();
  }
  ~MILock() {
    this->mu_->unlock();
    XFUNC_TEST("managed_xftest_release", "managed_unlock", managed_unlock1,
               xf_manage_release, mi_);
  }
  ManagedIterator* GetManagedIterator() { return mi_; }

 private:
  std::mutex* const mu_;
  ManagedIterator* mi_;
  // No copying allowed
  MILock(const MILock&) = delete;
  void operator=(const MILock&) = delete;
};
}  // anonymous namespace

//
// Synchronization between modifiers, releasers, creators
// If iterator operation, wait till (!in_use), set in_use, do op, reset in_use
//  if modifying mutable_iter, atomically exchange in_use:
//  return if in_use set / otherwise set in use,
//  atomically replace new iter with old , reset in use
//  The releaser is the new operation and it holds a lock for a very short time
//  The existing non-const iterator operations are supposed to be single
//  threaded and hold the lock for the duration of the operation
//  The existing const iterator operations use the cached key/values
//  and don't do any locking.
ManagedIterator::ManagedIterator(DBImpl* db, const ReadOptions& read_options,
                                 ColumnFamilyData* cfd)
    : db_(db),
      read_options_(read_options),
      cfd_(cfd),
      svnum_(cfd->GetSuperVersionNumber()),
      mutable_iter_(nullptr),
      valid_(false),
      snapshot_created_(false),
      release_supported_(true) {
  read_options_.managed = false;
  if ((!read_options_.tailing) && (read_options_.snapshot == nullptr)) {
    assert(read_options_.snapshot = db_->GetSnapshot());
    snapshot_created_ = true;
  }
  cfh_.SetCFD(cfd);
  mutable_iter_ = unique_ptr<Iterator>(db->NewIterator(read_options_, &cfh_));
  XFUNC_TEST("managed_xftest_dropold", "managed_create", xf_managed_create1,
             xf_manage_create, this);
}

ManagedIterator::~ManagedIterator() {
  Lock();
  if (snapshot_created_) {
    db_->ReleaseSnapshot(read_options_.snapshot);
    snapshot_created_ = false;
    read_options_.snapshot = nullptr;
  }
  UnLock();
}

bool ManagedIterator::Valid() const { return valid_; }

void ManagedIterator::SeekToLast() {
  MILock l(&in_use_, this);
  if (NeedToRebuild()) {
    RebuildIterator();
  }
  assert(mutable_iter_ != nullptr);
  mutable_iter_->SeekToLast();
  if (mutable_iter_->status().ok()) {
    UpdateCurrent();
  }
}

void ManagedIterator::SeekToFirst() {
  MILock l(&in_use_, this);
  SeekInternal(Slice(), true);
}

void ManagedIterator::Seek(const Slice& user_key) {
  MILock l(&in_use_, this);
  SeekInternal(user_key, false);
}

void ManagedIterator::SeekInternal(const Slice& user_key, bool seek_to_first) {
  if (NeedToRebuild()) {
    RebuildIterator();
  }
  assert(mutable_iter_ != nullptr);
  if (seek_to_first) {
    mutable_iter_->SeekToFirst();
  } else {
    mutable_iter_->Seek(user_key);
  }
  UpdateCurrent();
}

void ManagedIterator::Prev() {
  if (!valid_) {
    status_ = Status::InvalidArgument("Iterator value invalid");
    return;
  }
  MILock l(&in_use_, this);
  if (NeedToRebuild()) {
    std::string current_key = key().ToString();
    Slice old_key(current_key);
    RebuildIterator();
    SeekInternal(old_key, false);
    UpdateCurrent();
    if (!valid_) {
      return;
    }
    if (key().compare(old_key) != 0) {
      valid_ = false;
      status_ = Status::Incomplete("Cannot do Prev now");
      return;
    }
  }
  mutable_iter_->Prev();
  if (mutable_iter_->status().ok()) {
    UpdateCurrent();
    status_ = Status::OK();
  } else {
    status_ = mutable_iter_->status();
  }
}

void ManagedIterator::Next() {
  if (!valid_) {
    status_ = Status::InvalidArgument("Iterator value invalid");
    return;
  }
  MILock l(&in_use_, this);
  if (NeedToRebuild()) {
    std::string current_key = key().ToString();
    Slice old_key(current_key.data(), cached_key_.Size());
    RebuildIterator();
    SeekInternal(old_key, false);
    UpdateCurrent();
    if (!valid_) {
      return;
    }
    if (key().compare(old_key) != 0) {
      valid_ = false;
      status_ = Status::Incomplete("Cannot do Next now");
      return;
    }
  }
  mutable_iter_->Next();
  UpdateCurrent();
}

Slice ManagedIterator::key() const {
  assert(valid_);
  return cached_key_.GetKey();
}

Slice ManagedIterator::value() const {
  assert(valid_);
  return cached_value_.GetKey();
}

Status ManagedIterator::status() const { return status_; }

void ManagedIterator::RebuildIterator() {
  svnum_ = cfd_->GetSuperVersionNumber();
  mutable_iter_ = unique_ptr<Iterator>(db_->NewIterator(read_options_, &cfh_));
}

void ManagedIterator::UpdateCurrent() {
  assert(mutable_iter_ != nullptr);

  if (!(valid_ = mutable_iter_->Valid())) {
    status_ = mutable_iter_->status();
    return;
  }

  status_ = Status::OK();
  cached_key_.SetKey(mutable_iter_->key());
  cached_value_.SetKey(mutable_iter_->value());
}

void ManagedIterator::ReleaseIter(bool only_old) {
  if ((mutable_iter_ == nullptr) || (!release_supported_)) {
    return;
  }
  if (svnum_ != cfd_->GetSuperVersionNumber() || !only_old) {
    if (!TryLock()) {  // Don't release iter if in use
      return;
    }
    mutable_iter_ = nullptr;  // in_use for a very short time
    UnLock();
  }
}

bool ManagedIterator::NeedToRebuild() {
  if ((mutable_iter_ == nullptr) || (status_.IsIncomplete()) ||
      (!only_drop_old_ && (svnum_ != cfd_->GetSuperVersionNumber()))) {
    return true;
  }
  return false;
}

void ManagedIterator::Lock() {
  in_use_.lock();
  return;
}

bool ManagedIterator::TryLock() { return in_use_.try_lock(); }

void ManagedIterator::UnLock() {
  in_use_.unlock();
  XFUNC_TEST("managed_xftest_release", "managed_unlock", managed_unlock1,
             xf_manage_release, this);
}

}  // namespace rocksdb

#endif  // ROCKSDB_LITE
