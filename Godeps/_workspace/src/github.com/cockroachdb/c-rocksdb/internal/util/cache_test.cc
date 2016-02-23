//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "rocksdb/cache.h"

#include <forward_list>
#include <vector>
#include <string>
#include <iostream>
#include "util/coding.h"
#include "util/string_util.h"
#include "util/testharness.h"

namespace rocksdb {

// Conversions between numeric keys/values and the types expected by Cache.
static std::string EncodeKey(int k) {
  std::string result;
  PutFixed32(&result, k);
  return result;
}
static int DecodeKey(const Slice& k) {
  assert(k.size() == 4);
  return DecodeFixed32(k.data());
}
static void* EncodeValue(uintptr_t v) { return reinterpret_cast<void*>(v); }
static int DecodeValue(void* v) {
  return static_cast<int>(reinterpret_cast<uintptr_t>(v));
}

class CacheTest : public testing::Test {
 public:
  static CacheTest* current_;

  static void Deleter(const Slice& key, void* v) {
    current_->deleted_keys_.push_back(DecodeKey(key));
    current_->deleted_values_.push_back(DecodeValue(v));
  }

  static const int kCacheSize = 1000;
  static const int kNumShardBits = 4;

  static const int kCacheSize2 = 100;
  static const int kNumShardBits2 = 2;

  std::vector<int> deleted_keys_;
  std::vector<int> deleted_values_;
  shared_ptr<Cache> cache_;
  shared_ptr<Cache> cache2_;

  CacheTest() :
      cache_(NewLRUCache(kCacheSize, kNumShardBits)),
      cache2_(NewLRUCache(kCacheSize2, kNumShardBits2)) {
    current_ = this;
  }

  ~CacheTest() {
  }

  int Lookup(shared_ptr<Cache> cache, int key) {
    Cache::Handle* handle = cache->Lookup(EncodeKey(key));
    const int r = (handle == nullptr) ? -1 : DecodeValue(cache->Value(handle));
    if (handle != nullptr) {
      cache->Release(handle);
    }
    return r;
  }

  void Insert(shared_ptr<Cache> cache, int key, int value, int charge = 1) {
    cache->Release(cache->Insert(EncodeKey(key), EncodeValue(value), charge,
                                  &CacheTest::Deleter));
  }

  void Erase(shared_ptr<Cache> cache, int key) {
    cache->Erase(EncodeKey(key));
  }


  int Lookup(int key) {
    return Lookup(cache_, key);
  }

  void Insert(int key, int value, int charge = 1) {
    Insert(cache_, key, value, charge);
  }

  void Erase(int key) {
    Erase(cache_, key);
  }

  int Lookup2(int key) {
    return Lookup(cache2_, key);
  }

  void Insert2(int key, int value, int charge = 1) {
    Insert(cache2_, key, value, charge);
  }

