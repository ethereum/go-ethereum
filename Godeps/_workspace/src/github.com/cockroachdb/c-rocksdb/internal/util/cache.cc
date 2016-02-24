//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include <assert.h>
#include <stdio.h>
#include <stdlib.h>

#include "rocksdb/cache.h"
#include "port/port.h"
#include "util/autovector.h"
#include "util/hash.h"
#include "util/mutexlock.h"

namespace rocksdb {

Cache::~Cache() {
}

namespace {

// LRU cache implementation

// An entry is a variable length heap-allocated structure.
// Entries are referenced by cache and/or by any external entity.
// The cache keeps all its entries in table. Some elements
// are also stored on LRU list.
//
// LRUHandle can be in these states:
// 1. Referenced externally AND in hash table.
//  In that case the entry is *not* in the LRU. (refs > 1 && in_cache == true)
// 2. Not referenced externally and in hash table. In that case the entry is
// in the LRU and can be freed. (refs == 1 && in_cache == true)
// 3. Referenced externally and not in hash table. In that case the entry is
// in not on LRU and not in table. (refs >= 1 && in_cache == false)
//
// All newly created LRUHandles are in state 1. If you call LRUCache::Release
// on entry in state 1, it will go into state 2. To move from state 1 to
// state 3, either call LRUCache::Erase or LRUCache::Insert with the same key.
// To move from state 2 to state 1, use LRUCache::Lookup.
// Before destruction, make sure that no handles are in state 1. This means
// that any successful LRUCache::Lookup/LRUCache::Insert have a matching
// RUCache::Release (to move into state 2) or LRUCache::Erase (for state 3)

struct LRUHandle {
  void* value;
  void (*deleter)(const Slice&, void* value);
  LRUHandle* next_hash;
  LRUHandle* next;
  LRUHandle* prev;
  size_t charge;      // TODO(opt): Only allow uint32_t?
  size_t key_length;
  uint32_t refs;      // a number of refs to this entry
                      // cache itself is counted as 1
  bool in_cache;      // true, if this entry is referenced by the hash table
  uint32_t hash;      // Hash of key(); used for fast sharding and comparisons
  char key_data[1];   // Beginning of key

  Slice key() const {
    // For cheaper lookups, we allow a temporary Handle object
    // to store a pointer to a key in "value".
    if (next == this) {
      return *(reinterpret_cast<Slice*>(value));
    } else {
      return Slice(key_data, key_length);
    }
  }

  void Free() {
    assert((refs == 1 && in_cache) || (refs == 0 && !in_cache));
    (*deleter)(key(), value);
    free(this);
  }
};

// We provide our own simple hash table since it removes a whole bunch
// of porting hacks and is also faster than some of the built-in hash
// table implementations in some of the compiler/runtime combinations
// we have tested.  E.g., readrandom speeds up by ~5% over the g++
// 4.4.3's builtin hashtable.
class HandleTable {
 public:
  HandleTable() : length_(0), elems_(0), list_(nullptr) { Resize(); }

  template <typename T>
  void ApplyToAllCacheEntries(T func) {
    for (uint32_t i = 0; i < length_; i++) {
      LRUHandle* h = list_[i];
      while (h != nullptr) {
        auto n = h->next_hash;
        assert(h->in_cache);
        func(h);
        h = n;
      }
    }
  }

  ~HandleTable() {
    ApplyToAllCacheEntries([](LRUHandle* h) {
      if (h->refs == 1) {
        h->Free();
      }
    });
    delete[] list_;
  }

  LRUHandle* Lookup(const Slice& key, uint32_t hash) {
    return *FindPointer(key, hash);
  }

  LRUHandle* Insert(LRUHandle* h) {
    LRUHandle** ptr = FindPointer(h->key(), h->hash);
    LRUHandle* old = *ptr;
    h->next_hash = (old == nullptr ? nullptr : old->next_hash);
    *ptr = h;
    if (old == nullptr) {
      ++elems_;
      if (elems_ > length_) {
        // Since each cache entry is fairly large, we aim for a small
        // average linked list length (<= 1).
        Resize();
      }
    }
    return old;
  }

  LRUHandle* Remove(const Slice& key, uint32_t hash) {
    LRUHandle** ptr = FindPointer(key, hash);
    LRUHandle* result = *ptr;
    if (result != nullptr) {
      *ptr = result->next_hash;
      --elems_;
    }
    return result;
  }

 private:
  // The table consists of an array of buckets where each bucket is
  // a linked list of cache entries that hash into the bucket.
  uint32_t length_;
  uint32_t elems_;
  LRUHandle** list_;

