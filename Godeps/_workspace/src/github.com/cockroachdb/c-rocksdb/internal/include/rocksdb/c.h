/*  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
  This source code is licensed under the BSD-style license found in the
  LICENSE file in the root directory of this source tree. An additional grant
  of patent rights can be found in the PATENTS file in the same directory.
 Copyright (c) 2011 The LevelDB Authors. All rights reserved.
  Use of this source code is governed by a BSD-style license that can be
  found in the LICENSE file. See the AUTHORS file for names of contributors.

  C bindings for rocksdb.  May be useful as a stable ABI that can be
  used by programs that keep rocksdb in a shared library, or for
  a JNI api.

  Does not support:
  . getters for the option types
  . custom comparators that implement key shortening
  . capturing post-write-snapshot
  . custom iter, db, env, cache implementations using just the C bindings

  Some conventions:

  (1) We expose just opaque struct pointers and functions to clients.
  This allows us to change internal representations without having to
  recompile clients.

  (2) For simplicity, there is no equivalent to the Slice type.  Instead,
  the caller has to pass the pointer and length as separate
  arguments.

  (3) Errors are represented by a null-terminated c string.  NULL
  means no error.  All operations that can raise an error are passed
  a "char** errptr" as the last argument.  One of the following must
  be true on entry:
     *errptr == NULL
     *errptr points to a malloc()ed null-terminated error message
  On success, a leveldb routine leaves *errptr unchanged.
  On failure, leveldb frees the old value of *errptr and
  set *errptr to a malloc()ed error message.

  (4) Bools have the type unsigned char (0 == false; rest == true)

  (5) All of the pointer arguments must be non-NULL.
*/

#ifndef STORAGE_ROCKSDB_INCLUDE_C_H_
#define STORAGE_ROCKSDB_INCLUDE_C_H_

#pragma once

#ifdef _WIN32
#ifdef ROCKSDB_DLL
#ifdef ROCKSDB_LIBRARY_EXPORTS
#define ROCKSDB_LIBRARY_API __declspec(dllexport)
#else
#define ROCKSDB_LIBRARY_API __declspec(dllimport)
#endif
#else
#define ROCKSDB_LIBRARY_API
#endif
#else
#define ROCKSDB_LIBRARY_API
#endif

