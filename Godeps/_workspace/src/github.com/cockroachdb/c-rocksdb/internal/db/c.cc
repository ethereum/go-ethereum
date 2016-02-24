//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#ifndef ROCKSDB_LITE

#include "rocksdb/c.h"

#include <stdlib.h>
#include "port/port.h"
#include "rocksdb/cache.h"
#include "rocksdb/compaction_filter.h"
#include "rocksdb/comparator.h"
#include "rocksdb/convenience.h"
#include "rocksdb/db.h"
#include "rocksdb/env.h"
#include "rocksdb/filter_policy.h"
#include "rocksdb/iterator.h"
#include "rocksdb/merge_operator.h"
#include "rocksdb/options.h"
#include "rocksdb/status.h"
#include "rocksdb/write_batch.h"
#include "rocksdb/memtablerep.h"
#include "rocksdb/universal_compaction.h"
#include "rocksdb/statistics.h"
#include "rocksdb/slice_transform.h"
#include "rocksdb/table.h"
#include "rocksdb/utilities/backupable_db.h"
#include "utilities/merge_operators.h"

using rocksdb::Cache;
using rocksdb::ColumnFamilyDescriptor;
using rocksdb::ColumnFamilyHandle;
using rocksdb::ColumnFamilyOptions;
using rocksdb::CompactionFilter;
using rocksdb::CompactionFilterFactory;
using rocksdb::CompactionFilterContext;
using rocksdb::CompactionOptionsFIFO;
using rocksdb::Comparator;
using rocksdb::CompressionType;
using rocksdb::DB;
using rocksdb::DBOptions;
using rocksdb::Env;
using rocksdb::InfoLogLevel;
using rocksdb::FileLock;
using rocksdb::FilterPolicy;
using rocksdb::FlushOptions;
using rocksdb::Iterator;
using rocksdb::Logger;
using rocksdb::MergeOperator;
using rocksdb::MergeOperators;
using rocksdb::NewBloomFilterPolicy;
using rocksdb::NewLRUCache;
using rocksdb::Options;
using rocksdb::BlockBasedTableOptions;
using rocksdb::CuckooTableOptions;
using rocksdb::RandomAccessFile;
using rocksdb::Range;
using rocksdb::ReadOptions;
using rocksdb::SequentialFile;
using rocksdb::Slice;
using rocksdb::SliceParts;
using rocksdb::SliceTransform;
using rocksdb::Snapshot;
using rocksdb::Status;
using rocksdb::WritableFile;
using rocksdb::WriteBatch;
using rocksdb::WriteOptions;
using rocksdb::LiveFileMetaData;
using rocksdb::BackupEngine;
using rocksdb::BackupableDBOptions;
using rocksdb::BackupInfo;
using rocksdb::RestoreOptions;
using rocksdb::CompactRangeOptions;

using std::shared_ptr;