  // Return a pointer to slot that points to a cache entry that
  // matches key/hash.  If there is no such cache entry, return a
  // pointer to the trailing slot in the corresponding linked list.
  LRUHandle** FindPointer(const Slice& key, uint32_t hash) {
    LRUHandle** ptr = &list_[hash & (length_ - 1)];
    while (*ptr != nullptr &&
           ((*ptr)->hash != hash || key != (*ptr)->key())) {
      ptr = &(*ptr)->next_hash;
    }
    return ptr;
  }

  void Resize() {
    uint32_t new_length = 16;
    while (new_length < elems_ * 1.5) {
      new_length *= 2;
    }
    LRUHandle** new_list = new LRUHandle*[new_length];
    memset(new_list, 0, sizeof(new_list[0]) * new_length);
    uint32_t count = 0;
    for (uint32_t i = 0; i < length_; i++) {
      LRUHandle* h = list_[i];
      while (h != nullptr) {
        LRUHandle* next = h->next_hash;
        uint32_t hash = h->hash;
        LRUHandle** ptr = &new_list[hash & (new_length - 1)];
        h->next_hash = *ptr;
        *ptr = h;
        h = next;
        count++;
      }
    }
    assert(elems_ == count);
    delete[] list_;
    list_ = new_list;
    length_ = new_length;
  }
};

// A single shard of sharded cache.
class LRUCache {
 public:
  LRUCache();
  ~LRUCache();

  // Separate from constructor so caller can easily make an array of LRUCache
  // if current usage is more than new capacity, the function will attempt to
  // free the needed space
  void SetCapacity(size_t capacity);

  // Like Cache methods, but with an extra "hash" parameter.
  Cache::Handle* Insert(const Slice& key, uint32_t hash,
                        void* value, size_t charge,
                        void (*deleter)(const Slice& key, void* value));
  Cache::Handle* Lookup(const Slice& key, uint32_t hash);
  void Release(Cache::Handle* handle);
  void Erase(const Slice& key, uint32_t hash);

  // Although in some platforms the update of size_t is atomic, to make sure
  // GetUsage() and GetPinnedUsage() work correctly under any platform, we'll
  // protect them with mutex_.

  size_t GetUsage() const {
    MutexLock l(&mutex_);
    return usage_;
  }

  size_t GetPinnedUsage() const {
    MutexLock l(&mutex_);
    assert(usage_ >= lru_usage_);
    return usage_ - lru_usage_;
  }

  void ApplyToAllCacheEntries(void (*callback)(void*, size_t),
                              bool thread_safe);

 private:
  void LRU_Remove(LRUHandle* e);
  void LRU_Append(LRUHandle* e);
  // Just reduce the reference count by 1.
  // Return true if last reference
  bool Unref(LRUHandle* e);

  // Free some space following strict LRU policy until enough space
  // to hold (usage_ + charge) is freed or the lru list is empty
  // This function is not thread safe - it needs to be executed while
  // holding the mutex_
  void EvictFromLRU(size_t charge,
                    autovector<LRUHandle*>* deleted);

  // Initialized before use.
  size_t capacity_;

  // Memory size for entries residing in the cache
  size_t usage_;

  // Memory size for entries residing only in the LRU list
  size_t lru_usage_;

  // mutex_ protects the following state.
  // We don't count mutex_ as the cache's internal state so semantically we
  // don't mind mutex_ invoking the non-const actions.
  mutable port::Mutex mutex_;

  // Dummy head of LRU list.
  // lru.prev is newest entry, lru.next is oldest entry.
  // LRU contains items which can be evicted, ie reference only by cache
  LRUHandle lru_;