#ifdef __cplusplus
extern "C" {
#endif

#include <stdarg.h>
#include <stddef.h>
#include <stdint.h>

/* Exported types */

typedef struct rocksdb_t                 rocksdb_t;
typedef struct rocksdb_backup_engine_t   rocksdb_backup_engine_t;
typedef struct rocksdb_backup_engine_info_t   rocksdb_backup_engine_info_t;
typedef struct rocksdb_restore_options_t rocksdb_restore_options_t;
typedef struct rocksdb_cache_t           rocksdb_cache_t;
typedef struct rocksdb_compactionfilter_t rocksdb_compactionfilter_t;
typedef struct rocksdb_compactionfiltercontext_t
    rocksdb_compactionfiltercontext_t;
typedef struct rocksdb_compactionfilterfactory_t
    rocksdb_compactionfilterfactory_t;
typedef struct rocksdb_comparator_t      rocksdb_comparator_t;
typedef struct rocksdb_env_t             rocksdb_env_t;
typedef struct rocksdb_fifo_compaction_options_t rocksdb_fifo_compaction_options_t;
typedef struct rocksdb_filelock_t        rocksdb_filelock_t;
typedef struct rocksdb_filterpolicy_t    rocksdb_filterpolicy_t;
typedef struct rocksdb_flushoptions_t    rocksdb_flushoptions_t;
typedef struct rocksdb_iterator_t        rocksdb_iterator_t;
typedef struct rocksdb_logger_t          rocksdb_logger_t;
typedef struct rocksdb_mergeoperator_t   rocksdb_mergeoperator_t;
typedef struct rocksdb_options_t         rocksdb_options_t;
typedef struct rocksdb_block_based_table_options_t
    rocksdb_block_based_table_options_t;
typedef struct rocksdb_cuckoo_table_options_t
    rocksdb_cuckoo_table_options_t;
typedef struct rocksdb_randomfile_t      rocksdb_randomfile_t;
typedef struct rocksdb_readoptions_t     rocksdb_readoptions_t;
typedef struct rocksdb_seqfile_t         rocksdb_seqfile_t;
typedef struct rocksdb_slicetransform_t  rocksdb_slicetransform_t;
typedef struct rocksdb_snapshot_t        rocksdb_snapshot_t;
typedef struct rocksdb_writablefile_t    rocksdb_writablefile_t;
typedef struct rocksdb_writebatch_t      rocksdb_writebatch_t;
typedef struct rocksdb_writeoptions_t    rocksdb_writeoptions_t;
typedef struct rocksdb_universal_compaction_options_t rocksdb_universal_compaction_options_t;
typedef struct rocksdb_livefiles_t     rocksdb_livefiles_t;
typedef struct rocksdb_column_family_handle_t rocksdb_column_family_handle_t;

/* DB operations */

extern ROCKSDB_LIBRARY_API rocksdb_t* rocksdb_open(
    const rocksdb_options_t* options, const char* name, char** errptr);

extern ROCKSDB_LIBRARY_API rocksdb_t* rocksdb_open_for_read_only(
    const rocksdb_options_t* options, const char* name,
    unsigned char error_if_log_file_exist, char** errptr);

extern ROCKSDB_LIBRARY_API rocksdb_backup_engine_t* rocksdb_backup_engine_open(
    const rocksdb_options_t* options, const char* path, char** errptr);

extern ROCKSDB_LIBRARY_API void rocksdb_backup_engine_create_new_backup(
    rocksdb_backup_engine_t* be, rocksdb_t* db, char** errptr);

extern ROCKSDB_LIBRARY_API rocksdb_restore_options_t*
rocksdb_restore_options_create();
extern ROCKSDB_LIBRARY_API void rocksdb_restore_options_destroy(
    rocksdb_restore_options_t* opt);
extern ROCKSDB_LIBRARY_API void rocksdb_restore_options_set_keep_log_files(
    rocksdb_restore_options_t* opt, int v);

extern ROCKSDB_LIBRARY_API void
rocksdb_backup_engine_restore_db_from_latest_backup(
    rocksdb_backup_engine_t* be, const char* db_dir, const char* wal_dir,
    const rocksdb_restore_options_t* restore_options, char** errptr);

extern ROCKSDB_LIBRARY_API const rocksdb_backup_engine_info_t*
rocksdb_backup_engine_get_backup_info(rocksdb_backup_engine_t* be);

extern ROCKSDB_LIBRARY_API int rocksdb_backup_engine_info_count(
    const rocksdb_backup_engine_info_t* info);

extern ROCKSDB_LIBRARY_API int64_t
rocksdb_backup_engine_info_timestamp(const rocksdb_backup_engine_info_t* info,
                                     int index);

extern ROCKSDB_LIBRARY_API uint32_t
rocksdb_backup_engine_info_backup_id(const rocksdb_backup_engine_info_t* info,
                                     int index);

extern ROCKSDB_LIBRARY_API uint64_t
rocksdb_backup_engine_info_size(const rocksdb_backup_engine_info_t* info,
                                int index);

extern ROCKSDB_LIBRARY_API uint32_t rocksdb_backup_engine_info_number_files(
    const rocksdb_backup_engine_info_t* info, int index);

extern ROCKSDB_LIBRARY_API void rocksdb_backup_engine_info_destroy(
    const rocksdb_backup_engine_info_t* info);

extern ROCKSDB_LIBRARY_API void rocksdb_backup_engine_close(
    rocksdb_backup_engine_t* be);

extern ROCKSDB_LIBRARY_API rocksdb_t* rocksdb_open_column_families(
    const rocksdb_options_t* options, const char* name, int num_column_families,
    const char** column_family_names,
    const rocksdb_options_t** column_family_options,
    rocksdb_column_family_handle_t** column_family_handles, char** errptr);

extern ROCKSDB_LIBRARY_API rocksdb_t*
rocksdb_open_for_read_only_column_families(
    const rocksdb_options_t* options, const char* name, int num_column_families,
    const char** column_family_names,
    const rocksdb_options_t** column_family_options,
    rocksdb_column_family_handle_t** column_family_handles,
    unsigned char error_if_log_file_exist, char** errptr);

extern ROCKSDB_LIBRARY_API char** rocksdb_list_column_families(
    const rocksdb_options_t* options, const char* name, size_t* lencf,
    char** errptr);

extern ROCKSDB_LIBRARY_API void rocksdb_list_column_families_destroy(
    char** list, size_t len);

extern ROCKSDB_LIBRARY_API rocksdb_column_family_handle_t*
rocksdb_create_column_family(rocksdb_t* db,
                             const rocksdb_options_t* column_family_options,
                             const char* column_family_name, char** errptr);

extern ROCKSDB_LIBRARY_API void rocksdb_drop_column_family(
    rocksdb_t* db, rocksdb_column_family_handle_t* handle, char** errptr);

extern ROCKSDB_LIBRARY_API void rocksdb_column_family_handle_destroy(
    rocksdb_column_family_handle_t*);

extern ROCKSDB_LIBRARY_API void rocksdb_close(rocksdb_t* db);

extern ROCKSDB_LIBRARY_API void rocksdb_put(
    rocksdb_t* db, const rocksdb_writeoptions_t* options, const char* key,
    size_t keylen, const char* val, size_t vallen, char** errptr);

extern ROCKSDB_LIBRARY_API void rocksdb_put_cf(
    rocksdb_t* db, const rocksdb_writeoptions_t* options,
    rocksdb_column_family_handle_t* column_family, const char* key,
    size_t keylen, const char* val, size_t vallen, char** errptr);

extern ROCKSDB_LIBRARY_API void rocksdb_delete(
    rocksdb_t* db, const rocksdb_writeoptions_t* options, const char* key,
    size_t keylen, char** errptr);

extern ROCKSDB_LIBRARY_API void rocksdb_delete_cf(
    rocksdb_t* db, const rocksdb_writeoptions_t* options,
    rocksdb_column_family_handle_t* column_family, const char* key,
    size_t keylen, char** errptr);

extern ROCKSDB_LIBRARY_API void rocksdb_merge(
    rocksdb_t* db, const rocksdb_writeoptions_t* options, const char* key,
    size_t keylen, const char* val, size_t vallen, char** errptr);

extern ROCKSDB_LIBRARY_API void rocksdb_merge_cf(
    rocksdb_t* db, const rocksdb_writeoptions_t* options,
    rocksdb_column_family_handle_t* column_family, const char* key,
    size_t keylen, const char* val, size_t vallen, char** errptr);

extern ROCKSDB_LIBRARY_API void rocksdb_write(
    rocksdb_t* db, const rocksdb_writeoptions_t* options,
    rocksdb_writebatch_t* batch, char** errptr);

/* Returns NULL if not found.  A malloc()ed array otherwise.
   Stores the length of the array in *vallen. */
extern ROCKSDB_LIBRARY_API char* rocksdb_get(
    rocksdb_t* db, const rocksdb_readoptions_t* options, const char* key,
    size_t keylen, size_t* vallen, char** errptr);

extern ROCKSDB_LIBRARY_API char* rocksdb_get_cf(
    rocksdb_t* db, const rocksdb_readoptions_t* options,
    rocksdb_column_family_handle_t* column_family, const char* key,
    size_t keylen, size_t* vallen, char** errptr);

// if values_list[i] == NULL and errs[i] == NULL,
// then we got status.IsNotFound(), which we will not return.
// all errors except status status.ok() and status.IsNotFound() are returned.
//
// errs, values_list and values_list_sizes must be num_keys in length,
// allocated by the caller.
// errs is a list of strings as opposed to the conventional one error,
// where errs[i] is the status for retrieval of keys_list[i].
// each non-NULL errs entry is a malloc()ed, null terminated string.
// each non-NULL values_list entry is a malloc()ed array, with
// the length for each stored in values_list_sizes[i].
extern ROCKSDB_LIBRARY_API void rocksdb_multi_get(
    rocksdb_t* db, const rocksdb_readoptions_t* options, size_t num_keys,
    const char* const* keys_list, const size_t* keys_list_sizes,
    char** values_list, size_t* values_list_sizes, char** errs);

extern ROCKSDB_LIBRARY_API void rocksdb_multi_get_cf(
    rocksdb_t* db, const rocksdb_readoptions_t* options,
    const rocksdb_column_family_handle_t* const* column_families,
    size_t num_keys, const char* const* keys_list,
    const size_t* keys_list_sizes, char** values_list,
    size_t* values_list_sizes, char** errs);

extern ROCKSDB_LIBRARY_API rocksdb_iterator_t* rocksdb_create_iterator(
    rocksdb_t* db, const rocksdb_readoptions_t* options);

extern ROCKSDB_LIBRARY_API rocksdb_iterator_t* rocksdb_create_iterator_cf(
    rocksdb_t* db, const rocksdb_readoptions_t* options,
    rocksdb_column_family_handle_t* column_family);

extern ROCKSDB_LIBRARY_API const rocksdb_snapshot_t* rocksdb_create_snapshot(
    rocksdb_t* db);

extern ROCKSDB_LIBRARY_API void rocksdb_release_snapshot(
    rocksdb_t* db, const rocksdb_snapshot_t* snapshot);

/* Returns NULL if property name is unknown.
   Else returns a pointer to a malloc()-ed null-terminated value. */
extern ROCKSDB_LIBRARY_API char* rocksdb_property_value(rocksdb_t* db,
                                                        const char* propname);

extern ROCKSDB_LIBRARY_API char* rocksdb_property_value_cf(
    rocksdb_t* db, rocksdb_column_family_handle_t* column_family,
    const char* propname);

extern ROCKSDB_LIBRARY_API void rocksdb_approximate_sizes(
    rocksdb_t* db, int num_ranges, const char* const* range_start_key,
    const size_t* range_start_key_len, const char* const* range_limit_key,
    const size_t* range_limit_key_len, uint64_t* sizes);

extern ROCKSDB_LIBRARY_API void rocksdb_approximate_sizes_cf(
    rocksdb_t* db, rocksdb_column_family_handle_t* column_family,
    int num_ranges, const char* const* range_start_key,
    const size_t* range_start_key_len, const char* const* range_limit_key,
    const size_t* range_limit_key_len, uint64_t* sizes);

extern ROCKSDB_LIBRARY_API void rocksdb_compact_range(rocksdb_t* db,
                                                      const char* start_key,
                                                      size_t start_key_len,
                                                      const char* limit_key,
                                                      size_t limit_key_len);

extern ROCKSDB_LIBRARY_API void rocksdb_compact_range_cf(
    rocksdb_t* db, rocksdb_column_family_handle_t* column_family,
    const char* start_key, size_t start_key_len, const char* limit_key,
    size_t limit_key_len);

extern ROCKSDB_LIBRARY_API void rocksdb_delete_file(rocksdb_t* db,
                                                    const char* name);

extern ROCKSDB_LIBRARY_API const rocksdb_livefiles_t* rocksdb_livefiles(
    rocksdb_t* db);

extern ROCKSDB_LIBRARY_API void rocksdb_flush(
    rocksdb_t* db, const rocksdb_flushoptions_t* options, char** errptr);

extern ROCKSDB_LIBRARY_API void rocksdb_disable_file_deletions(rocksdb_t* db,
                                                               char** errptr);

extern ROCKSDB_LIBRARY_API void rocksdb_enable_file_deletions(
    rocksdb_t* db, unsigned char force, char** errptr);

/* Management operations */

extern ROCKSDB_LIBRARY_API void rocksdb_destroy_db(
    const rocksdb_options_t* options, const char* name, char** errptr);

extern ROCKSDB_LIBRARY_API void rocksdb_repair_db(
    const rocksdb_options_t* options, const char* name, char** errptr);

/* Iterator */

extern ROCKSDB_LIBRARY_API void rocksdb_iter_destroy(rocksdb_iterator_t*);
extern ROCKSDB_LIBRARY_API unsigned char rocksdb_iter_valid(
    const rocksdb_iterator_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_iter_seek_to_first(rocksdb_iterator_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_iter_seek_to_last(rocksdb_iterator_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_iter_seek(rocksdb_iterator_t*,
                                                  const char* k, size_t klen);
extern ROCKSDB_LIBRARY_API void rocksdb_iter_next(rocksdb_iterator_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_iter_prev(rocksdb_iterator_t*);
extern ROCKSDB_LIBRARY_API const char* rocksdb_iter_key(
    const rocksdb_iterator_t*, size_t* klen);
extern ROCKSDB_LIBRARY_API const char* rocksdb_iter_value(
    const rocksdb_iterator_t*, size_t* vlen);
extern ROCKSDB_LIBRARY_API void rocksdb_iter_get_error(
    const rocksdb_iterator_t*, char** errptr);

/* Write batch */

extern ROCKSDB_LIBRARY_API rocksdb_writebatch_t* rocksdb_writebatch_create();
extern ROCKSDB_LIBRARY_API rocksdb_writebatch_t* rocksdb_writebatch_create_from(
    const char* rep, size_t size);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_destroy(
    rocksdb_writebatch_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_clear(rocksdb_writebatch_t*);
extern ROCKSDB_LIBRARY_API int rocksdb_writebatch_count(rocksdb_writebatch_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_put(rocksdb_writebatch_t*,
                                                       const char* key,
                                                       size_t klen,
                                                       const char* val,
                                                       size_t vlen);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_put_cf(
    rocksdb_writebatch_t*, rocksdb_column_family_handle_t* column_family,
    const char* key, size_t klen, const char* val, size_t vlen);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_putv(
    rocksdb_writebatch_t* b, int num_keys, const char* const* keys_list,
    const size_t* keys_list_sizes, int num_values,
    const char* const* values_list, const size_t* values_list_sizes);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_putv_cf(
    rocksdb_writebatch_t* b, rocksdb_column_family_handle_t* column_family,
    int num_keys, const char* const* keys_list, const size_t* keys_list_sizes,
    int num_values, const char* const* values_list,
    const size_t* values_list_sizes);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_merge(rocksdb_writebatch_t*,
                                                         const char* key,
                                                         size_t klen,
                                                         const char* val,
                                                         size_t vlen);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_merge_cf(
    rocksdb_writebatch_t*, rocksdb_column_family_handle_t* column_family,
    const char* key, size_t klen, const char* val, size_t vlen);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_mergev(
    rocksdb_writebatch_t* b, int num_keys, const char* const* keys_list,
    const size_t* keys_list_sizes, int num_values,
    const char* const* values_list, const size_t* values_list_sizes);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_mergev_cf(
    rocksdb_writebatch_t* b, rocksdb_column_family_handle_t* column_family,
    int num_keys, const char* const* keys_list, const size_t* keys_list_sizes,
    int num_values, const char* const* values_list,
    const size_t* values_list_sizes);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_delete(rocksdb_writebatch_t*,
                                                          const char* key,
                                                          size_t klen);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_delete_cf(
    rocksdb_writebatch_t*, rocksdb_column_family_handle_t* column_family,
    const char* key, size_t klen);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_deletev(
    rocksdb_writebatch_t* b, int num_keys, const char* const* keys_list,
    const size_t* keys_list_sizes);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_deletev_cf(
    rocksdb_writebatch_t* b, rocksdb_column_family_handle_t* column_family,
    int num_keys, const char* const* keys_list, const size_t* keys_list_sizes);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_put_log_data(
    rocksdb_writebatch_t*, const char* blob, size_t len);
extern ROCKSDB_LIBRARY_API void rocksdb_writebatch_iterate(
    rocksdb_writebatch_t*, void* state,
    void (*put)(void*, const char* k, size_t klen, const char* v, size_t vlen),
    void (*deleted)(void*, const char* k, size_t klen));
extern ROCKSDB_LIBRARY_API const char* rocksdb_writebatch_data(
    rocksdb_writebatch_t*, size_t* size);

/* Block based table options */

extern ROCKSDB_LIBRARY_API rocksdb_block_based_table_options_t*
rocksdb_block_based_options_create();
extern ROCKSDB_LIBRARY_API void rocksdb_block_based_options_destroy(
    rocksdb_block_based_table_options_t* options);
extern ROCKSDB_LIBRARY_API void rocksdb_block_based_options_set_block_size(
    rocksdb_block_based_table_options_t* options, size_t block_size);
extern ROCKSDB_LIBRARY_API void
rocksdb_block_based_options_set_block_size_deviation(
    rocksdb_block_based_table_options_t* options, int block_size_deviation);
extern ROCKSDB_LIBRARY_API void
rocksdb_block_based_options_set_block_restart_interval(
    rocksdb_block_based_table_options_t* options, int block_restart_interval);
extern ROCKSDB_LIBRARY_API void rocksdb_block_based_options_set_filter_policy(
    rocksdb_block_based_table_options_t* options,
    rocksdb_filterpolicy_t* filter_policy);
extern ROCKSDB_LIBRARY_API void rocksdb_block_based_options_set_no_block_cache(
    rocksdb_block_based_table_options_t* options, unsigned char no_block_cache);
extern ROCKSDB_LIBRARY_API void rocksdb_block_based_options_set_block_cache(
    rocksdb_block_based_table_options_t* options, rocksdb_cache_t* block_cache);
extern ROCKSDB_LIBRARY_API void
rocksdb_block_based_options_set_block_cache_compressed(
    rocksdb_block_based_table_options_t* options,
    rocksdb_cache_t* block_cache_compressed);
extern ROCKSDB_LIBRARY_API void
rocksdb_block_based_options_set_whole_key_filtering(
    rocksdb_block_based_table_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_block_based_options_set_format_version(
    rocksdb_block_based_table_options_t*, int);
enum {
  rocksdb_block_based_table_index_type_binary_search = 0,
  rocksdb_block_based_table_index_type_hash_search = 1,
};
extern ROCKSDB_LIBRARY_API void rocksdb_block_based_options_set_index_type(
    rocksdb_block_based_table_options_t*, int);  // uses one of the above enums
extern ROCKSDB_LIBRARY_API void
rocksdb_block_based_options_set_hash_index_allow_collision(
    rocksdb_block_based_table_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void
rocksdb_block_based_options_set_cache_index_and_filter_blocks(
    rocksdb_block_based_table_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_block_based_table_factory(
    rocksdb_options_t* opt, rocksdb_block_based_table_options_t* table_options);

/* Cuckoo table options */

extern ROCKSDB_LIBRARY_API rocksdb_cuckoo_table_options_t*
rocksdb_cuckoo_options_create();
extern ROCKSDB_LIBRARY_API void rocksdb_cuckoo_options_destroy(
    rocksdb_cuckoo_table_options_t* options);
extern ROCKSDB_LIBRARY_API void rocksdb_cuckoo_options_set_hash_ratio(
    rocksdb_cuckoo_table_options_t* options, double v);
extern ROCKSDB_LIBRARY_API void rocksdb_cuckoo_options_set_max_search_depth(
    rocksdb_cuckoo_table_options_t* options, uint32_t v);
extern ROCKSDB_LIBRARY_API void rocksdb_cuckoo_options_set_cuckoo_block_size(
    rocksdb_cuckoo_table_options_t* options, uint32_t v);
extern ROCKSDB_LIBRARY_API void
rocksdb_cuckoo_options_set_identity_as_first_hash(
    rocksdb_cuckoo_table_options_t* options, unsigned char v);
extern ROCKSDB_LIBRARY_API void rocksdb_cuckoo_options_set_use_module_hash(
    rocksdb_cuckoo_table_options_t* options, unsigned char v);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_cuckoo_table_factory(
    rocksdb_options_t* opt, rocksdb_cuckoo_table_options_t* table_options);

/* Options */

extern ROCKSDB_LIBRARY_API rocksdb_options_t* rocksdb_options_create();
extern ROCKSDB_LIBRARY_API void rocksdb_options_destroy(rocksdb_options_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_options_increase_parallelism(
    rocksdb_options_t* opt, int total_threads);
extern ROCKSDB_LIBRARY_API void rocksdb_options_optimize_for_point_lookup(
    rocksdb_options_t* opt, uint64_t block_cache_size_mb);
extern ROCKSDB_LIBRARY_API void rocksdb_options_optimize_level_style_compaction(
    rocksdb_options_t* opt, uint64_t memtable_memory_budget);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_optimize_universal_style_compaction(
    rocksdb_options_t* opt, uint64_t memtable_memory_budget);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_compaction_filter(
    rocksdb_options_t*, rocksdb_compactionfilter_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_compaction_filter_factory(
    rocksdb_options_t*, rocksdb_compactionfilterfactory_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_comparator(
    rocksdb_options_t*, rocksdb_comparator_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_merge_operator(
    rocksdb_options_t*, rocksdb_mergeoperator_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_uint64add_merge_operator(
    rocksdb_options_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_compression_per_level(
    rocksdb_options_t* opt, int* level_values, size_t num_levels);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_create_if_missing(
    rocksdb_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_create_missing_column_families(rocksdb_options_t*,
                                                   unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_error_if_exists(
    rocksdb_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_paranoid_checks(
    rocksdb_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_env(rocksdb_options_t*,
                                                        rocksdb_env_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_info_log(rocksdb_options_t*,
                                                             rocksdb_logger_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_info_log_level(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_write_buffer_size(
    rocksdb_options_t*, size_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_max_open_files(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_max_total_wal_size(
    rocksdb_options_t* opt, uint64_t n);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_compression_options(
    rocksdb_options_t*, int, int, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_prefix_extractor(
    rocksdb_options_t*, rocksdb_slicetransform_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_num_levels(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_level0_file_num_compaction_trigger(rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_level0_slowdown_writes_trigger(rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_level0_stop_writes_trigger(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_max_mem_compaction_level(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_target_file_size_base(
    rocksdb_options_t*, uint64_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_target_file_size_multiplier(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_max_bytes_for_level_base(
    rocksdb_options_t*, uint64_t);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_max_bytes_for_level_multiplier(rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_expanded_compaction_factor(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_max_grandparent_overlap_factor(rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_max_bytes_for_level_multiplier_additional(
    rocksdb_options_t*, int* level_values, size_t num_levels);
extern ROCKSDB_LIBRARY_API void rocksdb_options_enable_statistics(
    rocksdb_options_t*);

/* returns a pointer to a malloc()-ed, null terminated string */
extern ROCKSDB_LIBRARY_API char* rocksdb_options_statistics_get_string(
    rocksdb_options_t* opt);

extern ROCKSDB_LIBRARY_API void rocksdb_options_set_max_write_buffer_number(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_min_write_buffer_number_to_merge(rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_max_write_buffer_number_to_maintain(rocksdb_options_t*,
                                                        int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_max_background_compactions(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_max_background_flushes(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_max_log_file_size(
    rocksdb_options_t*, size_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_log_file_time_to_roll(
    rocksdb_options_t*, size_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_keep_log_file_num(
    rocksdb_options_t*, size_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_soft_rate_limit(
    rocksdb_options_t*, double);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_hard_rate_limit(
    rocksdb_options_t*, double);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_rate_limit_delay_max_milliseconds(rocksdb_options_t*,
                                                      unsigned int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_max_manifest_file_size(
    rocksdb_options_t*, size_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_no_block_cache(
    rocksdb_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_table_cache_numshardbits(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_table_cache_remove_scan_count_limit(rocksdb_options_t*,
                                                        int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_arena_block_size(
    rocksdb_options_t*, size_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_use_fsync(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_db_log_dir(
    rocksdb_options_t*, const char*);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_wal_dir(rocksdb_options_t*,
                                                            const char*);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_WAL_ttl_seconds(
    rocksdb_options_t*, uint64_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_WAL_size_limit_MB(
    rocksdb_options_t*, uint64_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_manifest_preallocation_size(
    rocksdb_options_t*, size_t);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_purge_redundant_kvs_while_flush(rocksdb_options_t*,
                                                    unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_allow_os_buffer(
    rocksdb_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_allow_mmap_reads(
    rocksdb_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_allow_mmap_writes(
    rocksdb_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_is_fd_close_on_exec(
    rocksdb_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_skip_log_error_on_recovery(
    rocksdb_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_stats_dump_period_sec(
    rocksdb_options_t*, unsigned int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_block_size_deviation(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_advise_random_on_open(
    rocksdb_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_access_hint_on_compaction_start(rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_use_adaptive_mutex(
    rocksdb_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_bytes_per_sync(
    rocksdb_options_t*, uint64_t);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_verify_checksums_in_compaction(rocksdb_options_t*,
                                                   unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_filter_deletes(
    rocksdb_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_max_sequential_skip_in_iterations(rocksdb_options_t*,
                                                      uint64_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_disable_data_sync(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_disable_auto_compactions(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_delete_obsolete_files_period_micros(rocksdb_options_t*,
                                                        uint64_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_source_compaction_factor(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_options_prepare_for_bulk_load(
    rocksdb_options_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_memtable_vector_rep(
    rocksdb_options_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_hash_skip_list_rep(
    rocksdb_options_t*, size_t, int32_t, int32_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_hash_link_list_rep(
    rocksdb_options_t*, size_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_plain_table_factory(
    rocksdb_options_t*, uint32_t, int, double, size_t);

extern ROCKSDB_LIBRARY_API void rocksdb_options_set_min_level_to_compress(
    rocksdb_options_t* opt, int level);

extern ROCKSDB_LIBRARY_API void rocksdb_options_set_memtable_prefix_bloom_bits(
    rocksdb_options_t*, uint32_t);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_memtable_prefix_bloom_probes(rocksdb_options_t*, uint32_t);

extern ROCKSDB_LIBRARY_API void rocksdb_options_set_max_successive_merges(
    rocksdb_options_t*, size_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_min_partial_merge_operands(
    rocksdb_options_t*, uint32_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_bloom_locality(
    rocksdb_options_t*, uint32_t);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_inplace_update_support(
    rocksdb_options_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_inplace_update_num_locks(
    rocksdb_options_t*, size_t);

enum {
  rocksdb_no_compression = 0,
  rocksdb_snappy_compression = 1,
  rocksdb_zlib_compression = 2,
  rocksdb_bz2_compression = 3,
  rocksdb_lz4_compression = 4,
  rocksdb_lz4hc_compression = 5
};
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_compression(
    rocksdb_options_t*, int);

enum {
  rocksdb_level_compaction = 0,
  rocksdb_universal_compaction = 1,
  rocksdb_fifo_compaction = 2
};
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_compaction_style(
    rocksdb_options_t*, int);
extern ROCKSDB_LIBRARY_API void
rocksdb_options_set_universal_compaction_options(
    rocksdb_options_t*, rocksdb_universal_compaction_options_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_options_set_fifo_compaction_options(
    rocksdb_options_t* opt, rocksdb_fifo_compaction_options_t* fifo);

/* Compaction Filter */

extern ROCKSDB_LIBRARY_API rocksdb_compactionfilter_t*
rocksdb_compactionfilter_create(
    void* state, void (*destructor)(void*),
    unsigned char (*filter)(void*, int level, const char* key,
                            size_t key_length, const char* existing_value,
                            size_t value_length, char** new_value,
                            size_t* new_value_length,
                            unsigned char* value_changed),
    const char* (*name)(void*));
extern ROCKSDB_LIBRARY_API void rocksdb_compactionfilter_destroy(
    rocksdb_compactionfilter_t*);

/* Compaction Filter Context */

extern ROCKSDB_LIBRARY_API unsigned char
rocksdb_compactionfiltercontext_is_full_compaction(
    rocksdb_compactionfiltercontext_t* context);

extern ROCKSDB_LIBRARY_API unsigned char
rocksdb_compactionfiltercontext_is_manual_compaction(
    rocksdb_compactionfiltercontext_t* context);

/* Compaction Filter Factory */

extern ROCKSDB_LIBRARY_API rocksdb_compactionfilterfactory_t*
rocksdb_compactionfilterfactory_create(
    void* state, void (*destructor)(void*),
    rocksdb_compactionfilter_t* (*create_compaction_filter)(
        void*, rocksdb_compactionfiltercontext_t* context),
    const char* (*name)(void*));
extern ROCKSDB_LIBRARY_API void rocksdb_compactionfilterfactory_destroy(
    rocksdb_compactionfilterfactory_t*);

/* Comparator */

extern ROCKSDB_LIBRARY_API rocksdb_comparator_t* rocksdb_comparator_create(
    void* state, void (*destructor)(void*),
    int (*compare)(void*, const char* a, size_t alen, const char* b,
                   size_t blen),
    const char* (*name)(void*));
extern ROCKSDB_LIBRARY_API void rocksdb_comparator_destroy(
    rocksdb_comparator_t*);

/* Filter policy */

extern ROCKSDB_LIBRARY_API rocksdb_filterpolicy_t* rocksdb_filterpolicy_create(
    void* state, void (*destructor)(void*),
    char* (*create_filter)(void*, const char* const* key_array,
                           const size_t* key_length_array, int num_keys,
                           size_t* filter_length),
    unsigned char (*key_may_match)(void*, const char* key, size_t length,
                                   const char* filter, size_t filter_length),
    void (*delete_filter)(void*, const char* filter, size_t filter_length),
    const char* (*name)(void*));
extern ROCKSDB_LIBRARY_API void rocksdb_filterpolicy_destroy(
    rocksdb_filterpolicy_t*);

extern ROCKSDB_LIBRARY_API rocksdb_filterpolicy_t*
rocksdb_filterpolicy_create_bloom(int bits_per_key);

/* Merge Operator */

extern ROCKSDB_LIBRARY_API rocksdb_mergeoperator_t*
rocksdb_mergeoperator_create(
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
    const char* (*name)(void*));
extern ROCKSDB_LIBRARY_API void rocksdb_mergeoperator_destroy(
    rocksdb_mergeoperator_t*);

/* Read options */

extern ROCKSDB_LIBRARY_API rocksdb_readoptions_t* rocksdb_readoptions_create();
extern ROCKSDB_LIBRARY_API void rocksdb_readoptions_destroy(
    rocksdb_readoptions_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_readoptions_set_verify_checksums(
    rocksdb_readoptions_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_readoptions_set_fill_cache(
    rocksdb_readoptions_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_readoptions_set_snapshot(
    rocksdb_readoptions_t*, const rocksdb_snapshot_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_readoptions_set_iterate_upper_bound(
    rocksdb_readoptions_t*, const char* key, size_t keylen);
extern ROCKSDB_LIBRARY_API void rocksdb_readoptions_set_read_tier(
    rocksdb_readoptions_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_readoptions_set_tailing(
    rocksdb_readoptions_t*, unsigned char);

/* Write options */

extern ROCKSDB_LIBRARY_API rocksdb_writeoptions_t*
rocksdb_writeoptions_create();
extern ROCKSDB_LIBRARY_API void rocksdb_writeoptions_destroy(
    rocksdb_writeoptions_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_writeoptions_set_sync(
    rocksdb_writeoptions_t*, unsigned char);
extern ROCKSDB_LIBRARY_API void rocksdb_writeoptions_disable_WAL(
    rocksdb_writeoptions_t* opt, int disable);

/* Flush options */

extern ROCKSDB_LIBRARY_API rocksdb_flushoptions_t*
rocksdb_flushoptions_create();
extern ROCKSDB_LIBRARY_API void rocksdb_flushoptions_destroy(
    rocksdb_flushoptions_t*);
extern ROCKSDB_LIBRARY_API void rocksdb_flushoptions_set_wait(
    rocksdb_flushoptions_t*, unsigned char);

/* Cache */

extern ROCKSDB_LIBRARY_API rocksdb_cache_t* rocksdb_cache_create_lru(
    size_t capacity);
extern ROCKSDB_LIBRARY_API void rocksdb_cache_destroy(rocksdb_cache_t* cache);

/* Env */

extern ROCKSDB_LIBRARY_API rocksdb_env_t* rocksdb_create_default_env();
extern ROCKSDB_LIBRARY_API void rocksdb_env_set_background_threads(
    rocksdb_env_t* env, int n);
extern ROCKSDB_LIBRARY_API void
rocksdb_env_set_high_priority_background_threads(rocksdb_env_t* env, int n);
extern ROCKSDB_LIBRARY_API void rocksdb_env_join_all_threads(
    rocksdb_env_t* env);
extern ROCKSDB_LIBRARY_API void rocksdb_env_destroy(rocksdb_env_t*);

/* SliceTransform */

extern ROCKSDB_LIBRARY_API rocksdb_slicetransform_t*
rocksdb_slicetransform_create(
    void* state, void (*destructor)(void*),
    char* (*transform)(void*, const char* key, size_t length,
                       size_t* dst_length),
    unsigned char (*in_domain)(void*, const char* key, size_t length),
    unsigned char (*in_range)(void*, const char* key, size_t length),
    const char* (*name)(void*));
extern ROCKSDB_LIBRARY_API rocksdb_slicetransform_t*
    rocksdb_slicetransform_create_fixed_prefix(size_t);
extern ROCKSDB_LIBRARY_API rocksdb_slicetransform_t*
rocksdb_slicetransform_create_noop();
extern ROCKSDB_LIBRARY_API void rocksdb_slicetransform_destroy(
    rocksdb_slicetransform_t*);

/* Universal Compaction options */

enum {
  rocksdb_similar_size_compaction_stop_style = 0,
  rocksdb_total_size_compaction_stop_style = 1
};

extern ROCKSDB_LIBRARY_API rocksdb_universal_compaction_options_t*
rocksdb_universal_compaction_options_create();
extern ROCKSDB_LIBRARY_API void
rocksdb_universal_compaction_options_set_size_ratio(
    rocksdb_universal_compaction_options_t*, int);
extern ROCKSDB_LIBRARY_API void
rocksdb_universal_compaction_options_set_min_merge_width(
    rocksdb_universal_compaction_options_t*, int);
extern ROCKSDB_LIBRARY_API void
rocksdb_universal_compaction_options_set_max_merge_width(
    rocksdb_universal_compaction_options_t*, int);
extern ROCKSDB_LIBRARY_API void
rocksdb_universal_compaction_options_set_max_size_amplification_percent(
    rocksdb_universal_compaction_options_t*, int);
extern ROCKSDB_LIBRARY_API void
rocksdb_universal_compaction_options_set_compression_size_percent(
    rocksdb_universal_compaction_options_t*, int);
extern ROCKSDB_LIBRARY_API void
rocksdb_universal_compaction_options_set_stop_style(
    rocksdb_universal_compaction_options_t*, int);
extern ROCKSDB_LIBRARY_API void rocksdb_universal_compaction_options_destroy(
    rocksdb_universal_compaction_options_t*);

extern ROCKSDB_LIBRARY_API rocksdb_fifo_compaction_options_t*
rocksdb_fifo_compaction_options_create();
extern ROCKSDB_LIBRARY_API void
rocksdb_fifo_compaction_options_set_max_table_files_size(
    rocksdb_fifo_compaction_options_t* fifo_opts, uint64_t size);
extern ROCKSDB_LIBRARY_API void rocksdb_fifo_compaction_options_destroy(
    rocksdb_fifo_compaction_options_t* fifo_opts);

extern ROCKSDB_LIBRARY_API int rocksdb_livefiles_count(
    const rocksdb_livefiles_t*);
extern ROCKSDB_LIBRARY_API const char* rocksdb_livefiles_name(
    const rocksdb_livefiles_t*, int index);
extern ROCKSDB_LIBRARY_API int rocksdb_livefiles_level(
    const rocksdb_livefiles_t*, int index);
extern ROCKSDB_LIBRARY_API size_t
rocksdb_livefiles_size(const rocksdb_livefiles_t*, int index);
extern ROCKSDB_LIBRARY_API const char* rocksdb_livefiles_smallestkey(
    const rocksdb_livefiles_t*, int index, size_t* size);
extern ROCKSDB_LIBRARY_API const char* rocksdb_livefiles_largestkey(
    const rocksdb_livefiles_t*, int index, size_t* size);
extern ROCKSDB_LIBRARY_API void rocksdb_livefiles_destroy(
    const rocksdb_livefiles_t*);

/* Utility Helpers */

extern ROCKSDB_LIBRARY_API void rocksdb_get_options_from_string(
    const rocksdb_options_t* base_options, const char* opts_str,
    rocksdb_options_t* new_options, char** errptr);

// referring to convention (3), this should be used by client
// to free memory that was malloc()ed
extern ROCKSDB_LIBRARY_API void rocksdb_free(void* ptr);

#ifdef __cplusplus
}  /* end extern "C" */
#endif

#endif  /* STORAGE_ROCKSDB_INCLUDE_C_H_ */
