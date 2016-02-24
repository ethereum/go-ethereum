//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "db/memtable.h"

#include <memory>
#include <algorithm>
#include <limits>

#include "db/dbformat.h"
#include "db/merge_context.h"
#include "db/writebuffer.h"
#include "rocksdb/comparator.h"
#include "rocksdb/env.h"
#include "rocksdb/iterator.h"
#include "rocksdb/merge_operator.h"
#include "rocksdb/slice_transform.h"
#include "table/merger.h"
#include "util/arena.h"
#include "util/coding.h"
#include "util/murmurhash.h"
#include "util/mutexlock.h"
#include "util/perf_context_imp.h"
#include "util/statistics.h"
#include "util/stop_watch.h"

namespace rocksdb {

MemTableOptions::MemTableOptions(
    const ImmutableCFOptions& ioptions,
    const MutableCFOptions& mutable_cf_options)
  : write_buffer_size(mutable_cf_options.write_buffer_size),
    arena_block_size(mutable_cf_options.arena_block_size),
    memtable_prefix_bloom_bits(mutable_cf_options.memtable_prefix_bloom_bits),
    memtable_prefix_bloom_probes(
        mutable_cf_options.memtable_prefix_bloom_probes),
    memtable_prefix_bloom_huge_page_tlb_size(
        mutable_cf_options.memtable_prefix_bloom_huge_page_tlb_size),
    inplace_update_support(ioptions.inplace_update_support),
    inplace_update_num_locks(mutable_cf_options.inplace_update_num_locks),
    inplace_callback(ioptions.inplace_callback),
    max_successive_merges(mutable_cf_options.max_successive_merges),
    filter_deletes(mutable_cf_options.filter_deletes),
    statistics(ioptions.statistics),
    merge_operator(ioptions.merge_operator),
    info_log(ioptions.info_log) {}

MemTable::MemTable(const InternalKeyComparator& cmp,
                   const ImmutableCFOptions& ioptions,
                   const MutableCFOptions& mutable_cf_options,
                   WriteBuffer* write_buffer, SequenceNumber earliest_seq)
    : comparator_(cmp),
      moptions_(ioptions, mutable_cf_options),
      refs_(0),
      kArenaBlockSize(OptimizeBlockSize(moptions_.arena_block_size)),
      arena_(moptions_.arena_block_size),
      allocator_(&arena_, write_buffer),
      table_(ioptions.memtable_factory->CreateMemTableRep(
          comparator_, &allocator_, ioptions.prefix_extractor,
          ioptions.info_log)),
      data_size_(0),
      num_entries_(0),
      num_deletes_(0),
      flush_in_progress_(false),
      flush_completed_(false),
      file_number_(0),
      first_seqno_(0),
      earliest_seqno_(earliest_seq),
      mem_next_logfile_number_(0),
      locks_(moptions_.inplace_update_support
                 ? moptions_.inplace_update_num_locks
                 : 0),
      prefix_extractor_(ioptions.prefix_extractor),
      should_flush_(ShouldFlushNow()),
      flush_scheduled_(false),
      env_(ioptions.env) {
  // if should_flush_ == true without an entry inserted, something must have
  // gone wrong already.
  assert(!should_flush_);
  if (prefix_extractor_ && moptions_.memtable_prefix_bloom_bits > 0) {
    prefix_bloom_.reset(new DynamicBloom(
        &allocator_,
        moptions_.memtable_prefix_bloom_bits, ioptions.bloom_locality,
        moptions_.memtable_prefix_bloom_probes, nullptr,
        moptions_.memtable_prefix_bloom_huge_page_tlb_size,
        ioptions.info_log));
  }
}

MemTable::~MemTable() { assert(refs_ == 0); }

size_t MemTable::ApproximateMemoryUsage() {
  size_t arena_usage = arena_.ApproximateMemoryUsage();
  size_t table_usage = table_->ApproximateMemoryUsage();
  // let MAX_USAGE =  std::numeric_limits<size_t>::max()
  // then if arena_usage + total_usage >= MAX_USAGE, return MAX_USAGE.
  // the following variation is to avoid numeric overflow.
  if (arena_usage >= std::numeric_limits<size_t>::max() - table_usage) {
    return std::numeric_limits<size_t>::max();
  }
  // otherwise, return the actual usage
  return arena_usage + table_usage;
}

bool MemTable::ShouldFlushNow() const {
  // In a lot of times, we cannot allocate arena blocks that exactly matches the
  // buffer size. Thus we have to decide if we should over-allocate or
  // under-allocate.
  // This constant variable can be interpreted as: if we still have more than
  // "kAllowOverAllocationRatio * kArenaBlockSize" space left, we'd try to over
  // allocate one more block.
  const double kAllowOverAllocationRatio = 0.6;

  // If arena still have room for new block allocation, we can safely say it
  // shouldn't flush.
  auto allocated_memory =
      table_->ApproximateMemoryUsage() + arena_.MemoryAllocatedBytes();

  // if we can still allocate one more block without exceeding the
  // over-allocation ratio, then we should not flush.
  if (allocated_memory + kArenaBlockSize <
      moptions_.write_buffer_size +
      kArenaBlockSize * kAllowOverAllocationRatio) {
    return false;
  }

  // if user keeps adding entries that exceeds moptions.write_buffer_size,
  // we need to flush earlier even though we still have much available
  // memory left.
  if (allocated_memory > moptions_.write_buffer_size +
      kArenaBlockSize * kAllowOverAllocationRatio) {
    return true;
  }

  // In this code path, Arena has already allocated its "last block", which
  // means the total allocatedmemory size is either:
  //  (1) "moderately" over allocated the memory (no more than `0.6 * arena
  // block size`. Or,
  //  (2) the allocated memory is less than write buffer size, but we'll stop
  // here since if we allocate a new arena block, we'll over allocate too much
  // more (half of the arena block size) memory.
  //
  // In either case, to avoid over-allocate, the last block will stop allocation
  // when its usage reaches a certain ratio, which we carefully choose "0.75
  // full" as the stop condition because it addresses the following issue with
  // great simplicity: What if the next inserted entry's size is
  // bigger than AllocatedAndUnused()?
  //
  // The answer is: if the entry size is also bigger than 0.25 *
  // kArenaBlockSize, a dedicated block will be allocated for it; otherwise
  // arena will anyway skip the AllocatedAndUnused() and allocate a new, empty
  // and regular block. In either case, we *overly* over-allocated.
  //
  // Therefore, setting the last block to be at most "0.75 full" avoids both
  // cases.
  //
  // NOTE: the average percentage of waste space of this approach can be counted
  // as: "arena block size * 0.25 / write buffer size". User who specify a small
  // write buffer size and/or big arena block size may suffer.
  return arena_.AllocatedAndUnused() < kArenaBlockSize / 4;
}

int MemTable::KeyComparator::operator()(const char* prefix_len_key1,
                                        const char* prefix_len_key2) const {
  // Internal keys are encoded as length-prefixed strings.
  Slice k1 = GetLengthPrefixedSlice(prefix_len_key1);
  Slice k2 = GetLengthPrefixedSlice(prefix_len_key2);
  return comparator.Compare(k1, k2);
}

int MemTable::KeyComparator::operator()(const char* prefix_len_key,
                                        const Slice& key)
    const {
  // Internal keys are encoded as length-prefixed strings.
  Slice a = GetLengthPrefixedSlice(prefix_len_key);
  return comparator.Compare(a, key);
}

Slice MemTableRep::UserKey(const char* key) const {
  Slice slice = GetLengthPrefixedSlice(key);
  return Slice(slice.data(), slice.size() - 8);
}

KeyHandle MemTableRep::Allocate(const size_t len, char** buf) {
  *buf = allocator_->Allocate(len);
  return static_cast<KeyHandle>(*buf);
}

// Encode a suitable internal key target for "target" and return it.
// Uses *scratch as scratch space, and the returned pointer will point
// into this scratch space.
const char* EncodeKey(std::string* scratch, const Slice& target) {
  scratch->clear();
  PutVarint32(scratch, static_cast<uint32_t>(target.size()));
  scratch->append(target.data(), target.size());
  return scratch->data();
}

class MemTableIterator: public Iterator {
 public:
  MemTableIterator(
      const MemTable& mem, const ReadOptions& read_options, Arena* arena)
      : bloom_(nullptr),
        prefix_extractor_(mem.prefix_extractor_),
        valid_(false),
        arena_mode_(arena != nullptr) {
    if (prefix_extractor_ != nullptr && !read_options.total_order_seek) {
      bloom_ = mem.prefix_bloom_.get();
      iter_ = mem.table_->GetDynamicPrefixIterator(arena);
    } else {
      iter_ = mem.table_->GetIterator(arena);
    }
  }