  void Erase2(int key) {
    Erase(cache2_, key);
  }
};
CacheTest* CacheTest::current_;

namespace {
void dumbDeleter(const Slice& key, void* value) { }
}  // namespace

TEST_F(CacheTest, UsageTest) {
  // cache is shared_ptr and will be automatically cleaned up.
  const uint64_t kCapacity = 100000;
  auto cache = NewLRUCache(kCapacity, 8);

  size_t usage = 0;
  const char* value = "abcdef";
  // make sure everything will be cached
  for (int i = 1; i < 100; ++i) {
    std::string key(i, 'a');
    auto kv_size = key.size() + 5;
    cache->Release(
        cache->Insert(key, (void*)value, kv_size, dumbDeleter)
    );
    usage += kv_size;
    ASSERT_EQ(usage, cache->GetUsage());
  }

  // make sure the cache will be overloaded
  for (uint64_t i = 1; i < kCapacity; ++i) {
    auto key = ToString(i);
    cache->Release(
        cache->Insert(key, (void*)value, key.size() + 5, dumbDeleter)
    );
  }

  // the usage should be close to the capacity
  ASSERT_GT(kCapacity, cache->GetUsage());
  ASSERT_LT(kCapacity * 0.95, cache->GetUsage());
}

TEST_F(CacheTest, PinnedUsageTest) {
  // cache is shared_ptr and will be automatically cleaned up.
  const uint64_t kCapacity = 100000;
  auto cache = NewLRUCache(kCapacity, 8);

  size_t pinned_usage = 0;
  const char* value = "abcdef";

  std::forward_list<Cache::Handle*> unreleased_handles;

  // Add entries. Unpin some of them after insertion. Then, pin some of them
  // again. Check GetPinnedUsage().
  for (int i = 1; i < 100; ++i) {
    std::string key(i, 'a');
    auto kv_size = key.size() + 5;
    auto handle = cache->Insert(key, (void*)value, kv_size, dumbDeleter);
    pinned_usage += kv_size;
    ASSERT_EQ(pinned_usage, cache->GetPinnedUsage());
    if (i % 2 == 0) {
      cache->Release(handle);
      pinned_usage -= kv_size;
      ASSERT_EQ(pinned_usage, cache->GetPinnedUsage());
    } else {
      unreleased_handles.push_front(handle);
    }
    if (i % 3 == 0) {
      unreleased_handles.push_front(cache->Lookup(key));
      // If i % 2 == 0, then the entry was unpinned before Lookup, so pinned
      // usage increased
      if (i % 2 == 0) {
        pinned_usage += kv_size;
      }
      ASSERT_EQ(pinned_usage, cache->GetPinnedUsage());
    }
  }

  // check that overloading the cache does not change the pinned usage
  for (uint64_t i = 1; i < 2 * kCapacity; ++i) {
    auto key = ToString(i);
    cache->Release(
        cache->Insert(key, (void*)value, key.size() + 5, dumbDeleter));
  }
  ASSERT_EQ(pinned_usage, cache->GetPinnedUsage());

  // release handles for pinned entries to prevent memory leaks
  for (auto handle : unreleased_handles) {
    cache->Release(handle);
  }
}

TEST_F(CacheTest, HitAndMiss) {
  ASSERT_EQ(-1, Lookup(100));

  Insert(100, 101);
  ASSERT_EQ(101, Lookup(100));
  ASSERT_EQ(-1,  Lookup(200));
  ASSERT_EQ(-1,  Lookup(300));

  Insert(200, 201);
  ASSERT_EQ(101, Lookup(100));
  ASSERT_EQ(201, Lookup(200));
  ASSERT_EQ(-1,  Lookup(300));

  Insert(100, 102);
  ASSERT_EQ(102, Lookup(100));
  ASSERT_EQ(201, Lookup(200));
  ASSERT_EQ(-1,  Lookup(300));

  ASSERT_EQ(1U, deleted_keys_.size());
  ASSERT_EQ(100, deleted_keys_[0]);
  ASSERT_EQ(101, deleted_values_[0]);
}

TEST_F(CacheTest, Erase) {
  Erase(200);
  ASSERT_EQ(0U, deleted_keys_.size());

  Insert(100, 101);
  Insert(200, 201);
  Erase(100);
  ASSERT_EQ(-1,  Lookup(100));
  ASSERT_EQ(201, Lookup(200));
  ASSERT_EQ(1U, deleted_keys_.size());
  ASSERT_EQ(100, deleted_keys_[0]);
  ASSERT_EQ(101, deleted_values_[0]);

  Erase(100);
  ASSERT_EQ(-1,  Lookup(100));
  ASSERT_EQ(201, Lookup(200));
  ASSERT_EQ(1U, deleted_keys_.size());
}

TEST_F(CacheTest, EntriesArePinned) {
  Insert(100, 101);
  Cache::Handle* h1 = cache_->Lookup(EncodeKey(100));
  ASSERT_EQ(101, DecodeValue(cache_->Value(h1)));
  ASSERT_EQ(1U, cache_->GetUsage());

  Insert(100, 102);
  Cache::Handle* h2 = cache_->Lookup(EncodeKey(100));
  ASSERT_EQ(102, DecodeValue(cache_->Value(h2)));
  ASSERT_EQ(0U, deleted_keys_.size());
  ASSERT_EQ(2U, cache_->GetUsage());

  cache_->Release(h1);
  ASSERT_EQ(1U, deleted_keys_.size());
  ASSERT_EQ(100, deleted_keys_[0]);
  ASSERT_EQ(101, deleted_values_[0]);
  ASSERT_EQ(1U, cache_->GetUsage());

  Erase(100);
  ASSERT_EQ(-1, Lookup(100));
  ASSERT_EQ(1U, deleted_keys_.size());
  ASSERT_EQ(1U, cache_->GetUsage());

  cache_->Release(h2);
  ASSERT_EQ(2U, deleted_keys_.size());
  ASSERT_EQ(100, deleted_keys_[1]);
  ASSERT_EQ(102, deleted_values_[1]);
  ASSERT_EQ(0U, cache_->GetUsage());
}

TEST_F(CacheTest, EvictionPolicy) {
  Insert(100, 101);
  Insert(200, 201);

  // Frequently used entry must be kept around
  for (int i = 0; i < kCacheSize + 100; i++) {
    Insert(1000+i, 2000+i);
    ASSERT_EQ(2000+i, Lookup(1000+i));
    ASSERT_EQ(101, Lookup(100));
  }
  ASSERT_EQ(101, Lookup(100));
  ASSERT_EQ(-1, Lookup(200));
}

TEST_F(CacheTest, EvictionPolicyRef) {
  Insert(100, 101);
  Insert(101, 102);
  Insert(102, 103);
  Insert(103, 104);
  Insert(200, 101);
  Insert(201, 102);
  Insert(202, 103);
  Insert(203, 104);
  Cache::Handle* h201 = cache_->Lookup(EncodeKey(200));
  Cache::Handle* h202 = cache_->Lookup(EncodeKey(201));
  Cache::Handle* h203 = cache_->Lookup(EncodeKey(202));
  Cache::Handle* h204 = cache_->Lookup(EncodeKey(203));
  Insert(300, 101);
  Insert(301, 102);
  Insert(302, 103);
  Insert(303, 104);

  // Insert entries much more than Cache capacity
  for (int i = 0; i < kCacheSize + 100; i++) {
    Insert(1000 + i, 2000 + i);
  }

  // Check whether the entries inserted in the beginning
  // are evicted. Ones without extra ref are evicted and
  // those with are not.
  ASSERT_EQ(-1, Lookup(100));
  ASSERT_EQ(-1, Lookup(101));
  ASSERT_EQ(-1, Lookup(102));
  ASSERT_EQ(-1, Lookup(103));

  ASSERT_EQ(-1, Lookup(300));
  ASSERT_EQ(-1, Lookup(301));
  ASSERT_EQ(-1, Lookup(302));
  ASSERT_EQ(-1, Lookup(303));

  ASSERT_EQ(101, Lookup(200));
  ASSERT_EQ(102, Lookup(201));
  ASSERT_EQ(103, Lookup(202));
  ASSERT_EQ(104, Lookup(203));

  // Cleaning up all the handles
  cache_->Release(h201);
  cache_->Release(h202);
  cache_->Release(h203);
  cache_->Release(h204);
}

TEST_F(CacheTest, ErasedHandleState) {
  // insert a key and get two handles
  Insert(100, 1000);
  Cache::Handle* h1 = cache_->Lookup(EncodeKey(100));
  Cache::Handle* h2 = cache_->Lookup(EncodeKey(100));
  ASSERT_EQ(h1, h2);
  ASSERT_EQ(DecodeValue(cache_->Value(h1)), 1000);
  ASSERT_EQ(DecodeValue(cache_->Value(h2)), 1000);

  // delete the key from the cache
  Erase(100);
  // can no longer find in the cache
  ASSERT_EQ(-1, Lookup(100));

  // release one handle
  cache_->Release(h1);
  // still can't find in cache
  ASSERT_EQ(-1, Lookup(100));

  cache_->Release(h2);
}

TEST_F(CacheTest, HeavyEntries) {
  // Add a bunch of light and heavy entries and then count the combined
  // size of items still in the cache, which must be approximately the
  // same as the total capacity.
  const int kLight = 1;
  const int kHeavy = 10;
  int added = 0;
  int index = 0;
  while (added < 2*kCacheSize) {
    const int weight = (index & 1) ? kLight : kHeavy;
    Insert(index, 1000+index, weight);
    added += weight;
    index++;
  }

  int cached_weight = 0;
  for (int i = 0; i < index; i++) {
    const int weight = (i & 1 ? kLight : kHeavy);
    int r = Lookup(i);
    if (r >= 0) {
      cached_weight += weight;
      ASSERT_EQ(1000+i, r);
    }
  }
  ASSERT_LE(cached_weight, kCacheSize + kCacheSize/10);
}

TEST_F(CacheTest, NewId) {
  uint64_t a = cache_->NewId();
  uint64_t b = cache_->NewId();
  ASSERT_NE(a, b);
}


class Value {
 private:
  size_t v_;
 public:
  explicit Value(size_t v) : v_(v) { }

