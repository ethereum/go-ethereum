//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "db/column_family.h"

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <inttypes.h>
#include <vector>
#include <string>
#include <algorithm>
#include <limits>

#include "db/compaction_picker.h"
#include "db/db_impl.h"
#include "db/internal_stats.h"
#include "db/job_context.h"
#include "db/table_properties_collector.h"
#include "db/version_set.h"
#include "db/write_controller.h"
#include "db/writebuffer.h"
#include "util/autovector.h"
#include "util/compression.h"
#include "util/hash_skiplist_rep.h"
#include "util/options_helper.h"
#include "util/thread_status_util.h"
#include "util/xfunc.h"

namespace rocksdb {

ColumnFamilyHandleImpl::ColumnFamilyHandleImpl(
    ColumnFamilyData* column_family_data, DBImpl* db, InstrumentedMutex* mutex)
    : cfd_(column_family_data), db_(db), mutex_(mutex) {
  if (cfd_ != nullptr) {
    cfd_->Ref();
  }
}

ColumnFamilyHandleImpl::~ColumnFamilyHandleImpl() {
  if (cfd_ != nullptr) {
    // Job id == 0 means that this is not our background process, but rather
    // user thread
    JobContext job_context(0);
    mutex_->Lock();
    if (cfd_->Unref()) {
      delete cfd_;
    }
    db_->FindObsoleteFiles(&job_context, false, true);
    mutex_->Unlock();
    if (job_context.HaveSomethingToDelete()) {
      db_->PurgeObsoleteFiles(job_context);
    }
    job_context.Clean();
  }
}

uint32_t ColumnFamilyHandleImpl::GetID() const { return cfd()->GetID(); }

const std::string& ColumnFamilyHandleImpl::GetName() const {
  return cfd()->GetName();
}

const Comparator* ColumnFamilyHandleImpl::user_comparator() const {
  return cfd()->user_comparator();
}

void GetIntTblPropCollectorFactory(
    const ColumnFamilyOptions& cf_options,
    std::vector<std::unique_ptr<IntTblPropCollectorFactory>>*
        int_tbl_prop_collector_factories) {
  auto& collector_factories = cf_options.table_properties_collector_factories;
  for (size_t i = 0; i < cf_options.table_properties_collector_factories.size();
       ++i) {
    assert(collector_factories[i]);
    int_tbl_prop_collector_factories->emplace_back(
        new UserKeyTablePropertiesCollectorFactory(collector_factories[i]));
  }
  // Add collector to collect internal key statistics
  int_tbl_prop_collector_factories->emplace_back(
      new InternalKeyPropertiesCollectorFactory);
}

Status CheckCompressionSupported(const ColumnFamilyOptions& cf_options) {
  if (!cf_options.compression_per_level.empty()) {
    for (size_t level = 0; level < cf_options.compression_per_level.size();
         ++level) {
      if (!CompressionTypeSupported(cf_options.compression_per_level[level])) {
        return Status::InvalidArgument(
            "Compression type " +
            CompressionTypeToString(cf_options.compression_per_level[level]) +
            " is not linked with the binary.");
      }
    }
  } else {
    if (!CompressionTypeSupported(cf_options.compression)) {
      return Status::InvalidArgument(
          "Compression type " +
          CompressionTypeToString(cf_options.compression) +
          " is not linked with the binary.");
    }
  }
  return Status::OK();
}

ColumnFamilyOptions SanitizeOptions(const DBOptions& db_options,
                                    const InternalKeyComparator* icmp,
                                    const ColumnFamilyOptions& src) {
  ColumnFamilyOptions result = src;
  result.comparator = icmp;
#ifdef OS_MACOSX
  // TODO(icanadi) make write_buffer_size uint64_t instead of size_t
  ClipToRange(&result.write_buffer_size, ((size_t)64) << 10, ((size_t)1) << 30);
#else
  ClipToRange(&result.write_buffer_size,
              ((size_t)64) << 10, ((size_t)64) << 30);
#endif
  // if user sets arena_block_size, we trust user to use this value. Otherwise,
  // calculate a proper value from writer_buffer_size;
  if (result.arena_block_size <= 0) {
    result.arena_block_size = result.write_buffer_size / 8;

    // Align up to 4k
    const size_t align = 4 * 1024;
    result.arena_block_size =
        ((result.arena_block_size + align - 1) / align) * align;
  }
  result.min_write_buffer_number_to_merge =
      std::min(result.min_write_buffer_number_to_merge,
               result.max_write_buffer_number - 1);
  if (result.num_levels < 1) {
    result.num_levels = 1;
  }
  if (result.compaction_style == kCompactionStyleLevel &&
      result.num_levels < 2) {
    result.num_levels = 2;
  }
  if (result.max_write_buffer_number < 2) {
    result.max_write_buffer_number = 2;
  }
  if (result.max_write_buffer_number_to_maintain < 0) {
    result.max_write_buffer_number_to_maintain = result.max_write_buffer_number;
  }
  XFUNC_TEST("memtablelist_history", "transaction_xftest_SanitizeOptions",
             xf_transaction_set_memtable_history1,
             xf_transaction_set_memtable_history,
             &result.max_write_buffer_number_to_maintain);
  XFUNC_TEST("memtablelist_history_clear", "transaction_xftest_SanitizeOptions",
             xf_transaction_clear_memtable_history1,
             xf_transaction_clear_memtable_history,
             &result.max_write_buffer_number_to_maintain);

  if (!result.prefix_extractor) {
    assert(result.memtable_factory);
    Slice name = result.memtable_factory->Name();
    if (name.compare("HashSkipListRepFactory") == 0 ||
        name.compare("HashLinkListRepFactory") == 0) {
      result.memtable_factory = std::make_shared<SkipListFactory>();
    }
  }

  if (result.compaction_style == kCompactionStyleFIFO) {
    result.num_levels = 1;
    // since we delete level0 files in FIFO compaction when there are too many
    // of them, these options don't really mean anything
    result.level0_file_num_compaction_trigger = std::numeric_limits<int>::max();
    result.level0_slowdown_writes_trigger = std::numeric_limits<int>::max();
    result.level0_stop_writes_trigger = std::numeric_limits<int>::max();
  }

  if (result.level0_stop_writes_trigger <
          result.level0_slowdown_writes_trigger ||
      result.level0_slowdown_writes_trigger <
          result.level0_file_num_compaction_trigger) {
    Warn(db_options.info_log.get(),
         "This condition must be satisfied: "
         "level0_stop_writes_trigger(%d) >= "
         "level0_slowdown_writes_trigger(%d) >= "
         "level0_file_num_compaction_trigger(%d)",
         result.level0_stop_writes_trigger,
         result.level0_slowdown_writes_trigger,
         result.level0_file_num_compaction_trigger);
    if (result.level0_slowdown_writes_trigger <
        result.level0_file_num_compaction_trigger) {
      result.level0_slowdown_writes_trigger =
          result.level0_file_num_compaction_trigger;
    }
    if (result.level0_stop_writes_trigger <
        result.level0_slowdown_writes_trigger) {
      result.level0_stop_writes_trigger = result.level0_slowdown_writes_trigger;
    }
    Warn(db_options.info_log.get(),
         "Adjust the value to "
         "level0_stop_writes_trigger(%d)"
         "level0_slowdown_writes_trigger(%d)"
         "level0_file_num_compaction_trigger(%d)",
         result.level0_stop_writes_trigger,
         result.level0_slowdown_writes_trigger,
         result.level0_file_num_compaction_trigger);
  }
  if (result.level_compaction_dynamic_level_bytes) {
    if (result.compaction_style != kCompactionStyleLevel ||
        db_options.db_paths.size() > 1U) {
      // 1. level_compaction_dynamic_level_bytes only makes sense for
      //    level-based compaction.
      // 2. we don't yet know how to make both of this feature and multiple
      //    DB path work.
      result.level_compaction_dynamic_level_bytes = false;
    }
  }

  return result;
}

int SuperVersion::dummy = 0;
void* const SuperVersion::kSVInUse = &SuperVersion::dummy;
void* const SuperVersion::kSVObsolete = nullptr;

SuperVersion::~SuperVersion() {
  for (auto td : to_delete) {
    delete td;
  }
}

SuperVersion* SuperVersion::Ref() {
  refs.fetch_add(1, std::memory_order_relaxed);
  return this;
}

bool SuperVersion::Unref() {
  // fetch_sub returns the previous value of ref
  uint32_t previous_refs = refs.fetch_sub(1);
  assert(previous_refs > 0);
  return previous_refs == 1;
}

void SuperVersion::Cleanup() {
  assert(refs.load(std::memory_order_relaxed) == 0);
  imm->Unref(&to_delete);
  MemTable* m = mem->Unref();
  if (m != nullptr) {
    auto* memory_usage = current->cfd()->imm()->current_memory_usage();
    assert(*memory_usage >= m->ApproximateMemoryUsage());
    *memory_usage -= m->ApproximateMemoryUsage();
    to_delete.push_back(m);
  }
  current->Unref();
}

void SuperVersion::Init(MemTable* new_mem, MemTableListVersion* new_imm,
                        Version* new_current) {
  mem = new_mem;
  imm = new_imm;
  current = new_current;
  mem->Ref();
  imm->Ref();
  current->Ref();
  refs.store(1, std::memory_order_relaxed);
}

namespace {
void SuperVersionUnrefHandle(void* ptr) {
  // UnrefHandle is called when a thread exists or a ThreadLocalPtr gets
  // destroyed. When former happens, the thread shouldn't see kSVInUse.
  // When latter happens, we are in ~ColumnFamilyData(), no get should happen as
  // well.
  SuperVersion* sv = static_cast<SuperVersion*>(ptr);
  if (sv->Unref()) {
    sv->db_mutex->Lock();
    sv->Cleanup();
    sv->db_mutex->Unlock();
    delete sv;
  }
}
}  // anonymous namespace

ColumnFamilyData::ColumnFamilyData(
    uint32_t id, const std::string& name, Version* _dummy_versions,
    Cache* _table_cache, WriteBuffer* write_buffer,
    const ColumnFamilyOptions& cf_options, const DBOptions* db_options,
    const EnvOptions& env_options, ColumnFamilySet* column_family_set)
    : id_(id),
      name_(name),
      dummy_versions_(_dummy_versions),
      current_(nullptr),
      refs_(0),
      dropped_(false),
      internal_comparator_(cf_options.comparator),
      options_(*db_options,
               SanitizeOptions(*db_options, &internal_comparator_, cf_options)),
      ioptions_(options_),
      mutable_cf_options_(options_, ioptions_),
      write_buffer_(write_buffer),
      mem_(nullptr),
      imm_(options_.min_write_buffer_number_to_merge,
           options_.max_write_buffer_number_to_maintain),
      super_version_(nullptr),
      super_version_number_(0),
      local_sv_(new ThreadLocalPtr(&SuperVersionUnrefHandle)),
      next_(nullptr),
      prev_(nullptr),
      log_number_(0),
      column_family_set_(column_family_set),
      pending_flush_(false),
      pending_compaction_(false) {
  Ref();

  // Convert user defined table properties collector factories to internal ones.
  GetIntTblPropCollectorFactory(options_, &int_tbl_prop_collector_factories_);

  // if _dummy_versions is nullptr, then this is a dummy column family.
  if (_dummy_versions != nullptr) {
    internal_stats_.reset(
        new InternalStats(ioptions_.num_levels, db_options->env, this));
    table_cache_.reset(new TableCache(ioptions_, env_options, _table_cache));
    if (ioptions_.compaction_style == kCompactionStyleLevel) {
      compaction_picker_.reset(
          new LevelCompactionPicker(ioptions_, &internal_comparator_));
#ifndef ROCKSDB_LITE
    } else if (ioptions_.compaction_style == kCompactionStyleUniversal) {
      compaction_picker_.reset(
          new UniversalCompactionPicker(ioptions_, &internal_comparator_));
    } else if (ioptions_.compaction_style == kCompactionStyleFIFO) {
      compaction_picker_.reset(
          new FIFOCompactionPicker(ioptions_, &internal_comparator_));
    } else if (ioptions_.compaction_style == kCompactionStyleNone) {
      compaction_picker_.reset(new NullCompactionPicker(
          ioptions_, &internal_comparator_));
      Log(InfoLogLevel::WARN_LEVEL, ioptions_.info_log,
          "Column family %s does not use any background compaction. "
          "Compactions can only be done via CompactFiles\n",
          GetName().c_str());
#endif  // !ROCKSDB_LITE
    } else {
      Log(InfoLogLevel::ERROR_LEVEL, ioptions_.info_log,
          "Unable to recognize the specified compaction style %d. "
          "Column family %s will use kCompactionStyleLevel.\n",
          ioptions_.compaction_style, GetName().c_str());
      compaction_picker_.reset(
          new LevelCompactionPicker(ioptions_, &internal_comparator_));
    }

    if (column_family_set_->NumberOfColumnFamilies() < 10) {
      Log(InfoLogLevel::INFO_LEVEL, ioptions_.info_log,
          "--------------- Options for column family [%s]:\n", name.c_str());
      options_.DumpCFOptions(ioptions_.info_log);
    } else {
      Log(InfoLogLevel::INFO_LEVEL, ioptions_.info_log,
          "\t(skipping printing options)\n");
    }
  }

  RecalculateWriteStallConditions(mutable_cf_options_);
}

// DB mutex held
ColumnFamilyData::~ColumnFamilyData() {
  assert(refs_.load(std::memory_order_relaxed) == 0);
  // remove from linked list
  auto prev = prev_;
  auto next = next_;
  prev->next_ = next;
  next->prev_ = prev;

  if (!dropped_ && column_family_set_ != nullptr) {
    // If it's dropped, it's already removed from column family set
    // If column_family_set_ == nullptr, this is dummy CFD and not in
    // ColumnFamilySet
    column_family_set_->RemoveColumnFamily(this);
  }

  if (current_ != nullptr) {
    current_->Unref();
  }

  // It would be wrong if this ColumnFamilyData is in flush_queue_ or
  // compaction_queue_ and we destroyed it
  assert(!pending_flush_);
  assert(!pending_compaction_);

  if (super_version_ != nullptr) {
    // Release SuperVersion reference kept in ThreadLocalPtr.
    // This must be done outside of mutex_ since unref handler can lock mutex.
    super_version_->db_mutex->Unlock();
    local_sv_.reset();
    super_version_->db_mutex->Lock();

    bool is_last_reference __attribute__((unused));
    is_last_reference = super_version_->Unref();
    assert(is_last_reference);
    super_version_->Cleanup();
    delete super_version_;
    super_version_ = nullptr;
  }

  if (dummy_versions_ != nullptr) {
    // List must be empty
    assert(dummy_versions_->TEST_Next() == dummy_versions_);
    bool deleted __attribute__((unused)) = dummy_versions_->Unref();
    assert(deleted);
  }

  if (mem_ != nullptr) {
    delete mem_->Unref();
  }
  autovector<MemTable*> to_delete;
  imm_.current()->Unref(&to_delete);
  for (MemTable* m : to_delete) {
    delete m;
  }
}

void ColumnFamilyData::SetDropped() {
  // can't drop default CF
  assert(id_ != 0);
  dropped_ = true;
  write_controller_token_.reset();

  // remove from column_family_set
  column_family_set_->RemoveColumnFamily(this);
}

void ColumnFamilyData::RecalculateWriteStallConditions(
      const MutableCFOptions& mutable_cf_options) {
  if (current_ != nullptr) {
    auto* vstorage = current_->storage_info();
    const double score = vstorage->max_compaction_score();
    const int max_level = vstorage->max_compaction_score_level();
    auto write_controller = column_family_set_->write_controller_;

    if (imm()->NumNotFlushed() >= mutable_cf_options.max_write_buffer_number) {
      write_controller_token_ = write_controller->GetStopToken();
      internal_stats_->AddCFStats(InternalStats::MEMTABLE_COMPACTION, 1);
      Log(InfoLogLevel::WARN_LEVEL, ioptions_.info_log,
          "[%s] Stopping writes because we have %d immutable memtables "
          "(waiting for flush), max_write_buffer_number is set to %d",
          name_.c_str(), imm()->NumNotFlushed(),
          mutable_cf_options.max_write_buffer_number);
    } else if (vstorage->l0_delay_trigger_count() >=
               mutable_cf_options.level0_stop_writes_trigger) {
      write_controller_token_ = write_controller->GetStopToken();
      internal_stats_->AddCFStats(InternalStats::LEVEL0_NUM_FILES, 1);
      Log(InfoLogLevel::WARN_LEVEL, ioptions_.info_log,
          "[%s] Stopping writes because we have %d level-0 files",
          name_.c_str(), vstorage->l0_delay_trigger_count());
    } else if (mutable_cf_options.level0_slowdown_writes_trigger >= 0 &&
               vstorage->l0_delay_trigger_count() >=
                   mutable_cf_options.level0_slowdown_writes_trigger) {
      write_controller_token_ = write_controller->GetDelayToken();
      internal_stats_->AddCFStats(InternalStats::LEVEL0_SLOWDOWN, 1);
      Log(InfoLogLevel::WARN_LEVEL, ioptions_.info_log,
          "[%s] Stalling writes because we have %d level-0 files",
          name_.c_str(), vstorage->l0_delay_trigger_count());
    } else if (mutable_cf_options.soft_rate_limit > 0.0 &&
               score > mutable_cf_options.soft_rate_limit) {
      write_controller_token_ = write_controller->GetDelayToken();
      internal_stats_->RecordLevelNSlowdown(max_level, true);
      Log(InfoLogLevel::WARN_LEVEL, ioptions_.info_log,
          "[%s] Stalling writes because we hit soft limit on level %d",
          name_.c_str(), max_level);
    } else {
      write_controller_token_.reset();
    }
  }
}

const EnvOptions* ColumnFamilyData::soptions() const {
  return &(column_family_set_->env_options_);
}

void ColumnFamilyData::SetCurrent(Version* current_version) {
  current_ = current_version;
}

uint64_t ColumnFamilyData::GetNumLiveVersions() const {
  return VersionSet::GetNumLiveVersions(dummy_versions_);
}

uint64_t ColumnFamilyData::GetTotalSstFilesSize() const {
  return VersionSet::GetTotalSstFilesSize(dummy_versions_);
}

MemTable* ColumnFamilyData::ConstructNewMemtable(
    const MutableCFOptions& mutable_cf_options, SequenceNumber earliest_seq) {
  assert(current_ != nullptr);
  return new MemTable(internal_comparator_, ioptions_, mutable_cf_options,
                      write_buffer_, earliest_seq);
}

void ColumnFamilyData::CreateNewMemtable(
    const MutableCFOptions& mutable_cf_options, SequenceNumber earliest_seq) {
  if (mem_ != nullptr) {
    delete mem_->Unref();
  }
  SetMemtable(ConstructNewMemtable(mutable_cf_options, earliest_seq));
  mem_->Ref();
}

bool ColumnFamilyData::NeedsCompaction() const {
  return compaction_picker_->NeedsCompaction(current_->storage_info());
}

Compaction* ColumnFamilyData::PickCompaction(
    const MutableCFOptions& mutable_options, LogBuffer* log_buffer) {
  auto* result = compaction_picker_->PickCompaction(
      GetName(), mutable_options, current_->storage_info(), log_buffer);
  if (result != nullptr) {
    result->SetInputVersion(current_);
  }
  return result;
}

const int ColumnFamilyData::kCompactAllLevels = -1;
const int ColumnFamilyData::kCompactToBaseLevel = -2;

Compaction* ColumnFamilyData::CompactRange(
    const MutableCFOptions& mutable_cf_options,
    int input_level, int output_level, uint32_t output_path_id,
    const InternalKey* begin, const InternalKey* end,
    InternalKey** compaction_end) {
  auto* result = compaction_picker_->CompactRange(
      GetName(), mutable_cf_options, current_->storage_info(), input_level,
      output_level, output_path_id, begin, end, compaction_end);
  if (result != nullptr) {
    result->SetInputVersion(current_);
  }
  return result;
}

SuperVersion* ColumnFamilyData::GetReferencedSuperVersion(
    InstrumentedMutex* db_mutex) {
  SuperVersion* sv = nullptr;
  sv = GetThreadLocalSuperVersion(db_mutex);
  sv->Ref();
  if (!ReturnThreadLocalSuperVersion(sv)) {
    sv->Unref();
  }
  return sv;
}

SuperVersion* ColumnFamilyData::GetThreadLocalSuperVersion(
    InstrumentedMutex* db_mutex) {
  SuperVersion* sv = nullptr;
  // The SuperVersion is cached in thread local storage to avoid acquiring
  // mutex when SuperVersion does not change since the last use. When a new
  // SuperVersion is installed, the compaction or flush thread cleans up
  // cached SuperVersion in all existing thread local storage. To avoid
  // acquiring mutex for this operation, we use atomic Swap() on the thread
  // local pointer to guarantee exclusive access. If the thread local pointer
  // is being used while a new SuperVersion is installed, the cached
  // SuperVersion can become stale. In that case, the background thread would
  // have swapped in kSVObsolete. We re-check the value at when returning
  // SuperVersion back to thread local, with an atomic compare and swap.
  // The superversion will need to be released if detected to be stale.
  void* ptr = local_sv_->Swap(SuperVersion::kSVInUse);
  // Invariant:
  // (1) Scrape (always) installs kSVObsolete in ThreadLocal storage
  // (2) the Swap above (always) installs kSVInUse, ThreadLocal storage
  // should only keep kSVInUse before ReturnThreadLocalSuperVersion call
  // (if no Scrape happens).
  assert(ptr != SuperVersion::kSVInUse);
  sv = static_cast<SuperVersion*>(ptr);
  if (sv == SuperVersion::kSVObsolete ||
      sv->version_number != super_version_number_.load()) {
    RecordTick(ioptions_.statistics, NUMBER_SUPERVERSION_ACQUIRES);
    SuperVersion* sv_to_delete = nullptr;

    if (sv && sv->Unref()) {
      RecordTick(ioptions_.statistics, NUMBER_SUPERVERSION_CLEANUPS);
      db_mutex->Lock();
      // NOTE: underlying resources held by superversion (sst files) might
      // not be released until the next background job.
      sv->Cleanup();
      sv_to_delete = sv;
    } else {
      db_mutex->Lock();
    }
    sv = super_version_->Ref();
    db_mutex->Unlock();

    delete sv_to_delete;
  }
  assert(sv != nullptr);
  return sv;
}

bool ColumnFamilyData::ReturnThreadLocalSuperVersion(SuperVersion* sv) {
  assert(sv != nullptr);
  // Put the SuperVersion back
  void* expected = SuperVersion::kSVInUse;
  if (local_sv_->CompareAndSwap(static_cast<void*>(sv), expected)) {
    // When we see kSVInUse in the ThreadLocal, we are sure ThreadLocal
    // storage has not been altered and no Scrape has happened. The
    // SuperVersion is still current.
    return true;
  } else {
    // ThreadLocal scrape happened in the process of this GetImpl call (after
    // thread local Swap() at the beginning and before CompareAndSwap()).
    // This means the SuperVersion it holds is obsolete.
    assert(expected == SuperVersion::kSVObsolete);
  }
  return false;
}

SuperVersion* ColumnFamilyData::InstallSuperVersion(
    SuperVersion* new_superversion, InstrumentedMutex* db_mutex) {
  db_mutex->AssertHeld();
  return InstallSuperVersion(new_superversion, db_mutex, mutable_cf_options_);
}

SuperVersion* ColumnFamilyData::InstallSuperVersion(
    SuperVersion* new_superversion, InstrumentedMutex* db_mutex,
    const MutableCFOptions& mutable_cf_options) {
  new_superversion->db_mutex = db_mutex;
  new_superversion->mutable_cf_options = mutable_cf_options;
  new_superversion->Init(mem_, imm_.current(), current_);
  SuperVersion* old_superversion = super_version_;
  super_version_ = new_superversion;
  ++super_version_number_;
  super_version_->version_number = super_version_number_;
  // Reset SuperVersions cached in thread local storage
  ResetThreadLocalSuperVersions();

  RecalculateWriteStallConditions(mutable_cf_options);

  if (old_superversion != nullptr && old_superversion->Unref()) {
    old_superversion->Cleanup();
    return old_superversion;  // will let caller delete outside of mutex
  }
  return nullptr;
}

void ColumnFamilyData::ResetThreadLocalSuperVersions() {
  autovector<void*> sv_ptrs;
  local_sv_->Scrape(&sv_ptrs, SuperVersion::kSVObsolete);
  for (auto ptr : sv_ptrs) {
    assert(ptr);
    if (ptr == SuperVersion::kSVInUse) {
      continue;
    }
    auto sv = static_cast<SuperVersion*>(ptr);
    if (sv->Unref()) {
      sv->Cleanup();
      delete sv;
    }
  }
}

#ifndef ROCKSDB_LITE
Status ColumnFamilyData::SetOptions(
      const std::unordered_map<std::string, std::string>& options_map) {
  MutableCFOptions new_mutable_cf_options;
  Status s = GetMutableOptionsFromStrings(mutable_cf_options_, options_map,
                                          &new_mutable_cf_options);
  if (s.ok()) {
    mutable_cf_options_ = new_mutable_cf_options;
    mutable_cf_options_.RefreshDerivedOptions(ioptions_);
  }
  return s;
}
#endif  // ROCKSDB_LITE

ColumnFamilySet::ColumnFamilySet(const std::string& dbname,
                                 const DBOptions* db_options,
                                 const EnvOptions& env_options,
                                 Cache* table_cache,
                                 WriteBuffer* write_buffer,
                                 WriteController* write_controller)
    : max_column_family_(0),
      dummy_cfd_(new ColumnFamilyData(0, "", nullptr, nullptr, nullptr,
                                      ColumnFamilyOptions(), db_options,
                                      env_options, nullptr)),
      default_cfd_cache_(nullptr),
      db_name_(dbname),
      db_options_(db_options),
      env_options_(env_options),
      table_cache_(table_cache),
      write_buffer_(write_buffer),
      write_controller_(write_controller) {
  // initialize linked list
  dummy_cfd_->prev_ = dummy_cfd_;
  dummy_cfd_->next_ = dummy_cfd_;
}

ColumnFamilySet::~ColumnFamilySet() {
  while (column_family_data_.size() > 0) {
    // cfd destructor will delete itself from column_family_data_
    auto cfd = column_family_data_.begin()->second;
    cfd->Unref();
    delete cfd;
  }
  dummy_cfd_->Unref();
  delete dummy_cfd_;
}

ColumnFamilyData* ColumnFamilySet::GetDefault() const {
  assert(default_cfd_cache_ != nullptr);
  return default_cfd_cache_;
}

ColumnFamilyData* ColumnFamilySet::GetColumnFamily(uint32_t id) const {
  auto cfd_iter = column_family_data_.find(id);
  if (cfd_iter != column_family_data_.end()) {
    return cfd_iter->second;
  } else {
    return nullptr;
  }
}

ColumnFamilyData* ColumnFamilySet::GetColumnFamily(const std::string& name)
    const {
  auto cfd_iter = column_families_.find(name);
  if (cfd_iter != column_families_.end()) {
    auto cfd = GetColumnFamily(cfd_iter->second);
    assert(cfd != nullptr);
    return cfd;
  } else {
    return nullptr;
  }
}

uint32_t ColumnFamilySet::GetNextColumnFamilyID() {
  return ++max_column_family_;
}

uint32_t ColumnFamilySet::GetMaxColumnFamily() { return max_column_family_; }

void ColumnFamilySet::UpdateMaxColumnFamily(uint32_t new_max_column_family) {
  max_column_family_ = std::max(new_max_column_family, max_column_family_);
}

size_t ColumnFamilySet::NumberOfColumnFamilies() const {
  return column_families_.size();
}

// under a DB mutex AND write thread
ColumnFamilyData* ColumnFamilySet::CreateColumnFamily(
    const std::string& name, uint32_t id, Version* dummy_versions,
    const ColumnFamilyOptions& options) {
  assert(column_families_.find(name) == column_families_.end());
  ColumnFamilyData* new_cfd =
      new ColumnFamilyData(id, name, dummy_versions, table_cache_,
                           write_buffer_, options, db_options_,
                           env_options_, this);
  column_families_.insert({name, id});
  column_family_data_.insert({id, new_cfd});
  max_column_family_ = std::max(max_column_family_, id);
  // add to linked list
  new_cfd->next_ = dummy_cfd_;
  auto prev = dummy_cfd_->prev_;
  new_cfd->prev_ = prev;
  prev->next_ = new_cfd;
  dummy_cfd_->prev_ = new_cfd;
  if (id == 0) {
    default_cfd_cache_ = new_cfd;
  }
  return new_cfd;
}

// REQUIRES: DB mutex held
void ColumnFamilySet::FreeDeadColumnFamilies() {
  autovector<ColumnFamilyData*> to_delete;
  for (auto cfd = dummy_cfd_->next_; cfd != dummy_cfd_; cfd = cfd->next_) {
    if (cfd->refs_.load(std::memory_order_relaxed) == 0) {
      to_delete.push_back(cfd);
    }
  }
  for (auto cfd : to_delete) {
    // this is very rare, so it's not a problem that we do it under a mutex
    delete cfd;
  }
}

// under a DB mutex AND from a write thread
void ColumnFamilySet::RemoveColumnFamily(ColumnFamilyData* cfd) {
  auto cfd_iter = column_family_data_.find(cfd->GetID());
  assert(cfd_iter != column_family_data_.end());
  column_family_data_.erase(cfd_iter);
  column_families_.erase(cfd->GetName());
}

// under a DB mutex OR from a write thread
bool ColumnFamilyMemTablesImpl::Seek(uint32_t column_family_id) {
  if (column_family_id == 0) {
    // optimization for common case
    current_ = column_family_set_->GetDefault();
  } else {
    current_ = column_family_set_->GetColumnFamily(column_family_id);
  }
  handle_.SetCFD(current_);
  return current_ != nullptr;
}

uint64_t ColumnFamilyMemTablesImpl::GetLogNumber() const {
  assert(current_ != nullptr);
  return current_->GetLogNumber();
}

MemTable* ColumnFamilyMemTablesImpl::GetMemTable() const {
  assert(current_ != nullptr);
  return current_->mem();
}

ColumnFamilyHandle* ColumnFamilyMemTablesImpl::GetColumnFamilyHandle() {
  assert(current_ != nullptr);
  return &handle_;
}

void ColumnFamilyMemTablesImpl::CheckMemtableFull() {
  if (current_ != nullptr && current_->mem()->ShouldScheduleFlush()) {
    flush_scheduler_->ScheduleFlush(current_);
    current_->mem()->MarkFlushScheduled();
  }
}

uint32_t GetColumnFamilyID(ColumnFamilyHandle* column_family) {
  uint32_t column_family_id = 0;
  if (column_family != nullptr) {
    auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
    column_family_id = cfh->GetID();
  }
  return column_family_id;
}

const Comparator* GetColumnFamilyUserComparator(
    ColumnFamilyHandle* column_family) {
  if (column_family != nullptr) {
    auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
    return cfh->user_comparator();
  }
  return nullptr;
}

}  // namespace rocksdb