  ~MemTableIterator() {
    if (arena_mode_) {
      iter_->~Iterator();
    } else {
      delete iter_;
    }
  }

  virtual bool Valid() const override { return valid_; }
  virtual void Seek(const Slice& k) override {
    PERF_TIMER_GUARD(seek_on_memtable_time);
    PERF_COUNTER_ADD(seek_on_memtable_count, 1);
    if (bloom_ != nullptr &&
        !bloom_->MayContain(prefix_extractor_->Transform(ExtractUserKey(k)))) {
      valid_ = false;
      return;
    }
    iter_->Seek(k, nullptr);
    valid_ = iter_->Valid();
  }
  virtual void SeekToFirst() override {
    iter_->SeekToFirst();
    valid_ = iter_->Valid();
  }
  virtual void SeekToLast() override {
    iter_->SeekToLast();
    valid_ = iter_->Valid();
  }
  virtual void Next() override {
    assert(Valid());
    iter_->Next();
    valid_ = iter_->Valid();
  }
  virtual void Prev() override {
    assert(Valid());
    iter_->Prev();
    valid_ = iter_->Valid();
  }
  virtual Slice key() const override {
    assert(Valid());
    return GetLengthPrefixedSlice(iter_->key());
  }
  virtual Slice value() const override {
    assert(Valid());
    Slice key_slice = GetLengthPrefixedSlice(iter_->key());
    return GetLengthPrefixedSlice(key_slice.data() + key_slice.size());
  }