extern "C" {

struct rocksdb_t                 { DB*               rep; };
struct rocksdb_backup_engine_t   { BackupEngine*     rep; };
struct rocksdb_backup_engine_info_t { std::vector<BackupInfo> rep; };
struct rocksdb_restore_options_t { RestoreOptions rep; };
struct rocksdb_iterator_t        { Iterator*         rep; };
struct rocksdb_writebatch_t      { WriteBatch        rep; };
struct rocksdb_snapshot_t        { const Snapshot*   rep; };
struct rocksdb_flushoptions_t    { FlushOptions      rep; };
struct rocksdb_fifo_compaction_options_t { CompactionOptionsFIFO rep; };
struct rocksdb_readoptions_t {
   ReadOptions rep;
   Slice upper_bound; // stack variable to set pointer to in ReadOptions
};
struct rocksdb_writeoptions_t    { WriteOptions      rep; };
struct rocksdb_options_t         { Options           rep; };
struct rocksdb_block_based_table_options_t  { BlockBasedTableOptions rep; };
struct rocksdb_cuckoo_table_options_t  { CuckooTableOptions rep; };
struct rocksdb_seqfile_t         { SequentialFile*   rep; };
struct rocksdb_randomfile_t      { RandomAccessFile* rep; };
struct rocksdb_writablefile_t    { WritableFile*     rep; };
struct rocksdb_filelock_t        { FileLock*         rep; };
struct rocksdb_logger_t          { shared_ptr<Logger>  rep; };
struct rocksdb_cache_t           { shared_ptr<Cache>   rep; };
struct rocksdb_livefiles_t       { std::vector<LiveFileMetaData> rep; };
struct rocksdb_column_family_handle_t  { ColumnFamilyHandle* rep; };

struct rocksdb_compactionfiltercontext_t {
  CompactionFilter::Context rep;
};

struct rocksdb_compactionfilter_t : public CompactionFilter {
  void* state_;
  void (*destructor_)(void*);
  unsigned char (*filter_)(
      void*,
      int level,
      const char* key, size_t key_length,
      const char* existing_value, size_t value_length,
      char** new_value, size_t *new_value_length,
      unsigned char* value_changed);
  const char* (*name_)(void*);

  virtual ~rocksdb_compactionfilter_t() {
    (*destructor_)(state_);
  }

  virtual bool Filter(int level, const Slice& key, const Slice& existing_value,
                      std::string* new_value,
                      bool* value_changed) const override {
    char* c_new_value = nullptr;
    size_t new_value_length = 0;
    unsigned char c_value_changed = 0;
    unsigned char result = (*filter_)(
        state_,
        level,
        key.data(), key.size(),
        existing_value.data(), existing_value.size(),
        &c_new_value, &new_value_length, &c_value_changed);
    if (c_value_changed) {
      new_value->assign(c_new_value, new_value_length);
      *value_changed = true;
    }
    return result;
  }

  virtual const char* Name() const override { return (*name_)(state_); }
};

struct rocksdb_compactionfilterfactory_t : public CompactionFilterFactory {
  void* state_;
  void (*destructor_)(void*);
  rocksdb_compactionfilter_t* (*create_compaction_filter_)(
      void*, rocksdb_compactionfiltercontext_t* context);
  const char* (*name_)(void*);

  virtual ~rocksdb_compactionfilterfactory_t() { (*destructor_)(state_); }

  virtual std::unique_ptr<CompactionFilter> CreateCompactionFilter(
      const CompactionFilter::Context& context) override {
    rocksdb_compactionfiltercontext_t ccontext;
    ccontext.rep = context;
    CompactionFilter* cf = (*create_compaction_filter_)(state_, &ccontext);
    return std::unique_ptr<CompactionFilter>(cf);
  }

  virtual const char* Name() const override { return (*name_)(state_); }
};

struct rocksdb_comparator_t : public Comparator {
  void* state_;
  void (*destructor_)(void*);
  int (*compare_)(
      void*,
      const char* a, size_t alen,
      const char* b, size_t blen);
  const char* (*name_)(void*);

  virtual ~rocksdb_comparator_t() {
    (*destructor_)(state_);
  }

  virtual int Compare(const Slice& a, const Slice& b) const override {
    return (*compare_)(state_, a.data(), a.size(), b.data(), b.size());
  }

  virtual const char* Name() const override { return (*name_)(state_); }

  // No-ops since the C binding does not support key shortening methods.
  virtual void FindShortestSeparator(std::string*,
                                     const Slice&) const override {}
  virtual void FindShortSuccessor(std::string* key) const override {}
};

struct rocksdb_filterpolicy_t : public FilterPolicy {
  void* state_;
  void (*destructor_)(void*);
  const char* (*name_)(void*);
  char* (*create_)(
      void*,
      const char* const* key_array, const size_t* key_length_array,
      int num_keys,
      size_t* filter_length);
  unsigned char (*key_match_)(
      void*,
      const char* key, size_t length,
      const char* filter, size_t filter_length);
  void (*delete_filter_)(
      void*,
      const char* filter, size_t filter_length);

  virtual ~rocksdb_filterpolicy_t() {
    (*destructor_)(state_);
  }

  virtual const char* Name() const override { return (*name_)(state_); }

  virtual void CreateFilter(const Slice* keys, int n,
                            std::string* dst) const override {
    std::vector<const char*> key_pointers(n);
    std::vector<size_t> key_sizes(n);
    for (int i = 0; i < n; i++) {
      key_pointers[i] = keys[i].data();
      key_sizes[i] = keys[i].size();
    }
    size_t len;
    char* filter = (*create_)(state_, &key_pointers[0], &key_sizes[0], n, &len);
    dst->append(filter, len);

    if (delete_filter_ != nullptr) {
      (*delete_filter_)(state_, filter, len);
    } else {
      free(filter);
    }
  }

  virtual bool KeyMayMatch(const Slice& key,
                           const Slice& filter) const override {
    return (*key_match_)(state_, key.data(), key.size(),
                         filter.data(), filter.size());
  }
};

struct rocksdb_mergeoperator_t : public MergeOperator {
  void* state_;
  void (*destructor_)(void*);
  const char* (*name_)(void*);
  char* (*full_merge_)(
      void*,
      const char* key, size_t key_length,
      const char* existing_value, size_t existing_value_length,
      const char* const* operands_list, const size_t* operands_list_length,
      int num_operands,
      unsigned char* success, size_t* new_value_length);
  char* (*partial_merge_)(void*, const char* key, size_t key_length,
                          const char* const* operands_list,
                          const size_t* operands_list_length, int num_operands,
                          unsigned char* success, size_t* new_value_length);
  void (*delete_value_)(
      void*,
      const char* value, size_t value_length);

  virtual ~rocksdb_mergeoperator_t() {
    (*destructor_)(state_);
  }

  virtual const char* Name() const override { return (*name_)(state_); }

  virtual bool FullMerge(const Slice& key, const Slice* existing_value,
                         const std::deque<std::string>& operand_list,
                         std::string* new_value,
                         Logger* logger) const override {
    size_t n = operand_list.size();
    std::vector<const char*> operand_pointers(n);
    std::vector<size_t> operand_sizes(n);
    for (size_t i = 0; i < n; i++) {
      Slice operand(operand_list[i]);
      operand_pointers[i] = operand.data();
      operand_sizes[i] = operand.size();
    }

    const char* existing_value_data = nullptr;
    size_t existing_value_len = 0;
    if (existing_value != nullptr) {
      existing_value_data = existing_value->data();
      existing_value_len = existing_value->size();
    }

    unsigned char success;
    size_t new_value_len;
    char* tmp_new_value = (*full_merge_)(
        state_, key.data(), key.size(), existing_value_data, existing_value_len,
        &operand_pointers[0], &operand_sizes[0], static_cast<int>(n), &success,
        &new_value_len);
    new_value->assign(tmp_new_value, new_value_len);

    if (delete_value_ != nullptr) {
      (*delete_value_)(state_, tmp_new_value, new_value_len);
    } else {
      free(tmp_new_value);
    }

    return success;
  }

  virtual bool PartialMergeMulti(const Slice& key,
                                 const std::deque<Slice>& operand_list,
                                 std::string* new_value,
                                 Logger* logger) const override {
    size_t operand_count = operand_list.size();
    std::vector<const char*> operand_pointers(operand_count);
    std::vector<size_t> operand_sizes(operand_count);
    for (size_t i = 0; i < operand_count; ++i) {
      Slice operand(operand_list[i]);
      operand_pointers[i] = operand.data();
      operand_sizes[i] = operand.size();
    }

    unsigned char success;
    size_t new_value_len;
    char* tmp_new_value = (*partial_merge_)(
        state_, key.data(), key.size(), &operand_pointers[0], &operand_sizes[0],
        static_cast<int>(operand_count), &success, &new_value_len);
    new_value->assign(tmp_new_value, new_value_len);

    if (delete_value_ != nullptr) {
      (*delete_value_)(state_, tmp_new_value, new_value_len);
    } else {
      free(tmp_new_value);
    }

    return success;
  }
};

struct rocksdb_env_t {
  Env* rep;
  bool is_default;
};

struct rocksdb_slicetransform_t : public SliceTransform {
  void* state_;
  void (*destructor_)(void*);
  const char* (*name_)(void*);
  char* (*transform_)(
      void*,
      const char* key, size_t length,
      size_t* dst_length);
  unsigned char (*in_domain_)(
      void*,
      const char* key, size_t length);
  unsigned char (*in_range_)(
      void*,
      const char* key, size_t length);

  virtual ~rocksdb_slicetransform_t() {
    (*destructor_)(state_);
  }

  virtual const char* Name() const override { return (*name_)(state_); }

  virtual Slice Transform(const Slice& src) const override {
    size_t len;
    char* dst = (*transform_)(state_, src.data(), src.size(), &len);
    return Slice(dst, len);
  }

  virtual bool InDomain(const Slice& src) const override {
    return (*in_domain_)(state_, src.data(), src.size());
  }

  virtual bool InRange(const Slice& src) const override {
    return (*in_range_)(state_, src.data(), src.size());
  }
};

struct rocksdb_universal_compaction_options_t {
  rocksdb::CompactionOptionsUniversal *rep;
};

static bool SaveError(char** errptr, const Status& s) {
  assert(errptr != nullptr);
  if (s.ok()) {
    return false;
  } else if (*errptr == nullptr) {
    *errptr = strdup(s.ToString().c_str());
  } else {
    // TODO(sanjay): Merge with existing error?
    // This is a bug if *errptr is not created by malloc()
    free(*errptr);
    *errptr = strdup(s.ToString().c_str());
  }
  return true;
}

static char* CopyString(const std::string& str) {
  char* result = reinterpret_cast<char*>(malloc(sizeof(char) * str.size()));
  memcpy(result, str.data(), sizeof(char) * str.size());
  return result;
}

rocksdb_t* rocksdb_open(
    const rocksdb_options_t* options,
    const char* name,
    char** errptr) {
  DB* db;
  if (SaveError(errptr, DB::Open(options->rep, std::string(name), &db))) {
    return nullptr;
  }
  rocksdb_t* result = new rocksdb_t;
  result->rep = db;
  return result;
}

rocksdb_t* rocksdb_open_for_read_only(
    const rocksdb_options_t* options,
    const char* name,
    unsigned char error_if_log_file_exist,
    char** errptr) {
  DB* db;
  if (SaveError(errptr, DB::OpenForReadOnly(options->rep, std::string(name), &db, error_if_log_file_exist))) {
    return nullptr;
  }
  rocksdb_t* result = new rocksdb_t;
  result->rep = db;
  return result;
}

rocksdb_backup_engine_t* rocksdb_backup_engine_open(
    const rocksdb_options_t* options, const char* path, char** errptr) {
  BackupEngine* be;
  if (SaveError(errptr, BackupEngine::Open(options->rep.env,
                                           BackupableDBOptions(path), &be))) {
    return nullptr;
  }
  rocksdb_backup_engine_t* result = new rocksdb_backup_engine_t;
  result->rep = be;
  return result;
}

void rocksdb_backup_engine_create_new_backup(rocksdb_backup_engine_t* be,
                                             rocksdb_t* db, char** errptr) {
  SaveError(errptr, be->rep->CreateNewBackup(db->rep));
}

rocksdb_restore_options_t* rocksdb_restore_options_create() {
  return new rocksdb_restore_options_t;
}

void rocksdb_restore_options_destroy(rocksdb_restore_options_t* opt) {
  delete opt;
}

void rocksdb_restore_options_set_keep_log_files(rocksdb_restore_options_t* opt,
                                                int v) {
  opt->rep.keep_log_files = v;
}

void rocksdb_backup_engine_restore_db_from_latest_backup(
    rocksdb_backup_engine_t* be, const char* db_dir, const char* wal_dir,
    const rocksdb_restore_options_t* restore_options, char** errptr) {
  SaveError(errptr, be->rep->RestoreDBFromLatestBackup(std::string(db_dir),
                                                       std::string(wal_dir),
                                                       restore_options->rep));
}

const rocksdb_backup_engine_info_t* rocksdb_backup_engine_get_backup_info(
    rocksdb_backup_engine_t* be) {
  rocksdb_backup_engine_info_t* result = new rocksdb_backup_engine_info_t;
  be->rep->GetBackupInfo(&result->rep);
  return result;
}

int rocksdb_backup_engine_info_count(const rocksdb_backup_engine_info_t* info) {
  return static_cast<int>(info->rep.size());
}

int64_t rocksdb_backup_engine_info_timestamp(
    const rocksdb_backup_engine_info_t* info, int index) {
  return info->rep[index].timestamp;
}

uint32_t rocksdb_backup_engine_info_backup_id(
    const rocksdb_backup_engine_info_t* info, int index) {
  return info->rep[index].backup_id;
}

uint64_t rocksdb_backup_engine_info_size(
    const rocksdb_backup_engine_info_t* info, int index) {
  return info->rep[index].size;
}

uint32_t rocksdb_backup_engine_info_number_files(
    const rocksdb_backup_engine_info_t* info, int index) {
  return info->rep[index].number_files;
}

void rocksdb_backup_engine_info_destroy(
    const rocksdb_backup_engine_info_t* info) {
  delete info;
}

void rocksdb_backup_engine_close(rocksdb_backup_engine_t* be) {
  delete be->rep;
  delete be;
}

void rocksdb_close(rocksdb_t* db) {
  delete db->rep;
  delete db;
}

void rocksdb_options_set_uint64add_merge_operator(rocksdb_options_t* opt) {
  opt->rep.merge_operator = rocksdb::MergeOperators::CreateUInt64AddOperator();
}

rocksdb_t* rocksdb_open_column_families(
    const rocksdb_options_t* db_options,
    const char* name,
    int num_column_families,
    const char** column_family_names,
    const rocksdb_options_t** column_family_options,
    rocksdb_column_family_handle_t** column_family_handles,
    char** errptr) {
  std::vector<ColumnFamilyDescriptor> column_families;
  for (int i = 0; i < num_column_families; i++) {
    column_families.push_back(ColumnFamilyDescriptor(
        std::string(column_family_names[i]),
        ColumnFamilyOptions(column_family_options[i]->rep)));
  }

  DB* db;
  std::vector<ColumnFamilyHandle*> handles;
  if (SaveError(errptr, DB::Open(DBOptions(db_options->rep),
          std::string(name), column_families, &handles, &db))) {
    return nullptr;
  }

  for (size_t i = 0; i < handles.size(); i++) {
    rocksdb_column_family_handle_t* c_handle = new rocksdb_column_family_handle_t;
    c_handle->rep = handles[i];
    column_family_handles[i] = c_handle;
  }
  rocksdb_t* result = new rocksdb_t;
  result->rep = db;
  return result;
}

rocksdb_t* rocksdb_open_for_read_only_column_families(
    const rocksdb_options_t* db_options,
    const char* name,
    int num_column_families,
    const char** column_family_names,
    const rocksdb_options_t** column_family_options,
    rocksdb_column_family_handle_t** column_family_handles,
    unsigned char error_if_log_file_exist,
    char** errptr) {
  std::vector<ColumnFamilyDescriptor> column_families;
  for (int i = 0; i < num_column_families; i++) {
    column_families.push_back(ColumnFamilyDescriptor(
        std::string(column_family_names[i]),
        ColumnFamilyOptions(column_family_options[i]->rep)));
  }

  DB* db;
  std::vector<ColumnFamilyHandle*> handles;
  if (SaveError(errptr, DB::OpenForReadOnly(DBOptions(db_options->rep),
          std::string(name), column_families, &handles, &db, error_if_log_file_exist))) {
    return nullptr;
  }

  for (size_t i = 0; i < handles.size(); i++) {
    rocksdb_column_family_handle_t* c_handle = new rocksdb_column_family_handle_t;
    c_handle->rep = handles[i];
    column_family_handles[i] = c_handle;
  }
  rocksdb_t* result = new rocksdb_t;
  result->rep = db;
  return result;
}

char** rocksdb_list_column_families(
    const rocksdb_options_t* options,
    const char* name,
    size_t* lencfs,
    char** errptr) {
  std::vector<std::string> fams;
  SaveError(errptr,
      DB::ListColumnFamilies(DBOptions(options->rep),
        std::string(name), &fams));

  *lencfs = fams.size();
  char** column_families = static_cast<char**>(malloc(sizeof(char*) * fams.size()));
  for (size_t i = 0; i < fams.size(); i++) {
    column_families[i] = strdup(fams[i].c_str());
  }
  return column_families;
}

void rocksdb_list_column_families_destroy(char** list, size_t len) {
  for (size_t i = 0; i < len; ++i) {
    free(list[i]);
  }
  free(list);
}

rocksdb_column_family_handle_t* rocksdb_create_column_family(
    rocksdb_t* db,
    const rocksdb_options_t* column_family_options,
    const char* column_family_name,
    char** errptr) {
  rocksdb_column_family_handle_t* handle = new rocksdb_column_family_handle_t;
  SaveError(errptr,
      db->rep->CreateColumnFamily(ColumnFamilyOptions(column_family_options->rep),
        std::string(column_family_name), &(handle->rep)));
  return handle;
}

void rocksdb_drop_column_family(
    rocksdb_t* db,
    rocksdb_column_family_handle_t* handle,
    char** errptr) {
  SaveError(errptr, db->rep->DropColumnFamily(handle->rep));
}

void rocksdb_column_family_handle_destroy(rocksdb_column_family_handle_t* handle) {
  delete handle->rep;
  delete handle;
}

void rocksdb_put(
    rocksdb_t* db,
    const rocksdb_writeoptions_t* options,
    const char* key, size_t keylen,
    const char* val, size_t vallen,
    char** errptr) {
  SaveError(errptr,
            db->rep->Put(options->rep, Slice(key, keylen), Slice(val, vallen)));
}

void rocksdb_put_cf(
    rocksdb_t* db,
    const rocksdb_writeoptions_t* options,
    rocksdb_column_family_handle_t* column_family,
    const char* key, size_t keylen,
    const char* val, size_t vallen,
    char** errptr) {
  SaveError(errptr,
            db->rep->Put(options->rep, column_family->rep,
              Slice(key, keylen), Slice(val, vallen)));
}

void rocksdb_delete(
    rocksdb_t* db,
    const rocksdb_writeoptions_t* options,
    const char* key, size_t keylen,
    char** errptr) {
  SaveError(errptr, db->rep->Delete(options->rep, Slice(key, keylen)));
}

void rocksdb_delete_cf(
    rocksdb_t* db,
    const rocksdb_writeoptions_t* options,
    rocksdb_column_family_handle_t* column_family,
    const char* key, size_t keylen,
    char** errptr) {
  SaveError(errptr, db->rep->Delete(options->rep, column_family->rep,
        Slice(key, keylen)));
}

void rocksdb_merge(
    rocksdb_t* db,
    const rocksdb_writeoptions_t* options,
    const char* key, size_t keylen,
    const char* val, size_t vallen,
    char** errptr) {
  SaveError(errptr,
            db->rep->Merge(options->rep, Slice(key, keylen), Slice(val, vallen)));
}

void rocksdb_merge_cf(
    rocksdb_t* db,
    const rocksdb_writeoptions_t* options,
    rocksdb_column_family_handle_t* column_family,
    const char* key, size_t keylen,
    const char* val, size_t vallen,
    char** errptr) {
  SaveError(errptr,
            db->rep->Merge(options->rep, column_family->rep,
              Slice(key, keylen), Slice(val, vallen)));
}

void rocksdb_write(
    rocksdb_t* db,
    const rocksdb_writeoptions_t* options,
    rocksdb_writebatch_t* batch,
    char** errptr) {
  SaveError(errptr, db->rep->Write(options->rep, &batch->rep));
}

char* rocksdb_get(
    rocksdb_t* db,
    const rocksdb_readoptions_t* options,
    const char* key, size_t keylen,
    size_t* vallen,
    char** errptr) {
  char* result = nullptr;
  std::string tmp;
  Status s = db->rep->Get(options->rep, Slice(key, keylen), &tmp);
  if (s.ok()) {
    *vallen = tmp.size();
    result = CopyString(tmp);
  } else {
    *vallen = 0;
    if (!s.IsNotFound()) {
      SaveError(errptr, s);
    }
  }
  return result;
}

char* rocksdb_get_cf(
    rocksdb_t* db,
    const rocksdb_readoptions_t* options,
    rocksdb_column_family_handle_t* column_family,
    const char* key, size_t keylen,
    size_t* vallen,
    char** errptr) {
  char* result = nullptr;
  std::string tmp;
  Status s = db->rep->Get(options->rep, column_family->rep,
      Slice(key, keylen), &tmp);
  if (s.ok()) {
    *vallen = tmp.size();
    result = CopyString(tmp);
  } else {
    *vallen = 0;
    if (!s.IsNotFound()) {
      SaveError(errptr, s);
    }
  }
  return result;
}

void rocksdb_multi_get(
    rocksdb_t* db,
    const rocksdb_readoptions_t* options,
    size_t num_keys, const char* const* keys_list,
    const size_t* keys_list_sizes,
    char** values_list, size_t* values_list_sizes,
    char** errs) {
  std::vector<Slice> keys(num_keys);
  for (size_t i = 0; i < num_keys; i++) {
    keys[i] = Slice(keys_list[i], keys_list_sizes[i]);
  }
  std::vector<std::string> values(num_keys);
  std::vector<Status> statuses = db->rep->MultiGet(options->rep, keys, &values);
  for (size_t i = 0; i < num_keys; i++) {
    if (statuses[i].ok()) {
      values_list[i] = CopyString(values[i]);
      values_list_sizes[i] = values[i].size();
      errs[i] = nullptr;
    } else {
      values_list[i] = nullptr;
      values_list_sizes[i] = 0;
      if (!statuses[i].IsNotFound()) {
        errs[i] = strdup(statuses[i].ToString().c_str());
      } else {
        errs[i] = nullptr;
      }
    }
  }
}

void rocksdb_multi_get_cf(
    rocksdb_t* db,
    const rocksdb_readoptions_t* options,
    const rocksdb_column_family_handle_t* const* column_families,
    size_t num_keys, const char* const* keys_list,
    const size_t* keys_list_sizes,
    char** values_list, size_t* values_list_sizes,
    char** errs) {
  std::vector<Slice> keys(num_keys);
  std::vector<ColumnFamilyHandle*> cfs(num_keys);
  for (size_t i = 0; i < num_keys; i++) {
    keys[i] = Slice(keys_list[i], keys_list_sizes[i]);
    cfs[i] = column_families[i]->rep;
  }
  std::vector<std::string> values(num_keys);
  std::vector<Status> statuses = db->rep->MultiGet(options->rep, cfs, keys, &values);
  for (size_t i = 0; i < num_keys; i++) {
    if (statuses[i].ok()) {
      values_list[i] = CopyString(values[i]);
      values_list_sizes[i] = values[i].size();
      errs[i] = nullptr;
    } else {
      values_list[i] = nullptr;
      values_list_sizes[i] = 0;
      if (!statuses[i].IsNotFound()) {
        errs[i] = strdup(statuses[i].ToString().c_str());
      } else {
        errs[i] = nullptr;
      }
    }
  }
}

rocksdb_iterator_t* rocksdb_create_iterator(
    rocksdb_t* db,
    const rocksdb_readoptions_t* options) {
  rocksdb_iterator_t* result = new rocksdb_iterator_t;
  result->rep = db->rep->NewIterator(options->rep);
  return result;
}

rocksdb_iterator_t* rocksdb_create_iterator_cf(
    rocksdb_t* db,
    const rocksdb_readoptions_t* options,
    rocksdb_column_family_handle_t* column_family) {
  rocksdb_iterator_t* result = new rocksdb_iterator_t;
  result->rep = db->rep->NewIterator(options->rep, column_family->rep);
  return result;
}

const rocksdb_snapshot_t* rocksdb_create_snapshot(
    rocksdb_t* db) {
  rocksdb_snapshot_t* result = new rocksdb_snapshot_t;
  result->rep = db->rep->GetSnapshot();
  return result;
}

void rocksdb_release_snapshot(
    rocksdb_t* db,
    const rocksdb_snapshot_t* snapshot) {
  db->rep->ReleaseSnapshot(snapshot->rep);
  delete snapshot;
}

char* rocksdb_property_value(
    rocksdb_t* db,
    const char* propname) {
  std::string tmp;
  if (db->rep->GetProperty(Slice(propname), &tmp)) {
    // We use strdup() since we expect human readable output.
    return strdup(tmp.c_str());
  } else {
    return nullptr;
  }
}

char* rocksdb_property_value_cf(
    rocksdb_t* db,
    rocksdb_column_family_handle_t* column_family,
    const char* propname) {
  std::string tmp;
  if (db->rep->GetProperty(column_family->rep, Slice(propname), &tmp)) {
    // We use strdup() since we expect human readable output.
    return strdup(tmp.c_str());
  } else {
    return nullptr;
  }
}

void rocksdb_approximate_sizes(
    rocksdb_t* db,
    int num_ranges,
    const char* const* range_start_key, const size_t* range_start_key_len,
    const char* const* range_limit_key, const size_t* range_limit_key_len,
    uint64_t* sizes) {
  Range* ranges = new Range[num_ranges];
  for (int i = 0; i < num_ranges; i++) {
    ranges[i].start = Slice(range_start_key[i], range_start_key_len[i]);
    ranges[i].limit = Slice(range_limit_key[i], range_limit_key_len[i]);
  }
  db->rep->GetApproximateSizes(ranges, num_ranges, sizes);
  delete[] ranges;
}

void rocksdb_approximate_sizes_cf(
    rocksdb_t* db,
    rocksdb_column_family_handle_t* column_family,
    int num_ranges,
    const char* const* range_start_key, const size_t* range_start_key_len,
    const char* const* range_limit_key, const size_t* range_limit_key_len,
    uint64_t* sizes) {
  Range* ranges = new Range[num_ranges];
  for (int i = 0; i < num_ranges; i++) {
    ranges[i].start = Slice(range_start_key[i], range_start_key_len[i]);
    ranges[i].limit = Slice(range_limit_key[i], range_limit_key_len[i]);
  }
  db->rep->GetApproximateSizes(column_family->rep, ranges, num_ranges, sizes);
  delete[] ranges;
}

void rocksdb_delete_file(
    rocksdb_t* db,
    const char* name) {
  db->rep->DeleteFile(name);
}

const rocksdb_livefiles_t* rocksdb_livefiles(
    rocksdb_t* db) {
  rocksdb_livefiles_t* result = new rocksdb_livefiles_t;
  db->rep->GetLiveFilesMetaData(&result->rep);
  return result;
}

void rocksdb_compact_range(
    rocksdb_t* db,
    const char* start_key, size_t start_key_len,
    const char* limit_key, size_t limit_key_len) {
  Slice a, b;
  db->rep->CompactRange(
      CompactRangeOptions(),
      // Pass nullptr Slice if corresponding "const char*" is nullptr
      (start_key ? (a = Slice(start_key, start_key_len), &a) : nullptr),
      (limit_key ? (b = Slice(limit_key, limit_key_len), &b) : nullptr));
}

void rocksdb_compact_range_cf(
    rocksdb_t* db,
    rocksdb_column_family_handle_t* column_family,
    const char* start_key, size_t start_key_len,
    const char* limit_key, size_t limit_key_len) {
  Slice a, b;
  db->rep->CompactRange(
      CompactRangeOptions(), column_family->rep,
      // Pass nullptr Slice if corresponding "const char*" is nullptr
      (start_key ? (a = Slice(start_key, start_key_len), &a) : nullptr),
      (limit_key ? (b = Slice(limit_key, limit_key_len), &b) : nullptr));
}

void rocksdb_flush(
    rocksdb_t* db,
    const rocksdb_flushoptions_t* options,
    char** errptr) {
  SaveError(errptr, db->rep->Flush(options->rep));
}

void rocksdb_disable_file_deletions(
    rocksdb_t* db,
    char** errptr) {
  SaveError(errptr, db->rep->DisableFileDeletions());
}

void rocksdb_enable_file_deletions(
    rocksdb_t* db,
    unsigned char force,
    char** errptr) {
  SaveError(errptr, db->rep->EnableFileDeletions(force));
}

void rocksdb_destroy_db(
    const rocksdb_options_t* options,
    const char* name,
    char** errptr) {
  SaveError(errptr, DestroyDB(name, options->rep));
}

void rocksdb_repair_db(
    const rocksdb_options_t* options,
    const char* name,
    char** errptr) {
  SaveError(errptr, RepairDB(name, options->rep));
}

void rocksdb_iter_destroy(rocksdb_iterator_t* iter) {
  delete iter->rep;
  delete iter;
}

unsigned char rocksdb_iter_valid(const rocksdb_iterator_t* iter) {
  return iter->rep->Valid();
}

void rocksdb_iter_seek_to_first(rocksdb_iterator_t* iter) {
  iter->rep->SeekToFirst();
}

void rocksdb_iter_seek_to_last(rocksdb_iterator_t* iter) {
  iter->rep->SeekToLast();
}

void rocksdb_iter_seek(rocksdb_iterator_t* iter, const char* k, size_t klen) {
  iter->rep->Seek(Slice(k, klen));
}

void rocksdb_iter_next(rocksdb_iterator_t* iter) {
  iter->rep->Next();
}

void rocksdb_iter_prev(rocksdb_iterator_t* iter) {
  iter->rep->Prev();
}

const char* rocksdb_iter_key(const rocksdb_iterator_t* iter, size_t* klen) {
  Slice s = iter->rep->key();
  *klen = s.size();
  return s.data();
}

const char* rocksdb_iter_value(const rocksdb_iterator_t* iter, size_t* vlen) {
  Slice s = iter->rep->value();
  *vlen = s.size();
  return s.data();
}

void rocksdb_iter_get_error(const rocksdb_iterator_t* iter, char** errptr) {
  SaveError(errptr, iter->rep->status());
}

rocksdb_writebatch_t* rocksdb_writebatch_create() {
  return new rocksdb_writebatch_t;
}

rocksdb_writebatch_t* rocksdb_writebatch_create_from(const char* rep,
                                                     size_t size) {
  rocksdb_writebatch_t* b = new rocksdb_writebatch_t;
  b->rep = WriteBatch(std::string(rep, size));
  return b;
}

void rocksdb_writebatch_destroy(rocksdb_writebatch_t* b) {
  delete b;
}

void rocksdb_writebatch_clear(rocksdb_writebatch_t* b) {
  b->rep.Clear();
}

int rocksdb_writebatch_count(rocksdb_writebatch_t* b) {
  return b->rep.Count();
}

void rocksdb_writebatch_put(
    rocksdb_writebatch_t* b,
    const char* key, size_t klen,
    const char* val, size_t vlen) {
  b->rep.Put(Slice(key, klen), Slice(val, vlen));
}

void rocksdb_writebatch_put_cf(
    rocksdb_writebatch_t* b,
    rocksdb_column_family_handle_t* column_family,
    const char* key, size_t klen,
    const char* val, size_t vlen) {
  b->rep.Put(column_family->rep, Slice(key, klen), Slice(val, vlen));
}

void rocksdb_writebatch_putv(
    rocksdb_writebatch_t* b,
    int num_keys, const char* const* keys_list,
    const size_t* keys_list_sizes,
    int num_values, const char* const* values_list,
    const size_t* values_list_sizes) {
  std::vector<Slice> key_slices(num_keys);
  for (int i = 0; i < num_keys; i++) {
    key_slices[i] = Slice(keys_list[i], keys_list_sizes[i]);
  }
  std::vector<Slice> value_slices(num_values);
  for (int i = 0; i < num_values; i++) {
    value_slices[i] = Slice(values_list[i], values_list_sizes[i]);
  }
  b->rep.Put(SliceParts(key_slices.data(), num_keys),
             SliceParts(value_slices.data(), num_values));
}

void rocksdb_writebatch_putv_cf(
    rocksdb_writebatch_t* b,
    rocksdb_column_family_handle_t* column_family,
    int num_keys, const char* const* keys_list,
    const size_t* keys_list_sizes,
    int num_values, const char* const* values_list,
    const size_t* values_list_sizes) {
  std::vector<Slice> key_slices(num_keys);
  for (int i = 0; i < num_keys; i++) {
    key_slices[i] = Slice(keys_list[i], keys_list_sizes[i]);
  }
  std::vector<Slice> value_slices(num_values);
  for (int i = 0; i < num_values; i++) {
    value_slices[i] = Slice(values_list[i], values_list_sizes[i]);
  }
  b->rep.Put(column_family->rep, SliceParts(key_slices.data(), num_keys),
             SliceParts(value_slices.data(), num_values));
}

void rocksdb_writebatch_merge(
    rocksdb_writebatch_t* b,
    const char* key, size_t klen,
    const char* val, size_t vlen) {
  b->rep.Merge(Slice(key, klen), Slice(val, vlen));
}

void rocksdb_writebatch_merge_cf(
    rocksdb_writebatch_t* b,
    rocksdb_column_family_handle_t* column_family,
    const char* key, size_t klen,
    const char* val, size_t vlen) {
  b->rep.Merge(column_family->rep, Slice(key, klen), Slice(val, vlen));
}

void rocksdb_writebatch_mergev(
    rocksdb_writebatch_t* b,
    int num_keys, const char* const* keys_list,
    const size_t* keys_list_sizes,
    int num_values, const char* const* values_list,
    const size_t* values_list_sizes) {
  std::vector<Slice> key_slices(num_keys);
  for (int i = 0; i < num_keys; i++) {
    key_slices[i] = Slice(keys_list[i], keys_list_sizes[i]);
  }
  std::vector<Slice> value_slices(num_values);
  for (int i = 0; i < num_values; i++) {
    value_slices[i] = Slice(values_list[i], values_list_sizes[i]);
  }
  b->rep.Merge(SliceParts(key_slices.data(), num_keys),
               SliceParts(value_slices.data(), num_values));
}

void rocksdb_writebatch_mergev_cf(
    rocksdb_writebatch_t* b,
    rocksdb_column_family_handle_t* column_family,
    int num_keys, const char* const* keys_list,
    const size_t* keys_list_sizes,
    int num_values, const char* const* values_list,
    const size_t* values_list_sizes) {
  std::vector<Slice> key_slices(num_keys);
  for (int i = 0; i < num_keys; i++) {
    key_slices[i] = Slice(keys_list[i], keys_list_sizes[i]);
  }
  std::vector<Slice> value_slices(num_values);
  for (int i = 0; i < num_values; i++) {
    value_slices[i] = Slice(values_list[i], values_list_sizes[i]);
  }
  b->rep.Merge(column_family->rep, SliceParts(key_slices.data(), num_keys),
               SliceParts(value_slices.data(), num_values));
}

void rocksdb_writebatch_delete(
    rocksdb_writebatch_t* b,
    const char* key, size_t klen) {
  b->rep.Delete(Slice(key, klen));
}

void rocksdb_writebatch_delete_cf(
    rocksdb_writebatch_t* b,
    rocksdb_column_family_handle_t* column_family,
    const char* key, size_t klen) {
  b->rep.Delete(column_family->rep, Slice(key, klen));
}

void rocksdb_writebatch_deletev(
    rocksdb_writebatch_t* b,
    int num_keys, const char* const* keys_list,
    const size_t* keys_list_sizes) {
  std::vector<Slice> key_slices(num_keys);
  for (int i = 0; i < num_keys; i++) {
    key_slices[i] = Slice(keys_list[i], keys_list_sizes[i]);
  }
  b->rep.Delete(SliceParts(key_slices.data(), num_keys));
}

void rocksdb_writebatch_deletev_cf(
    rocksdb_writebatch_t* b,
    rocksdb_column_family_handle_t* column_family,
    int num_keys, const char* const* keys_list,
    const size_t* keys_list_sizes) {
  std::vector<Slice> key_slices(num_keys);
  for (int i = 0; i < num_keys; i++) {
    key_slices[i] = Slice(keys_list[i], keys_list_sizes[i]);
  }
  b->rep.Delete(column_family->rep, SliceParts(key_slices.data(), num_keys));
}

void rocksdb_writebatch_put_log_data(
    rocksdb_writebatch_t* b,
    const char* blob, size_t len) {
  b->rep.PutLogData(Slice(blob, len));
}

void rocksdb_writebatch_iterate(
    rocksdb_writebatch_t* b,
    void* state,
    void (*put)(void*, const char* k, size_t klen, const char* v, size_t vlen),
    void (*deleted)(void*, const char* k, size_t klen)) {
  class H : public WriteBatch::Handler {
   public:
    void* state_;
    void (*put_)(void*, const char* k, size_t klen, const char* v, size_t vlen);
    void (*deleted_)(void*, const char* k, size_t klen);
    virtual void Put(const Slice& key, const Slice& value) override {
      (*put_)(state_, key.data(), key.size(), value.data(), value.size());
    }
    virtual void Delete(const Slice& key) override {
      (*deleted_)(state_, key.data(), key.size());
    }
  };
  H handler;
  handler.state_ = state;
  handler.put_ = put;
  handler.deleted_ = deleted;
  b->rep.Iterate(&handler);
}

const char* rocksdb_writebatch_data(rocksdb_writebatch_t* b, size_t* size) {
  *size = b->rep.GetDataSize();
  return b->rep.Data().c_str();
}

rocksdb_block_based_table_options_t*
rocksdb_block_based_options_create() {
  return new rocksdb_block_based_table_options_t;
}

void rocksdb_block_based_options_destroy(
    rocksdb_block_based_table_options_t* options) {
  delete options;
}

void rocksdb_block_based_options_set_block_size(
    rocksdb_block_based_table_options_t* options, size_t block_size) {
  options->rep.block_size = block_size;
}

void rocksdb_block_based_options_set_block_size_deviation(
    rocksdb_block_based_table_options_t* options, int block_size_deviation) {
  options->rep.block_size_deviation = block_size_deviation;
}

void rocksdb_block_based_options_set_block_restart_interval(
    rocksdb_block_based_table_options_t* options, int block_restart_interval) {
  options->rep.block_restart_interval = block_restart_interval;
}

void rocksdb_block_based_options_set_filter_policy(
    rocksdb_block_based_table_options_t* options,
    rocksdb_filterpolicy_t* filter_policy) {
  options->rep.filter_policy.reset(filter_policy);
}

void rocksdb_block_based_options_set_no_block_cache(
    rocksdb_block_based_table_options_t* options,
    unsigned char no_block_cache) {
  options->rep.no_block_cache = no_block_cache;
}

void rocksdb_block_based_options_set_block_cache(
    rocksdb_block_based_table_options_t* options,
    rocksdb_cache_t* block_cache) {
  if (block_cache) {
    options->rep.block_cache = block_cache->rep;
  }
}

void rocksdb_block_based_options_set_block_cache_compressed(
    rocksdb_block_based_table_options_t* options,
    rocksdb_cache_t* block_cache_compressed) {
  if (block_cache_compressed) {
    options->rep.block_cache_compressed = block_cache_compressed->rep;
  }
}

void rocksdb_block_based_options_set_whole_key_filtering(
    rocksdb_block_based_table_options_t* options, unsigned char v) {
  options->rep.whole_key_filtering = v;
}

void rocksdb_block_based_options_set_format_version(
    rocksdb_block_based_table_options_t* options, int v) {
  options->rep.format_version = v;
}

void rocksdb_block_based_options_set_index_type(
    rocksdb_block_based_table_options_t* options, int v) {
  options->rep.index_type = static_cast<BlockBasedTableOptions::IndexType>(v);
}

void rocksdb_block_based_options_set_hash_index_allow_collision(
    rocksdb_block_based_table_options_t* options, unsigned char v) {
  options->rep.hash_index_allow_collision = v;
}

void rocksdb_block_based_options_set_cache_index_and_filter_blocks(
    rocksdb_block_based_table_options_t* options, unsigned char v) {
  options->rep.cache_index_and_filter_blocks = v;
}

void rocksdb_options_set_block_based_table_factory(
    rocksdb_options_t *opt,
    rocksdb_block_based_table_options_t* table_options) {
  if (table_options) {
    opt->rep.table_factory.reset(
        rocksdb::NewBlockBasedTableFactory(table_options->rep));
  }
}


rocksdb_cuckoo_table_options_t*
rocksdb_cuckoo_options_create() {
  return new rocksdb_cuckoo_table_options_t;
}

void rocksdb_cuckoo_options_destroy(
    rocksdb_cuckoo_table_options_t* options) {
  delete options;
}

void rocksdb_cuckoo_options_set_hash_ratio(
    rocksdb_cuckoo_table_options_t* options, double v) {
  options->rep.hash_table_ratio = v;
}

void rocksdb_cuckoo_options_set_max_search_depth(
    rocksdb_cuckoo_table_options_t* options, uint32_t v) {
  options->rep.max_search_depth = v;
}

void rocksdb_cuckoo_options_set_cuckoo_block_size(
    rocksdb_cuckoo_table_options_t* options, uint32_t v) {
  options->rep.cuckoo_block_size = v;
}

void rocksdb_cuckoo_options_set_identity_as_first_hash(
    rocksdb_cuckoo_table_options_t* options, unsigned char v) {
  options->rep.identity_as_first_hash = v;
}

void rocksdb_cuckoo_options_set_use_module_hash(
    rocksdb_cuckoo_table_options_t* options, unsigned char v) {
  options->rep.use_module_hash = v;
}

void rocksdb_options_set_cuckoo_table_factory(
    rocksdb_options_t *opt,
    rocksdb_cuckoo_table_options_t* table_options) {
  if (table_options) {
    opt->rep.table_factory.reset(
        rocksdb::NewCuckooTableFactory(table_options->rep));
  }
}


rocksdb_options_t* rocksdb_options_create() {
  return new rocksdb_options_t;
}

void rocksdb_options_destroy(rocksdb_options_t* options) {
  delete options;
}

void rocksdb_options_increase_parallelism(
    rocksdb_options_t* opt, int total_threads) {
  opt->rep.IncreaseParallelism(total_threads);
}

void rocksdb_options_optimize_for_point_lookup(
    rocksdb_options_t* opt, uint64_t block_cache_size_mb) {
  opt->rep.OptimizeForPointLookup(block_cache_size_mb);
}

void rocksdb_options_optimize_level_style_compaction(
    rocksdb_options_t* opt, uint64_t memtable_memory_budget) {
  opt->rep.OptimizeLevelStyleCompaction(memtable_memory_budget);
}

void rocksdb_options_optimize_universal_style_compaction(
    rocksdb_options_t* opt, uint64_t memtable_memory_budget) {
  opt->rep.OptimizeUniversalStyleCompaction(memtable_memory_budget);
}

void rocksdb_options_set_compaction_filter(
    rocksdb_options_t* opt,
    rocksdb_compactionfilter_t* filter) {
  opt->rep.compaction_filter = filter;
}

void rocksdb_options_set_compaction_filter_factory(
    rocksdb_options_t* opt, rocksdb_compactionfilterfactory_t* factory) {
  opt->rep.compaction_filter_factory =
      std::shared_ptr<CompactionFilterFactory>(factory);
}

void rocksdb_options_set_comparator(
    rocksdb_options_t* opt,
    rocksdb_comparator_t* cmp) {
  opt->rep.comparator = cmp;
}

void rocksdb_options_set_merge_operator(
    rocksdb_options_t* opt,
    rocksdb_mergeoperator_t* merge_operator) {
  opt->rep.merge_operator = std::shared_ptr<MergeOperator>(merge_operator);
}


void rocksdb_options_set_create_if_missing(
    rocksdb_options_t* opt, unsigned char v) {
  opt->rep.create_if_missing = v;
}

void rocksdb_options_set_create_missing_column_families(
    rocksdb_options_t* opt, unsigned char v) {
  opt->rep.create_missing_column_families = v;
}

void rocksdb_options_set_error_if_exists(
    rocksdb_options_t* opt, unsigned char v) {
  opt->rep.error_if_exists = v;
}

void rocksdb_options_set_paranoid_checks(
    rocksdb_options_t* opt, unsigned char v) {
  opt->rep.paranoid_checks = v;
}

void rocksdb_options_set_env(rocksdb_options_t* opt, rocksdb_env_t* env) {
  opt->rep.env = (env ? env->rep : nullptr);
}

void rocksdb_options_set_info_log(rocksdb_options_t* opt, rocksdb_logger_t* l) {
  if (l) {
    opt->rep.info_log = l->rep;
  }
}

void rocksdb_options_set_info_log_level(
    rocksdb_options_t* opt, int v) {
  opt->rep.info_log_level = static_cast<InfoLogLevel>(v);
}

void rocksdb_options_set_db_write_buffer_size(rocksdb_options_t* opt,
                                              size_t s) {
  opt->rep.db_write_buffer_size = s;
}

void rocksdb_options_set_write_buffer_size(rocksdb_options_t* opt, size_t s) {
  opt->rep.write_buffer_size = s;
}

void rocksdb_options_set_max_open_files(rocksdb_options_t* opt, int n) {
  opt->rep.max_open_files = n;
}

void rocksdb_options_set_max_total_wal_size(rocksdb_options_t* opt, uint64_t n) {
  opt->rep.max_total_wal_size = n;
}

void rocksdb_options_set_target_file_size_base(
    rocksdb_options_t* opt, uint64_t n) {
  opt->rep.target_file_size_base = n;
}

void rocksdb_options_set_target_file_size_multiplier(
    rocksdb_options_t* opt, int n) {
  opt->rep.target_file_size_multiplier = n;
}

void rocksdb_options_set_max_bytes_for_level_base(
    rocksdb_options_t* opt, uint64_t n) {
  opt->rep.max_bytes_for_level_base = n;
}

void rocksdb_options_set_max_bytes_for_level_multiplier(
    rocksdb_options_t* opt, int n) {
  opt->rep.max_bytes_for_level_multiplier = n;
}

void rocksdb_options_set_expanded_compaction_factor(
    rocksdb_options_t* opt, int n) {
  opt->rep.expanded_compaction_factor = n;
}

void rocksdb_options_set_max_grandparent_overlap_factor(
    rocksdb_options_t* opt, int n) {
  opt->rep.max_grandparent_overlap_factor = n;
}

void rocksdb_options_set_max_bytes_for_level_multiplier_additional(
    rocksdb_options_t* opt, int* level_values, size_t num_levels) {
  opt->rep.max_bytes_for_level_multiplier_additional.resize(num_levels);
  for (size_t i = 0; i < num_levels; ++i) {
    opt->rep.max_bytes_for_level_multiplier_additional[i] = level_values[i];
  }
}

void rocksdb_options_enable_statistics(rocksdb_options_t* opt) {
  opt->rep.statistics = rocksdb::CreateDBStatistics();
}

void rocksdb_options_set_num_levels(rocksdb_options_t* opt, int n) {
  opt->rep.num_levels = n;
}

void rocksdb_options_set_level0_file_num_compaction_trigger(
    rocksdb_options_t* opt, int n) {
  opt->rep.level0_file_num_compaction_trigger = n;
}

void rocksdb_options_set_level0_slowdown_writes_trigger(
    rocksdb_options_t* opt, int n) {
  opt->rep.level0_slowdown_writes_trigger = n;
}

void rocksdb_options_set_level0_stop_writes_trigger(
    rocksdb_options_t* opt, int n) {
  opt->rep.level0_stop_writes_trigger = n;
}

void rocksdb_options_set_max_mem_compaction_level(rocksdb_options_t* opt,
                                                  int n) {}

void rocksdb_options_set_compression(rocksdb_options_t* opt, int t) {
  opt->rep.compression = static_cast<CompressionType>(t);
}

void rocksdb_options_set_compression_per_level(rocksdb_options_t* opt,
                                               int* level_values,
                                               size_t num_levels) {
  opt->rep.compression_per_level.resize(num_levels);
  for (size_t i = 0; i < num_levels; ++i) {
    opt->rep.compression_per_level[i] =
      static_cast<CompressionType>(level_values[i]);
  }
}

void rocksdb_options_set_compression_options(
    rocksdb_options_t* opt, int w_bits, int level, int strategy) {
  opt->rep.compression_opts.window_bits = w_bits;
  opt->rep.compression_opts.level = level;
  opt->rep.compression_opts.strategy = strategy;
}

void rocksdb_options_set_prefix_extractor(
    rocksdb_options_t* opt, rocksdb_slicetransform_t* prefix_extractor) {
  opt->rep.prefix_extractor.reset(prefix_extractor);
}

void rocksdb_options_set_disable_data_sync(
    rocksdb_options_t* opt, int disable_data_sync) {
  opt->rep.disableDataSync = disable_data_sync;
}

void rocksdb_options_set_use_fsync(
    rocksdb_options_t* opt, int use_fsync) {
  opt->rep.use_fsync = use_fsync;
}

void rocksdb_options_set_db_log_dir(
    rocksdb_options_t* opt, const char* db_log_dir) {
  opt->rep.db_log_dir = db_log_dir;
}

void rocksdb_options_set_wal_dir(
    rocksdb_options_t* opt, const char* v) {
  opt->rep.wal_dir = v;
}

void rocksdb_options_set_WAL_ttl_seconds(rocksdb_options_t* opt, uint64_t ttl) {
  opt->rep.WAL_ttl_seconds = ttl;
}

void rocksdb_options_set_WAL_size_limit_MB(
    rocksdb_options_t* opt, uint64_t limit) {
  opt->rep.WAL_size_limit_MB = limit;
}

void rocksdb_options_set_manifest_preallocation_size(
    rocksdb_options_t* opt, size_t v) {
  opt->rep.manifest_preallocation_size = v;
}

// noop
void rocksdb_options_set_purge_redundant_kvs_while_flush(rocksdb_options_t* opt,
                                                         unsigned char v) {}

void rocksdb_options_set_allow_os_buffer(rocksdb_options_t* opt,
                                         unsigned char v) {
  opt->rep.allow_os_buffer = v;
}

void rocksdb_options_set_allow_mmap_reads(
    rocksdb_options_t* opt, unsigned char v) {
  opt->rep.allow_mmap_reads = v;
}

void rocksdb_options_set_allow_mmap_writes(
    rocksdb_options_t* opt, unsigned char v) {
  opt->rep.allow_mmap_writes = v;
}

void rocksdb_options_set_is_fd_close_on_exec(
    rocksdb_options_t* opt, unsigned char v) {
  opt->rep.is_fd_close_on_exec = v;
}

void rocksdb_options_set_skip_log_error_on_recovery(
    rocksdb_options_t* opt, unsigned char v) {
  opt->rep.skip_log_error_on_recovery = v;
}

void rocksdb_options_set_stats_dump_period_sec(
    rocksdb_options_t* opt, unsigned int v) {
  opt->rep.stats_dump_period_sec = v;
}

void rocksdb_options_set_advise_random_on_open(
    rocksdb_options_t* opt, unsigned char v) {
  opt->rep.advise_random_on_open = v;
}

void rocksdb_options_set_access_hint_on_compaction_start(
    rocksdb_options_t* opt, int v) {
  switch(v) {
    case 0:
      opt->rep.access_hint_on_compaction_start = rocksdb::Options::NONE;
      break;
    case 1:
      opt->rep.access_hint_on_compaction_start = rocksdb::Options::NORMAL;
      break;
    case 2:
      opt->rep.access_hint_on_compaction_start = rocksdb::Options::SEQUENTIAL;
      break;
    case 3:
      opt->rep.access_hint_on_compaction_start = rocksdb::Options::WILLNEED;
      break;
  }
}

void rocksdb_options_set_use_adaptive_mutex(
    rocksdb_options_t* opt, unsigned char v) {
  opt->rep.use_adaptive_mutex = v;
}

void rocksdb_options_set_bytes_per_sync(
    rocksdb_options_t* opt, uint64_t v) {
  opt->rep.bytes_per_sync = v;
}

void rocksdb_options_set_verify_checksums_in_compaction(
    rocksdb_options_t* opt, unsigned char v) {
  opt->rep.verify_checksums_in_compaction = v;
}

void rocksdb_options_set_filter_deletes(
    rocksdb_options_t* opt, unsigned char v) {
  opt->rep.filter_deletes = v;
}

void rocksdb_options_set_max_sequential_skip_in_iterations(
    rocksdb_options_t* opt, uint64_t v) {
  opt->rep.max_sequential_skip_in_iterations = v;
}

void rocksdb_options_set_max_write_buffer_number(rocksdb_options_t* opt, int n) {
  opt->rep.max_write_buffer_number = n;
}

void rocksdb_options_set_min_write_buffer_number_to_merge(rocksdb_options_t* opt, int n) {
  opt->rep.min_write_buffer_number_to_merge = n;
}

void rocksdb_options_set_max_write_buffer_number_to_maintain(
    rocksdb_options_t* opt, int n) {
  opt->rep.max_write_buffer_number_to_maintain = n;
}

void rocksdb_options_set_max_background_compactions(rocksdb_options_t* opt, int n) {
  opt->rep.max_background_compactions = n;
}

void rocksdb_options_set_max_background_flushes(rocksdb_options_t* opt, int n) {
  opt->rep.max_background_flushes = n;
}

void rocksdb_options_set_max_log_file_size(rocksdb_options_t* opt, size_t v) {
  opt->rep.max_log_file_size = v;
}

void rocksdb_options_set_log_file_time_to_roll(rocksdb_options_t* opt, size_t v) {
  opt->rep.log_file_time_to_roll = v;
}

void rocksdb_options_set_keep_log_file_num(rocksdb_options_t* opt, size_t v) {
  opt->rep.keep_log_file_num = v;
}

void rocksdb_options_set_soft_rate_limit(rocksdb_options_t* opt, double v) {
  opt->rep.soft_rate_limit = v;
}

void rocksdb_options_set_hard_rate_limit(rocksdb_options_t* opt, double v) {
  opt->rep.hard_rate_limit = v;
}

void rocksdb_options_set_rate_limit_delay_max_milliseconds(
    rocksdb_options_t* opt, unsigned int v) {
  opt->rep.rate_limit_delay_max_milliseconds = v;
}

void rocksdb_options_set_max_manifest_file_size(
    rocksdb_options_t* opt, size_t v) {
  opt->rep.max_manifest_file_size = v;
}

void rocksdb_options_set_table_cache_numshardbits(
    rocksdb_options_t* opt, int v) {
  opt->rep.table_cache_numshardbits = v;
}

void rocksdb_options_set_table_cache_remove_scan_count_limit(
    rocksdb_options_t* opt, int v) {
  // this option is deprecated
}

void rocksdb_options_set_arena_block_size(
    rocksdb_options_t* opt, size_t v) {
  opt->rep.arena_block_size = v;
}

void rocksdb_options_set_disable_auto_compactions(rocksdb_options_t* opt, int disable) {
  opt->rep.disable_auto_compactions = disable;
}

void rocksdb_options_set_delete_obsolete_files_period_micros(
    rocksdb_options_t* opt, uint64_t v) {
  opt->rep.delete_obsolete_files_period_micros = v;
}

void rocksdb_options_set_source_compaction_factor(
    rocksdb_options_t* opt, int n) {
  opt->rep.expanded_compaction_factor = n;
}

void rocksdb_options_prepare_for_bulk_load(rocksdb_options_t* opt) {
  opt->rep.PrepareForBulkLoad();
}

void rocksdb_options_set_memtable_vector_rep(rocksdb_options_t *opt) {
  static rocksdb::VectorRepFactory* factory = 0;
  if (!factory) {
    factory = new rocksdb::VectorRepFactory;
  }
  opt->rep.memtable_factory.reset(factory);
}

void rocksdb_options_set_memtable_prefix_bloom_bits(
    rocksdb_options_t* opt, uint32_t v) {
  opt->rep.memtable_prefix_bloom_bits = v;
}

void rocksdb_options_set_memtable_prefix_bloom_probes(
    rocksdb_options_t* opt, uint32_t v) {
  opt->rep.memtable_prefix_bloom_probes = v;
}

void rocksdb_options_set_hash_skip_list_rep(
    rocksdb_options_t *opt, size_t bucket_count,
    int32_t skiplist_height, int32_t skiplist_branching_factor) {
  static rocksdb::MemTableRepFactory* factory = 0;
  if (!factory) {
    factory = rocksdb::NewHashSkipListRepFactory(
        bucket_count, skiplist_height, skiplist_branching_factor);
  }
  opt->rep.memtable_factory.reset(factory);
}

void rocksdb_options_set_hash_link_list_rep(
    rocksdb_options_t *opt, size_t bucket_count) {
  static rocksdb::MemTableRepFactory* factory = 0;
  if (!factory) {
    factory = rocksdb::NewHashLinkListRepFactory(bucket_count);
  }
  opt->rep.memtable_factory.reset(factory);
}

void rocksdb_options_set_plain_table_factory(
    rocksdb_options_t *opt, uint32_t user_key_len, int bloom_bits_per_key,
    double hash_table_ratio, size_t index_sparseness) {
  static rocksdb::TableFactory* factory = 0;
  if (!factory) {
    rocksdb::PlainTableOptions options;
    options.user_key_len = user_key_len;
    options.bloom_bits_per_key = bloom_bits_per_key;
    options.hash_table_ratio = hash_table_ratio;
    options.index_sparseness = index_sparseness;

    factory = rocksdb::NewPlainTableFactory(options);
  }
  opt->rep.table_factory.reset(factory);
}

void rocksdb_options_set_max_successive_merges(
    rocksdb_options_t* opt, size_t v) {
  opt->rep.max_successive_merges = v;
}

void rocksdb_options_set_min_partial_merge_operands(
    rocksdb_options_t* opt, uint32_t v) {
  opt->rep.min_partial_merge_operands = v;
}

void rocksdb_options_set_bloom_locality(
    rocksdb_options_t* opt, uint32_t v) {
  opt->rep.bloom_locality = v;
}

void rocksdb_options_set_inplace_update_support(
    rocksdb_options_t* opt, unsigned char v) {
  opt->rep.inplace_update_support = v;
}

void rocksdb_options_set_inplace_update_num_locks(
    rocksdb_options_t* opt, size_t v) {
  opt->rep.inplace_update_num_locks = v;
}

void rocksdb_options_set_compaction_style(rocksdb_options_t *opt, int style) {
  opt->rep.compaction_style = static_cast<rocksdb::CompactionStyle>(style);
}

void rocksdb_options_set_universal_compaction_options(rocksdb_options_t *opt, rocksdb_universal_compaction_options_t *uco) {
  opt->rep.compaction_options_universal = *(uco->rep);
}

void rocksdb_options_set_fifo_compaction_options(
    rocksdb_options_t* opt,
    rocksdb_fifo_compaction_options_t* fifo) {
  opt->rep.compaction_options_fifo = fifo->rep;
}

char *rocksdb_options_statistics_get_string(rocksdb_options_t *opt) {
  rocksdb::Statistics *statistics = opt->rep.statistics.get();
  if (statistics) {
    return strdup(statistics->ToString().c_str());
  }
  return nullptr;
}

/*
TODO:
DB::OpenForReadOnly
DB::KeyMayExist
DB::GetOptions
DB::GetSortedWalFiles
DB::GetLatestSequenceNumber
DB::GetUpdatesSince
DB::GetDbIdentity
DB::RunManualCompaction
custom cache
table_properties_collectors
*/

rocksdb_compactionfilter_t* rocksdb_compactionfilter_create(
    void* state,
    void (*destructor)(void*),
    unsigned char (*filter)(
        void*,
        int level,
        const char* key, size_t key_length,
        const char* existing_value, size_t value_length,
        char** new_value, size_t *new_value_length,
        unsigned char* value_changed),
    const char* (*name)(void*)) {
  rocksdb_compactionfilter_t* result = new rocksdb_compactionfilter_t;
  result->state_ = state;
  result->destructor_ = destructor;
  result->filter_ = filter;
  result->name_ = name;
  return result;
}

void rocksdb_compactionfilter_destroy(rocksdb_compactionfilter_t* filter) {
  delete filter;
}

unsigned char rocksdb_compactionfiltercontext_is_full_compaction(
    rocksdb_compactionfiltercontext_t* context) {
  return context->rep.is_full_compaction;
}

unsigned char rocksdb_compactionfiltercontext_is_manual_compaction(
    rocksdb_compactionfiltercontext_t* context) {
  return context->rep.is_manual_compaction;
}

rocksdb_compactionfilterfactory_t* rocksdb_compactionfilterfactory_create(
    void* state, void (*destructor)(void*),
    rocksdb_compactionfilter_t* (*create_compaction_filter)(
        void*, rocksdb_compactionfiltercontext_t* context),
    const char* (*name)(void*)) {
  rocksdb_compactionfilterfactory_t* result =
      new rocksdb_compactionfilterfactory_t;
  result->state_ = state;
  result->destructor_ = destructor;
  result->create_compaction_filter_ = create_compaction_filter;
  result->name_ = name;
  return result;
}

void rocksdb_compactionfilterfactory_destroy(
    rocksdb_compactionfilterfactory_t* factory) {
  delete factory;
}

rocksdb_comparator_t* rocksdb_comparator_create(
    void* state,
    void (*destructor)(void*),
    int (*compare)(
        void*,
        const char* a, size_t alen,
        const char* b, size_t blen),
    const char* (*name)(void*)) {
  rocksdb_comparator_t* result = new rocksdb_comparator_t;
  result->state_ = state;
  result->destructor_ = destructor;
  result->compare_ = compare;
  result->name_ = name;
  return result;
}

void rocksdb_comparator_destroy(rocksdb_comparator_t* cmp) {
  delete cmp;
}

rocksdb_filterpolicy_t* rocksdb_filterpolicy_create(
    void* state,
    void (*destructor)(void*),
    char* (*create_filter)(
        void*,
        const char* const* key_array, const size_t* key_length_array,
        int num_keys,
        size_t* filter_length),
    unsigned char (*key_may_match)(
        void*,
        const char* key, size_t length,
        const char* filter, size_t filter_length),
    void (*delete_filter)(
        void*,
        const char* filter, size_t filter_length),
    const char* (*name)(void*)) {
  rocksdb_filterpolicy_t* result = new rocksdb_filterpolicy_t;
  result->state_ = state;
  result->destructor_ = destructor;
  result->create_ = create_filter;
  result->key_match_ = key_may_match;
  result->delete_filter_ = delete_filter;
  result->name_ = name;
  return result;
}

void rocksdb_filterpolicy_destroy(rocksdb_filterpolicy_t* filter) {
  delete filter;
}

rocksdb_filterpolicy_t* rocksdb_filterpolicy_create_bloom(int bits_per_key) {
  // Make a rocksdb_filterpolicy_t, but override all of its methods so
  // they delegate to a NewBloomFilterPolicy() instead of user
  // supplied C functions.
  struct Wrapper : public rocksdb_filterpolicy_t {
    const FilterPolicy* rep_;
    ~Wrapper() { delete rep_; }
    const char* Name() const override { return rep_->Name(); }
    void CreateFilter(const Slice* keys, int n,
                      std::string* dst) const override {
      return rep_->CreateFilter(keys, n, dst);
    }
    bool KeyMayMatch(const Slice& key, const Slice& filter) const override {
      return rep_->KeyMayMatch(key, filter);
    }
    static void DoNothing(void*) { }
  };
  Wrapper* wrapper = new Wrapper;
  wrapper->rep_ = NewBloomFilterPolicy(bits_per_key);
  wrapper->state_ = nullptr;
  wrapper->delete_filter_ = nullptr;
  wrapper->destructor_ = &Wrapper::DoNothing;
  return wrapper;
}

rocksdb_mergeoperator_t* rocksdb_mergeoperator_create(
    void* state, void (*destructor)(void*),
    char* (*full_merge)(void*, const char* key, size_t key_length,
                        const char* existing_value,
                        size_t existing_value_length,
                        const char* const* operands_list,
                        const size_t* operands_list_length, int num_operands,
                        unsigned char* success, size_t* new_value_length),
    char* (*partial_merge)(void*, const char* key, size_t key_length,
                           const char* const* operands_list,
                           const size_t* operands_list_length, int num_operands,
                           unsigned char* success, size_t* new_value_length),
    void (*delete_value)(void*, const char* value, size_t value_length),
    const char* (*name)(void*)) {
  rocksdb_mergeoperator_t* result = new rocksdb_mergeoperator_t;
  result->state_ = state;
  result->destructor_ = destructor;
  result->full_merge_ = full_merge;
  result->partial_merge_ = partial_merge;
  result->delete_value_ = delete_value;
  result->name_ = name;
  return result;
}

void rocksdb_mergeoperator_destroy(rocksdb_mergeoperator_t* merge_operator) {
  delete merge_operator;
}

rocksdb_readoptions_t* rocksdb_readoptions_create() {
  return new rocksdb_readoptions_t;
}

void rocksdb_readoptions_destroy(rocksdb_readoptions_t* opt) {
  delete opt;
}

void rocksdb_readoptions_set_verify_checksums(
    rocksdb_readoptions_t* opt,
    unsigned char v) {
  opt->rep.verify_checksums = v;
}

void rocksdb_readoptions_set_fill_cache(
    rocksdb_readoptions_t* opt, unsigned char v) {
  opt->rep.fill_cache = v;
}

void rocksdb_readoptions_set_snapshot(
    rocksdb_readoptions_t* opt,
    const rocksdb_snapshot_t* snap) {
  opt->rep.snapshot = (snap ? snap->rep : nullptr);
}

void rocksdb_readoptions_set_iterate_upper_bound(
    rocksdb_readoptions_t* opt,
    const char* key, size_t keylen) {
  if (key == nullptr) {
    opt->upper_bound = Slice();
    opt->rep.iterate_upper_bound = nullptr;

  } else {
    opt->upper_bound = Slice(key, keylen);
    opt->rep.iterate_upper_bound = &opt->upper_bound;
  }
}

void rocksdb_readoptions_set_read_tier(
    rocksdb_readoptions_t* opt, int v) {
  opt->rep.read_tier = static_cast<rocksdb::ReadTier>(v);
}

void rocksdb_readoptions_set_tailing(
    rocksdb_readoptions_t* opt, unsigned char v) {
  opt->rep.tailing = v;
}

rocksdb_writeoptions_t* rocksdb_writeoptions_create() {
  return new rocksdb_writeoptions_t;
}

void rocksdb_writeoptions_destroy(rocksdb_writeoptions_t* opt) {
  delete opt;
}

void rocksdb_writeoptions_set_sync(
    rocksdb_writeoptions_t* opt, unsigned char v) {
  opt->rep.sync = v;
}

void rocksdb_writeoptions_disable_WAL(rocksdb_writeoptions_t* opt, int disable) {
  opt->rep.disableWAL = disable;
}


rocksdb_flushoptions_t* rocksdb_flushoptions_create() {
  return new rocksdb_flushoptions_t;
}

void rocksdb_flushoptions_destroy(rocksdb_flushoptions_t* opt) {
  delete opt;
}

void rocksdb_flushoptions_set_wait(
    rocksdb_flushoptions_t* opt, unsigned char v) {
  opt->rep.wait = v;
}

rocksdb_cache_t* rocksdb_cache_create_lru(size_t capacity) {
  rocksdb_cache_t* c = new rocksdb_cache_t;
  c->rep = NewLRUCache(capacity);
  return c;
}

void rocksdb_cache_destroy(rocksdb_cache_t* cache) {
  delete cache;
}

rocksdb_env_t* rocksdb_create_default_env() {
  rocksdb_env_t* result = new rocksdb_env_t;
  result->rep = Env::Default();
  result->is_default = true;
  return result;
}

void rocksdb_env_set_background_threads(rocksdb_env_t* env, int n) {
  env->rep->SetBackgroundThreads(n);
}

void rocksdb_env_set_high_priority_background_threads(rocksdb_env_t* env, int n) {
  env->rep->SetBackgroundThreads(n, Env::HIGH);
}

void rocksdb_env_join_all_threads(rocksdb_env_t* env) {
  env->rep->WaitForJoin();
}

void rocksdb_env_destroy(rocksdb_env_t* env) {
  if (!env->is_default) delete env->rep;
  delete env;
}

rocksdb_slicetransform_t* rocksdb_slicetransform_create(
    void* state,
    void (*destructor)(void*),
    char* (*transform)(
        void*,
        const char* key, size_t length,
        size_t* dst_length),
    unsigned char (*in_domain)(
        void*,
        const char* key, size_t length),
    unsigned char (*in_range)(
        void*,
        const char* key, size_t length),
    const char* (*name)(void*)) {
  rocksdb_slicetransform_t* result = new rocksdb_slicetransform_t;
  result->state_ = state;
  result->destructor_ = destructor;
  result->transform_ = transform;
  result->in_domain_ = in_domain;
  result->in_range_ = in_range;
  result->name_ = name;
  return result;
}

void rocksdb_slicetransform_destroy(rocksdb_slicetransform_t* st) {
  delete st;
}

rocksdb_slicetransform_t* rocksdb_slicetransform_create_fixed_prefix(size_t prefixLen) {
  struct Wrapper : public rocksdb_slicetransform_t {
    const SliceTransform* rep_;
    ~Wrapper() { delete rep_; }
    const char* Name() const override { return rep_->Name(); }
    Slice Transform(const Slice& src) const override {
      return rep_->Transform(src);
    }
    bool InDomain(const Slice& src) const override {
      return rep_->InDomain(src);
    }
    bool InRange(const Slice& src) const override { return rep_->InRange(src); }
    static void DoNothing(void*) { }
  };
  Wrapper* wrapper = new Wrapper;
  wrapper->rep_ = rocksdb::NewFixedPrefixTransform(prefixLen);
  wrapper->state_ = nullptr;
  wrapper->destructor_ = &Wrapper::DoNothing;
  return wrapper;
}

rocksdb_slicetransform_t* rocksdb_slicetransform_create_noop() {
  struct Wrapper : public rocksdb_slicetransform_t {
    const SliceTransform* rep_;
    ~Wrapper() { delete rep_; }
    const char* Name() const override { return rep_->Name(); }
    Slice Transform(const Slice& src) const override {
      return rep_->Transform(src);
    }
    bool InDomain(const Slice& src) const override {
      return rep_->InDomain(src);
    }
    bool InRange(const Slice& src) const override { return rep_->InRange(src); }
    static void DoNothing(void*) { }
  };
  Wrapper* wrapper = new Wrapper;
  wrapper->rep_ = rocksdb::NewNoopTransform();
  wrapper->state_ = nullptr;
  wrapper->destructor_ = &Wrapper::DoNothing;
  return wrapper;
}

rocksdb_universal_compaction_options_t* rocksdb_universal_compaction_options_create() {
  rocksdb_universal_compaction_options_t* result = new rocksdb_universal_compaction_options_t;
  result->rep = new rocksdb::CompactionOptionsUniversal;
  return result;
}

void rocksdb_universal_compaction_options_set_size_ratio(
  rocksdb_universal_compaction_options_t* uco, int ratio) {
  uco->rep->size_ratio = ratio;
}

void rocksdb_universal_compaction_options_set_min_merge_width(
  rocksdb_universal_compaction_options_t* uco, int w) {
  uco->rep->min_merge_width = w;
}

void rocksdb_universal_compaction_options_set_max_merge_width(
  rocksdb_universal_compaction_options_t* uco, int w) {
  uco->rep->max_merge_width = w;
}

void rocksdb_universal_compaction_options_set_max_size_amplification_percent(
  rocksdb_universal_compaction_options_t* uco, int p) {
  uco->rep->max_size_amplification_percent = p;
}

void rocksdb_universal_compaction_options_set_compression_size_percent(
  rocksdb_universal_compaction_options_t* uco, int p) {
  uco->rep->compression_size_percent = p;
}

void rocksdb_universal_compaction_options_set_stop_style(
  rocksdb_universal_compaction_options_t* uco, int style) {
  uco->rep->stop_style = static_cast<rocksdb::CompactionStopStyle>(style);
}

void rocksdb_universal_compaction_options_destroy(
  rocksdb_universal_compaction_options_t* uco) {
  delete uco->rep;
  delete uco;
}

rocksdb_fifo_compaction_options_t* rocksdb_fifo_compaction_options_create() {
  rocksdb_fifo_compaction_options_t* result = new rocksdb_fifo_compaction_options_t;
  result->rep =  CompactionOptionsFIFO();
  return result;
}

void rocksdb_fifo_compaction_options_set_max_table_files_size(
    rocksdb_fifo_compaction_options_t* fifo_opts, uint64_t size) {
  fifo_opts->rep.max_table_files_size = size;
}

void rocksdb_fifo_compaction_options_destroy(
    rocksdb_fifo_compaction_options_t* fifo_opts) {
  delete fifo_opts;
}

void rocksdb_options_set_min_level_to_compress(rocksdb_options_t* opt, int level) {
  if (level >= 0) {
    assert(level <= opt->rep.num_levels);
    opt->rep.compression_per_level.resize(opt->rep.num_levels);
    for (int i = 0; i < level; i++) {
      opt->rep.compression_per_level[i] = rocksdb::kNoCompression;
    }
    for (int i = level; i < opt->rep.num_levels; i++) {
      opt->rep.compression_per_level[i] = opt->rep.compression;
    }
  }
}

int rocksdb_livefiles_count(
  const rocksdb_livefiles_t* lf) {
  return static_cast<int>(lf->rep.size());
}

const char* rocksdb_livefiles_name(
  const rocksdb_livefiles_t* lf,
  int index) {
  return lf->rep[index].name.c_str();
}

int rocksdb_livefiles_level(
  const rocksdb_livefiles_t* lf,
  int index) {
  return lf->rep[index].level;
}

size_t rocksdb_livefiles_size(
  const rocksdb_livefiles_t* lf,
  int index) {
  return lf->rep[index].size;
}

const char* rocksdb_livefiles_smallestkey(
  const rocksdb_livefiles_t* lf,
  int index,
  size_t* size) {
  *size = lf->rep[index].smallestkey.size();
  return lf->rep[index].smallestkey.data();
}

const char* rocksdb_livefiles_largestkey(
  const rocksdb_livefiles_t* lf,
  int index,
  size_t* size) {
  *size = lf->rep[index].largestkey.size();
  return lf->rep[index].largestkey.data();
}

extern void rocksdb_livefiles_destroy(
  const rocksdb_livefiles_t* lf) {
  delete lf;
}

void rocksdb_get_options_from_string(const rocksdb_options_t* base_options,
                                     const char* opts_str,
                                     rocksdb_options_t* new_options,
                                     char** errptr) {
  SaveError(errptr,
            GetOptionsFromString(base_options->rep, std::string(opts_str),
                                 &new_options->rep));
}

void rocksdb_free(void* ptr) { free(ptr); }

}  // end extern "C"

#endif  // !ROCKSDB_LITE