  HandleTable table_;
};

LRUCache::LRUCache() : usage_(0), lru_usage_(0) {
  // Make empty circular linked list
  lru_.next = &lru_;
  lru_.prev = &lru_;
}

LRUCache::~LRUCache() {}

bool LRUCache::Unref(LRUHandle* e) {
  assert(e->refs > 0);
  e->refs--;
  return e->refs == 0;
}

// Call deleter and free

void LRUCache::ApplyToAllCacheEntries(void (*callback)(void*, size_t),
                                      bool thread_safe) {
  if (thread_safe) {
    mutex_.Lock();
  }
  table_.ApplyToAllCacheEntries([callback](LRUHandle* h) {
    callback(h->value, h->charge);
  });
  if (thread_safe) {
    mutex_.Unlock();
  }
}

void LRUCache::LRU_Remove(LRUHandle* e) {
  assert(e->next != nullptr);
  assert(e->prev != nullptr);
  e->next->prev = e->prev;
  e->prev->next = e->next;
  e->prev = e->next = nullptr;
  lru_usage_ -= e->charge;
}

void LRUCache::LRU_Append(LRUHandle* e) {
  // Make "e" newest entry by inserting just before lru_
  assert(e->next == nullptr);
  assert(e->prev == nullptr);
  e->next = &lru_;
  e->prev = lru_.prev;
  e->prev->next = e;
  e->next->prev = e;
  lru_usage_ += e->charge;
}

void LRUCache::EvictFromLRU(size_t charge,
                            autovector<LRUHandle*>* deleted) {
  while (usage_ + charge > capacity_ && lru_.next != &lru_) {
    LRUHandle* old = lru_.next;
    assert(old->in_cache);
    assert(old->refs == 1);  // LRU list contains elements which may be evicted
    LRU_Remove(old);
    table_.Remove(old->key(), old->hash);
    old->in_cache = false;
    Unref(old);
    usage_ -= old->charge;
    deleted->push_back(old);
  }
}

void LRUCache::SetCapacity(size_t capacity) {
  autovector<LRUHandle*> last_reference_list;
  {
    MutexLock l(&mutex_);
    capacity_ = capacity;
    EvictFromLRU(0, &last_reference_list);
  }
  // we free the entries here outside of mutex for
  // performance reasons
  for (auto entry : last_reference_list) {
    entry->Free();
  }
}

Cache::Handle* LRUCache::Lookup(const Slice& key, uint32_t hash) {
  MutexLock l(&mutex_);
  LRUHandle* e = table_.Lookup(key, hash);
  if (e != nullptr) {
    assert(e->in_cache);
    if (e->refs == 1) {
      LRU_Remove(e);
    }
    e->refs++;
  }
  return reinterpret_cast<Cache::Handle*>(e);
}

void LRUCache::Release(Cache::Handle* handle) {
  LRUHandle* e = reinterpret_cast<LRUHandle*>(handle);
  bool last_reference = false;
  {
    MutexLock l(&mutex_);
    last_reference = Unref(e);
    if (last_reference) {
      usage_ -= e->charge;
    }
    if (e->refs == 1 && e->in_cache) {
      // The item is still in cache, and nobody else holds a reference to it
      if (usage_ > capacity_) {
        // the cache is full
        // The LRU list must be empty since the cache is full
        assert(lru_.next == &lru_);
        // take this opportunity and remove the item
        table_.Remove(e->key(), e->hash);
        e->in_cache = false;
        Unref(e);
        usage_ -= e->charge;
        last_reference = true;
      } else {
        // put the item on the list to be potentially freed
        LRU_Append(e);
      }
    }
  }

  // free outside of mutex
  if (last_reference) {
    e->Free();
  }
}

Cache::Handle* LRUCache::Insert(
    const Slice& key, uint32_t hash, void* value, size_t charge,
    void (*deleter)(const Slice& key, void* value)) {

  // Allocate the memory here outside of the mutex
  // If the cache is full, we'll have to release it
  // It shouldn't happen very often though.
  LRUHandle* e =
      reinterpret_cast<LRUHandle*>(malloc(sizeof(LRUHandle) - 1 + key.size()));
  autovector<LRUHandle*> last_reference_list;

  e->value = value;
  e->deleter = deleter;
  e->charge = charge;
  e->key_length = key.size();
  e->hash = hash;
  e->refs = 2;  // One from LRUCache, one for the returned handle
  e->next = e->prev = nullptr;
  e->in_cache = true;
  memcpy(e->key_data, key.data(), key.size());

  {
    MutexLock l(&mutex_);

    // Free the space following strict LRU policy until enough space
    // is freed or the lru list is empty
    EvictFromLRU(charge, &last_reference_list);

    // insert into the cache
    // note that the cache might get larger than its capacity if not enough
    // space was freed
    LRUHandle* old = table_.Insert(e);
    usage_ += e->charge;
    if (old != nullptr) {
      old->in_cache = false;
      if (Unref(old)) {
        usage_ -= old->charge;
        // old is on LRU because it's in cache and its reference count
        // was just 1 (Unref returned 0)
        LRU_Remove(old);
        last_reference_list.push_back(old);
      }
    }
  }

  // we free the entries here outside of mutex for
  // performance reasons
  for (auto entry : last_reference_list) {
    entry->Free();
  }

  return reinterpret_cast<Cache::Handle*>(e);
}

void LRUCache::Erase(const Slice& key, uint32_t hash) {
  LRUHandle* e;
  bool last_reference = false;
  {
    MutexLock l(&mutex_);
    e = table_.Remove(key, hash);
    if (e != nullptr) {
      last_reference = Unref(e);
      if (last_reference) {
        usage_ -= e->charge;
      }
      if (last_reference && e->in_cache) {
        LRU_Remove(e);
      }
      e->in_cache = false;
    }
  }

  // mutex not held here
  // last_reference will only be true if e != nullptr
  if (last_reference) {
    e->Free();
  }
}

static int kNumShardBits = 4;          // default values, can be overridden

class ShardedLRUCache : public Cache {
 private:
  LRUCache* shards_;
  port::Mutex id_mutex_;
  port::Mutex capacity_mutex_;
  uint64_t last_id_;
  int num_shard_bits_;
  size_t capacity_;