  ~Value() { std::cout << v_ << " is destructed\n"; }
};

namespace {
void deleter(const Slice& key, void* value) {
  delete static_cast<Value *>(value);
}
}  // namespace

TEST_F(CacheTest, SetCapacity) {
  // test1: increase capacity
  // lets create a cache with capacity 5,
  // then, insert 5 elements, then increase capacity
  // to 10, returned capacity should be 10, usage=5
  std::shared_ptr<Cache> cache = NewLRUCache(5, 0);
  std::vector<Cache::Handle*> handles(10);
  // Insert 5 entries, but not releasing.
  for (size_t i = 0; i < 5; i++) {
    std::string key = ToString(i+1);
    handles[i] = cache->Insert(key, new Value(i+1), 1, &deleter);
  }
  ASSERT_EQ(5U, cache->GetCapacity());
  ASSERT_EQ(5U, cache->GetUsage());
  cache->SetCapacity(10);
  ASSERT_EQ(10U, cache->GetCapacity());
  ASSERT_EQ(5U, cache->GetUsage());

  // test2: decrease capacity
  // insert 5 more elements to cache, then release 5,
  // then decrease capacity to 7, final capacity should be 7
  // and usage should be 7
  for (size_t i = 5; i < 10; i++) {
    std::string key = ToString(i+1);
    handles[i] = cache->Insert(key, new Value(i+1), 1, &deleter);
  }
  ASSERT_EQ(10U, cache->GetCapacity());
  ASSERT_EQ(10U, cache->GetUsage());
  for (size_t i = 0; i < 5; i++) {
    cache->Release(handles[i]);
  }
  ASSERT_EQ(10U, cache->GetCapacity());
  ASSERT_EQ(10U, cache->GetUsage());
  cache->SetCapacity(7);
  ASSERT_EQ(7, cache->GetCapacity());
  ASSERT_EQ(7, cache->GetUsage());

  // release remaining 5 to keep valgrind happy
  for (size_t i = 5; i < 10; i++) {
    cache->Release(handles[i]);
  }
}

TEST_F(CacheTest, OverCapacity) {
  size_t n = 10;

  // a LRUCache with n entries and one shard only
  std::shared_ptr<Cache> cache = NewLRUCache(n, 0);

  std::vector<Cache::Handle*> handles(n+1);

  // Insert n+1 entries, but not releasing.
  for (size_t i = 0; i < n + 1; i++) {
    std::string key = ToString(i+1);
    handles[i] = cache->Insert(key, new Value(i+1), 1, &deleter);
  }

  // Guess what's in the cache now?
  for (size_t i = 0; i < n + 1; i++) {
    std::string key = ToString(i+1);
    auto h = cache->Lookup(key);
    std::cout << key << (h?" found\n":" not found\n");
    ASSERT_TRUE(h != nullptr);
    if (h) cache->Release(h);
  }

  // the cache is over capacity since nothing could be evicted
  ASSERT_EQ(n + 1U, cache->GetUsage());
  for (size_t i = 0; i < n + 1; i++) {
    cache->Release(handles[i]);
  }

  // cache is under capacity now since elements were released
  ASSERT_EQ(n, cache->GetUsage());

  // element 0 is evicted and the rest is there
  // This is consistent with the LRU policy since the element 0
  // was released first
  for (size_t i = 0; i < n + 1; i++) {
    std::string key = ToString(i+1);
    auto h = cache->Lookup(key);
    if (h) {
      ASSERT_NE(i, 0U);
      cache->Release(h);
    } else {
      ASSERT_EQ(i, 0U);
    }
  }
}

namespace {
std::vector<std::pair<int, int>> callback_state;
void callback(void* entry, size_t charge) {
  callback_state.push_back({DecodeValue(entry), static_cast<int>(charge)});
}
};

TEST_F(CacheTest, ApplyToAllCacheEntiresTest) {
  std::vector<std::pair<int, int>> inserted;
  callback_state.clear();

  for (int i = 0; i < 10; ++i) {
    Insert(i, i * 2, i + 1);
    inserted.push_back({i * 2, i + 1});
  }
  cache_->ApplyToAllCacheEntries(callback, true);

  sort(inserted.begin(), inserted.end());
  sort(callback_state.begin(), callback_state.end());
  ASSERT_TRUE(inserted == callback_state);
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