  virtual Status status() const override { return Status::OK(); }

 private:
  DynamicBloom* bloom_;
  const SliceTransform* const prefix_extractor_;
  MemTableRep::Iterator* iter_;
  bool valid_;
  bool arena_mode_;

  // No copying allowed
  MemTableIterator(const MemTableIterator&);
  void operator=(const MemTableIterator&);
};

Iterator* MemTable::NewIterator(const ReadOptions& read_options, Arena* arena) {
  assert(arena != nullptr);
  auto mem = arena->AllocateAligned(sizeof(MemTableIterator));
  return new (mem) MemTableIterator(*this, read_options, arena);
}

port::RWMutex* MemTable::GetLock(const Slice& key) {
  static murmur_hash hash;
  return &locks_[hash(key) % locks_.size()];
}

uint64_t MemTable::ApproximateSize(const Slice& start_ikey,
                                   const Slice& end_ikey) {
  uint64_t entry_count = table_->ApproximateNumEntries(start_ikey, end_ikey);
  if (entry_count == 0) {
    return 0;
  }
  uint64_t n = num_entries_.load(std::memory_order_relaxed);
  if (n == 0) {
    return 0;
  }
  if (entry_count > n) {
    // table_->ApproximateNumEntries() is just an estimate so it can be larger
    // than actual entries we have. Cap it to entries we have to limit the
    // inaccuracy.
    entry_count = n;
  }
  uint64_t data_size = data_size_.load(std::memory_order_relaxed);
  return entry_count * (data_size / n);
}

void MemTable::Add(SequenceNumber s, ValueType type,
                   const Slice& key, /* user key */
                   const Slice& value) {
  // Format of an entry is concatenation of:
  //  key_size     : varint32 of internal_key.size()
  //  key bytes    : char[internal_key.size()]
  //  value_size   : varint32 of value.size()
  //  value bytes  : char[value.size()]
  uint32_t key_size = static_cast<uint32_t>(key.size());
  uint32_t val_size = static_cast<uint32_t>(value.size());
  uint32_t internal_key_size = key_size + 8;
  const uint32_t encoded_len = VarintLength(internal_key_size) +
                               internal_key_size + VarintLength(val_size) +
                               val_size;
  char* buf = nullptr;
  KeyHandle handle = table_->Allocate(encoded_len, &buf);
  assert(buf != nullptr);
  char* p = EncodeVarint32(buf, internal_key_size);
  memcpy(p, key.data(), key_size);
  p += key_size;
  uint64_t packed = PackSequenceAndType(s, type);
  EncodeFixed64(p, packed);
  p += 8;
  p = EncodeVarint32(p, val_size);
  memcpy(p, value.data(), val_size);
  assert((unsigned)(p + val_size - buf) == (unsigned)encoded_len);
  table_->Insert(handle);
  num_entries_.store(num_entries_.load(std::memory_order_relaxed) + 1,
                     std::memory_order_relaxed);
  data_size_.store(data_size_.load(std::memory_order_relaxed) + encoded_len,
                   std::memory_order_relaxed);
  if (type == kTypeDeletion) {
    num_deletes_++;
  }

  if (prefix_bloom_) {
    assert(prefix_extractor_);
    prefix_bloom_->Add(prefix_extractor_->Transform(key));
  }

  // The first sequence number inserted into the memtable
  assert(first_seqno_ == 0 || s > first_seqno_);
  if (first_seqno_ == 0) {
    first_seqno_ = s;

    if (earliest_seqno_ == kMaxSequenceNumber) {
      earliest_seqno_ = first_seqno_;
    }
    assert(first_seqno_ >= earliest_seqno_);
  }

  should_flush_ = ShouldFlushNow();
}

// Callback from MemTable::Get()
namespace {

struct Saver {
  Status* status;
  const LookupKey* key;
  bool* found_final_value;  // Is value set correctly? Used by KeyMayExist
  bool* merge_in_progress;
  std::string* value;
  SequenceNumber seq;
  const MergeOperator* merge_operator;
  // the merge operations encountered;
  MergeContext* merge_context;
  MemTable* mem;
  Logger* logger;
  Statistics* statistics;
  bool inplace_update_support;
  Env* env_;
};
}  // namespace

static bool SaveValue(void* arg, const char* entry) {
  Saver* s = reinterpret_cast<Saver*>(arg);
  MergeContext* merge_context = s->merge_context;
  const MergeOperator* merge_operator = s->merge_operator;

  assert(s != nullptr && merge_context != nullptr);

  // entry format is:
  //    klength  varint32
  //    userkey  char[klength-8]
  //    tag      uint64
  //    vlength  varint32
  //    value    char[vlength]
  // Check that it belongs to same user key.  We do not check the
  // sequence number since the Seek() call above should have skipped
  // all entries with overly large sequence numbers.
  uint32_t key_length;
  const char* key_ptr = GetVarint32Ptr(entry, entry + 5, &key_length);
  if (s->mem->GetInternalKeyComparator().user_comparator()->Equal(
          Slice(key_ptr, key_length - 8), s->key->user_key())) {
    // Correct user key
    const uint64_t tag = DecodeFixed64(key_ptr + key_length - 8);
    ValueType type;
    UnPackSequenceAndType(tag, &s->seq, &type);

    switch (type) {
      case kTypeValue: {
        if (s->inplace_update_support) {
          s->mem->GetLock(s->key->user_key())->ReadLock();
        }
        Slice v = GetLengthPrefixedSlice(key_ptr + key_length);
        *(s->status) = Status::OK();
        if (*(s->merge_in_progress)) {
          assert(merge_operator);
          bool merge_success = false;
          {
            StopWatchNano timer(s->env_, s->statistics != nullptr);
            PERF_TIMER_GUARD(merge_operator_time_nanos);
            merge_success = merge_operator->FullMerge(
                s->key->user_key(), &v, merge_context->GetOperands(), s->value,
                s->logger);
            RecordTick(s->statistics, MERGE_OPERATION_TOTAL_TIME,
                       timer.ElapsedNanos());
          }
          if (!merge_success) {
            RecordTick(s->statistics, NUMBER_MERGE_FAILURES);
            *(s->status) =
                Status::Corruption("Error: Could not perform merge.");
          }
        } else {
          s->value->assign(v.data(), v.size());
        }
        if (s->inplace_update_support) {
          s->mem->GetLock(s->key->user_key())->ReadUnlock();
        }
        *(s->found_final_value) = true;
        return false;
      }
      case kTypeDeletion: {
        if (*(s->merge_in_progress)) {
          assert(merge_operator);
          *(s->status) = Status::OK();
          bool merge_success = false;
          {
            StopWatchNano timer(s->env_, s->statistics != nullptr);
            PERF_TIMER_GUARD(merge_operator_time_nanos);
            merge_success = merge_operator->FullMerge(
                s->key->user_key(), nullptr, merge_context->GetOperands(),
                s->value, s->logger);
            RecordTick(s->statistics, MERGE_OPERATION_TOTAL_TIME,
                       timer.ElapsedNanos());
          }
          if (!merge_success) {
            RecordTick(s->statistics, NUMBER_MERGE_FAILURES);
            *(s->status) =
                Status::Corruption("Error: Could not perform merge.");
          }
        } else {
          *(s->status) = Status::NotFound();
        }
        *(s->found_final_value) = true;
        return false;
      }
      case kTypeMerge: {
        if (!merge_operator) {
          *(s->status) = Status::InvalidArgument(
              "merge_operator is not properly initialized.");
          // Normally we continue the loop (return true) when we see a merge
          // operand.  But in case of an error, we should stop the loop
          // immediately and pretend we have found the value to stop further
          // seek.  Otherwise, the later call will override this error status.
          *(s->found_final_value) = true;
          return false;
        }
        Slice v = GetLengthPrefixedSlice(key_ptr + key_length);
        *(s->merge_in_progress) = true;
        merge_context->PushOperand(v);
        return true;
      }
      default:
        assert(false);
        return true;
    }
  }

  // s->state could be Corrupt, merge or notfound
  return false;
}

bool MemTable::Get(const LookupKey& key, std::string* value, Status* s,
                   MergeContext* merge_context, SequenceNumber* seq) {
  // The sequence number is updated synchronously in version_set.h
  if (IsEmpty()) {
    // Avoiding recording stats for speed.
    return false;
  }
  PERF_TIMER_GUARD(get_from_memtable_time);

  Slice user_key = key.user_key();
  bool found_final_value = false;
  bool merge_in_progress = s->IsMergeInProgress();

  if (prefix_bloom_ &&
      !prefix_bloom_->MayContain(prefix_extractor_->Transform(user_key))) {
    // iter is null if prefix bloom says the key does not exist
    *seq = kMaxSequenceNumber;
  } else {
    Saver saver;
    saver.status = s;
    saver.found_final_value = &found_final_value;
    saver.merge_in_progress = &merge_in_progress;
    saver.key = &key;
    saver.value = value;
    saver.seq = kMaxSequenceNumber;
    saver.mem = this;
    saver.merge_context = merge_context;
    saver.merge_operator = moptions_.merge_operator;
    saver.logger = moptions_.info_log;
    saver.inplace_update_support = moptions_.inplace_update_support;
    saver.statistics = moptions_.statistics;
    saver.env_ = env_;
    table_->Get(key, &saver, SaveValue);

    *seq = saver.seq;
  }

  // No change to value, since we have not yet found a Put/Delete
  if (!found_final_value && merge_in_progress) {
    *s = Status::MergeInProgress();
  }
  PERF_COUNTER_ADD(get_from_memtable_count, 1);
  return found_final_value;
}

void MemTable::Update(SequenceNumber seq,
                      const Slice& key,
                      const Slice& value) {
  LookupKey lkey(key, seq);
  Slice mem_key = lkey.memtable_key();

  std::unique_ptr<MemTableRep::Iterator> iter(
      table_->GetDynamicPrefixIterator());
  iter->Seek(lkey.internal_key(), mem_key.data());

  if (iter->Valid()) {
    // entry format is:
    //    key_length  varint32
    //    userkey  char[klength-8]
    //    tag      uint64
    //    vlength  varint32
    //    value    char[vlength]
    // Check that it belongs to same user key.  We do not check the
    // sequence number since the Seek() call above should have skipped
    // all entries with overly large sequence numbers.
    const char* entry = iter->key();
    uint32_t key_length = 0;
    const char* key_ptr = GetVarint32Ptr(entry, entry + 5, &key_length);
    if (comparator_.comparator.user_comparator()->Equal(
            Slice(key_ptr, key_length - 8), lkey.user_key())) {
      // Correct user key
      const uint64_t tag = DecodeFixed64(key_ptr + key_length - 8);
      ValueType type;
      SequenceNumber unused;
      UnPackSequenceAndType(tag, &unused, &type);
      switch (type) {
        case kTypeValue: {
          Slice prev_value = GetLengthPrefixedSlice(key_ptr + key_length);
          uint32_t prev_size = static_cast<uint32_t>(prev_value.size());
          uint32_t new_size = static_cast<uint32_t>(value.size());

          // Update value, if new value size  <= previous value size
          if (new_size <= prev_size ) {
            char* p = EncodeVarint32(const_cast<char*>(key_ptr) + key_length,
                                     new_size);
            WriteLock wl(GetLock(lkey.user_key()));
            memcpy(p, value.data(), value.size());
            assert((unsigned)((p + value.size()) - entry) ==
                   (unsigned)(VarintLength(key_length) + key_length +
                              VarintLength(value.size()) + value.size()));
            return;
          }
        }
        default:
          // If the latest value is kTypeDeletion, kTypeMerge or kTypeLogData
          // we don't have enough space for update inplace
            Add(seq, kTypeValue, key, value);
            return;
      }
    }
  }

  // key doesn't exist
  Add(seq, kTypeValue, key, value);
}

bool MemTable::UpdateCallback(SequenceNumber seq,
                              const Slice& key,
                              const Slice& delta) {
  LookupKey lkey(key, seq);
  Slice memkey = lkey.memtable_key();

  std::unique_ptr<MemTableRep::Iterator> iter(
      table_->GetDynamicPrefixIterator());
  iter->Seek(lkey.internal_key(), memkey.data());

  if (iter->Valid()) {
    // entry format is:
    //    key_length  varint32
    //    userkey  char[klength-8]
    //    tag      uint64
    //    vlength  varint32
    //    value    char[vlength]
    // Check that it belongs to same user key.  We do not check the
    // sequence number since the Seek() call above should have skipped
    // all entries with overly large sequence numbers.
    const char* entry = iter->key();
    uint32_t key_length = 0;
    const char* key_ptr = GetVarint32Ptr(entry, entry + 5, &key_length);
    if (comparator_.comparator.user_comparator()->Equal(
            Slice(key_ptr, key_length - 8), lkey.user_key())) {
      // Correct user key
      const uint64_t tag = DecodeFixed64(key_ptr + key_length - 8);
      ValueType type;
      uint64_t unused;
      UnPackSequenceAndType(tag, &unused, &type);
      switch (type) {
        case kTypeValue: {
          Slice prev_value = GetLengthPrefixedSlice(key_ptr + key_length);
          uint32_t prev_size = static_cast<uint32_t>(prev_value.size());

          char* prev_buffer = const_cast<char*>(prev_value.data());
          uint32_t new_prev_size = prev_size;

          std::string str_value;
          WriteLock wl(GetLock(lkey.user_key()));
          auto status = moptions_.inplace_callback(prev_buffer, &new_prev_size,
                                                   delta, &str_value);
          if (status == UpdateStatus::UPDATED_INPLACE) {
            // Value already updated by callback.
            assert(new_prev_size <= prev_size);
            if (new_prev_size < prev_size) {
              // overwrite the new prev_size
              char* p = EncodeVarint32(const_cast<char*>(key_ptr) + key_length,
                                       new_prev_size);
              if (VarintLength(new_prev_size) < VarintLength(prev_size)) {
                // shift the value buffer as well.
                memcpy(p, prev_buffer, new_prev_size);
              }
            }
            RecordTick(moptions_.statistics, NUMBER_KEYS_UPDATED);
            should_flush_ = ShouldFlushNow();
            return true;
          } else if (status == UpdateStatus::UPDATED) {
            Add(seq, kTypeValue, key, Slice(str_value));
            RecordTick(moptions_.statistics, NUMBER_KEYS_WRITTEN);
            should_flush_ = ShouldFlushNow();
            return true;
          } else if (status == UpdateStatus::UPDATE_FAILED) {
            // No action required. Return.
            should_flush_ = ShouldFlushNow();
            return true;
          }
        }
        default:
          break;
      }
    }
  }
  // If the latest value is not kTypeValue
  // or key doesn't exist
  return false;
}

size_t MemTable::CountSuccessiveMergeEntries(const LookupKey& key) {
  Slice memkey = key.memtable_key();

  // A total ordered iterator is costly for some memtablerep (prefix aware
  // reps). By passing in the user key, we allow efficient iterator creation.
  // The iterator only needs to be ordered within the same user key.
  std::unique_ptr<MemTableRep::Iterator> iter(
      table_->GetDynamicPrefixIterator());
  iter->Seek(key.internal_key(), memkey.data());

  size_t num_successive_merges = 0;

  for (; iter->Valid(); iter->Next()) {
    const char* entry = iter->key();
    uint32_t key_length = 0;
    const char* iter_key_ptr = GetVarint32Ptr(entry, entry + 5, &key_length);
    if (!comparator_.comparator.user_comparator()->Equal(
            Slice(iter_key_ptr, key_length - 8), key.user_key())) {
      break;
    }

    const uint64_t tag = DecodeFixed64(iter_key_ptr + key_length - 8);
    ValueType type;
    uint64_t unused;
    UnPackSequenceAndType(tag, &unused, &type);
    if (type != kTypeMerge) {
      break;
    }

    ++num_successive_merges;
  }

  return num_successive_merges;
}

void MemTableRep::Get(const LookupKey& k, void* callback_args,
                      bool (*callback_func)(void* arg, const char* entry)) {
  auto iter = GetDynamicPrefixIterator();
  for (iter->Seek(k.internal_key(), k.memtable_key().data());
       iter->Valid() && callback_func(callback_args, iter->key());
       iter->Next()) {
  }
}

}  // namespace rocksdb