  static inline uint32_t HashSlice(const Slice& s) {
    return Hash(s.data(), s.size(), 0);
  }

  uint32_t Shard(uint32_t hash) {
    // Note, hash >> 32 yields hash in gcc, not the zero we expect!
    return (num_shard_bits_ > 0) ? (hash >> (32 - num_shard_bits_)) : 0;
  }

 public:
  ShardedLRUCache(size_t capacity, int num_shard_bits)
      : last_id_(0), num_shard_bits_(num_shard_bits), capacity_(capacity) {
    int num_shards = 1 << num_shard_bits_;
    shards_ = new LRUCache[num_shards];
    const size_t per_shard = (capacity + (num_shards - 1)) / num_shards;
    for (int s = 0; s < num_shards; s++) {
      shards_[s].SetCapacity(per_shard);
    }
  }
  virtual ~ShardedLRUCache() {
    delete[] shards_;
  }
  virtual void SetCapacity(size_t capacity) override {
    int num_shards = 1 << num_shard_bits_;
    const size_t per_shard = (capacity + (num_shards - 1)) / num_shards;
    MutexLock l(&capacity_mutex_);
    for (int s = 0; s < num_shards; s++) {
      shards_[s].SetCapacity(per_shard);
    }
    capacity_ = capacity;
  }
  virtual Handle* Insert(const Slice& key, void* value, size_t charge,
                         void (*deleter)(const Slice& key,
                                         void* value)) override {
    const uint32_t hash = HashSlice(key);
    return shards_[Shard(hash)].Insert(key, hash, value, charge, deleter);
  }
  virtual Handle* Lookup(const Slice& key) override {
    const uint32_t hash = HashSlice(key);
    return shards_[Shard(hash)].Lookup(key, hash);
  }
  virtual void Release(Handle* handle) override {
    LRUHandle* h = reinterpret_cast<LRUHandle*>(handle);
    shards_[Shard(h->hash)].Release(handle);
  }
  virtual void Erase(const Slice& key) override {
    const uint32_t hash = HashSlice(key);
    shards_[Shard(hash)].Erase(key, hash);
  }
  virtual void* Value(Handle* handle) override {
    return reinterpret_cast<LRUHandle*>(handle)->value;
  }
  virtual uint64_t NewId() override {
    MutexLock l(&id_mutex_);
    return ++(last_id_);
  }
  virtual size_t GetCapacity() const override { return capacity_; }

  virtual size_t GetUsage() const override {
    // We will not lock the cache when getting the usage from shards.
    int num_shards = 1 << num_shard_bits_;
    size_t usage = 0;
    for (int s = 0; s < num_shards; s++) {
      usage += shards_[s].GetUsage();
    }
    return usage;
  }
  virtual size_t GetPinnedUsage() const override {
    // We will not lock the cache when getting the usage from shards.
    int num_shards = 1 << num_shard_bits_;
    size_t usage = 0;
    for (int s = 0; s < num_shards; s++) {
      usage += shards_[s].GetPinnedUsage();
    }
    return usage;
  }

  virtual void DisownData() override { shards_ = nullptr; }

  virtual void ApplyToAllCacheEntries(void (*callback)(void*, size_t),
                                      bool thread_safe) override {
    int num_shards = 1 << num_shard_bits_;
    for (int s = 0; s < num_shards; s++) {
      shards_[s].ApplyToAllCacheEntries(callback, thread_safe);
    }
  }
};

}  // end anonymous namespace

shared_ptr<Cache> NewLRUCache(size_t capacity) {
  return NewLRUCache(capacity, kNumShardBits);
}

shared_ptr<Cache> NewLRUCache(size_t capacity, int num_shard_bits) {
  if (num_shard_bits >= 20) {
    return nullptr;  // the cache cannot be sharded into too many fine pieces
  }
  return std::make_shared<ShardedLRUCache>(capacity, num_shard_bits);
}

}  // namespace rocksdb
