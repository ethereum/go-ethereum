//  Copyright (c) 2015, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#pragma once
#ifndef ROCKSDB_LITE

#include <chrono>
#include <string>
#include <unordered_map>
#include <vector>

#include "rocksdb/utilities/transaction.h"
#include "util/instrumented_mutex.h"
#include "util/thread_local.h"
#include "utilities/transactions/transaction_impl.h"

namespace rocksdb {

class ColumnFamilyHandle;
struct LockInfo;
struct LockMap;
struct LockMapStripe;

class Slice;

class TransactionLockMgr {
 public:
  TransactionLockMgr(size_t default_num_stripes, int64_t max_num_locks,
                     std::shared_ptr<TransactionDBMutexFactory> factory);

  ~TransactionLockMgr();

  // Creates a new LockMap for this column family.  Caller should guarantee
  // that this column family does not already exist.
  void AddColumnFamily(uint32_t column_family_id);

  // Deletes the LockMap for this column family.  Caller should guarantee that
  // this column family is no longer in use.
  void RemoveColumnFamily(uint32_t column_family_id);

  // Attempt to lock key.  If OK status is returned, the caller is responsible
  // for calling UnLock() on this key.
  Status TryLock(const TransactionImpl* txn, uint32_t column_family_id,
                 const std::string& key, Env* env);

  // Unlock a key locked by TryLock().  txn must be the same Transaction that
  // locked this key.
  void UnLock(const TransactionImpl* txn, const TransactionKeyMap* keys,
              Env* env);
  void UnLock(TransactionImpl* txn, uint32_t column_family_id,
              const std::string& key, Env* env);

 private:
  // Default number of lock map stripes per column family
  const size_t default_num_stripes_;

  // Limit on number of keys locked per column family
  const int64_t max_num_locks_;

  // Used to allocate mutexes/condvars to use when locking keys
  std::shared_ptr<TransactionDBMutexFactory> mutex_factory_;

  // Must be held when accessing/modifying lock_maps_
  InstrumentedMutex lock_map_mutex_;

  // Map of ColumnFamilyId to locked key info
  using LockMaps = std::unordered_map<uint32_t, std::shared_ptr<LockMap>>;
  LockMaps lock_maps_;

  // Thread-local cache of entries in lock_maps_.  This is an optimization
  // to avoid acquiring a mutex in order to look up a LockMap
  std::unique_ptr<ThreadLocalPtr> lock_maps_cache_;

  bool IsLockExpired(const LockInfo& lock_info, Env* env, uint64_t* wait_time);

  std::shared_ptr<LockMap> GetLockMap(uint32_t column_family_id);

  Status AcquireWithTimeout(LockMap* lock_map, LockMapStripe* stripe,
                            const std::string& key, Env* env, int64_t timeout,
                            const LockInfo& lock_info);

  Status AcquireLocked(LockMap* lock_map, LockMapStripe* stripe,
                       const std::string& key, Env* env,
                       const LockInfo& lock_info, uint64_t* wait_time);

  // No copying allowed
  TransactionLockMgr(const TransactionLockMgr&);
  void operator=(const TransactionLockMgr&);
};

}  //  namespace rocksdb
#endif  // ROCKSDB_LITE
