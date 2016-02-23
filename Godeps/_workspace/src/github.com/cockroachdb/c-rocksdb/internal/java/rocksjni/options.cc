// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// This file implements the "bridge" between Java and C++ for rocksdb::Options.

#include <stdio.h>
#include <stdlib.h>
#include <jni.h>
#include <strings.h>
#include <memory>

#include "include/org_rocksdb_Options.h"
#include "include/org_rocksdb_DBOptions.h"
#include "include/org_rocksdb_ColumnFamilyOptions.h"
#include "include/org_rocksdb_WriteOptions.h"
#include "include/org_rocksdb_ReadOptions.h"
#include "include/org_rocksdb_ComparatorOptions.h"
#include "include/org_rocksdb_FlushOptions.h"

#include "rocksjni/comparatorjnicallback.h"
#include "rocksjni/portal.h"

#include "rocksdb/db.h"
#include "rocksdb/options.h"
#include "rocksdb/statistics.h"
#include "rocksdb/memtablerep.h"
#include "rocksdb/table.h"
#include "rocksdb/slice_transform.h"
#include "rocksdb/rate_limiter.h"
#include "rocksdb/comparator.h"
#include "rocksdb/convenience.h"
#include "rocksdb/merge_operator.h"
#include "utilities/merge_operators.h"

/*
 * Class:     org_rocksdb_Options
 * Method:    newOptions
 * Signature: ()V
 */
void Java_org_rocksdb_Options_newOptions__(JNIEnv* env, jobject jobj) {
  rocksdb::Options* op = new rocksdb::Options();
  rocksdb::OptionsJni::setHandle(env, jobj, op);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    newOptions
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_newOptions__JJ(JNIEnv* env, jobject jobj,
    jlong jdboptions, jlong jcfoptions) {
  auto dbOpt = reinterpret_cast<const rocksdb::DBOptions*>(jdboptions);
  auto cfOpt = reinterpret_cast<const rocksdb::ColumnFamilyOptions*>(
      jcfoptions);
  rocksdb::Options* op = new rocksdb::Options(*dbOpt, *cfOpt);
  rocksdb::OptionsJni::setHandle(env, jobj, op);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    disposeInternal
 * Signature: (J)V
 */
void Java_org_rocksdb_Options_disposeInternal(
    JNIEnv* env, jobject jobj, jlong handle) {
  delete reinterpret_cast<rocksdb::Options*>(handle);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setIncreaseParallelism
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setIncreaseParallelism(
    JNIEnv * evnv, jobject jobj, jlong jhandle, jint totalThreads) {
  reinterpret_cast<rocksdb::Options*>
      (jhandle)->IncreaseParallelism(static_cast<int>(totalThreads));
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setCreateIfMissing
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setCreateIfMissing(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean flag) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->create_if_missing = flag;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    createIfMissing
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_createIfMissing(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->create_if_missing;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setCreateMissingColumnFamilies
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setCreateMissingColumnFamilies(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean flag) {
  reinterpret_cast<rocksdb::Options*>
      (jhandle)->create_missing_column_families = flag;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    createMissingColumnFamilies
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_createMissingColumnFamilies(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>
      (jhandle)->create_missing_column_families;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setComparatorHandle
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setComparatorHandle__JI(
    JNIEnv* env, jobject jobj, jlong jhandle, jint builtinComparator) {
  switch (builtinComparator) {
    case 1:
      reinterpret_cast<rocksdb::Options*>(jhandle)->comparator =
          rocksdb::ReverseBytewiseComparator();
      break;
    default:
      reinterpret_cast<rocksdb::Options*>(jhandle)->comparator =
          rocksdb::BytewiseComparator();
      break;
  }
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setComparatorHandle
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setComparatorHandle__JJ(
    JNIEnv* env, jobject jobj, jlong jopt_handle, jlong jcomparator_handle) {
  reinterpret_cast<rocksdb::Options*>(jopt_handle)->comparator =
      reinterpret_cast<rocksdb::Comparator*>(jcomparator_handle);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMergeOperatorName
 * Signature: (JJjava/lang/String)V
 */
void Java_org_rocksdb_Options_setMergeOperatorName(
    JNIEnv* env, jobject jobj, jlong jhandle, jstring jop_name) {
  auto options = reinterpret_cast<rocksdb::Options*>(jhandle);
  const char* op_name = env->GetStringUTFChars(jop_name, 0);
  options->merge_operator = rocksdb::MergeOperators::CreateFromStringId(
        op_name);
  env->ReleaseStringUTFChars(jop_name, op_name);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMergeOperator
 * Signature: (JJjava/lang/String)V
 */
void Java_org_rocksdb_Options_setMergeOperator(
  JNIEnv* env, jobject jobj, jlong jhandle, jlong mergeOperatorHandle) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->merge_operator =
    *(reinterpret_cast<std::shared_ptr<rocksdb::MergeOperator>*>
      (mergeOperatorHandle));
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setWriteBufferSize
 * Signature: (JJ)I
 */
void Java_org_rocksdb_Options_setWriteBufferSize(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jwrite_buffer_size) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(jwrite_buffer_size);
  if (s.ok()) {
    reinterpret_cast<rocksdb::Options*>(jhandle)->write_buffer_size =
        jwrite_buffer_size;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_Options
 * Method:    writeBufferSize
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_writeBufferSize(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->write_buffer_size;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMaxWriteBufferNumber
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setMaxWriteBufferNumber(
    JNIEnv* env, jobject jobj, jlong jhandle, jint jmax_write_buffer_number) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->max_write_buffer_number =
          jmax_write_buffer_number;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    createStatistics
 * Signature: (J)V
 */
void Java_org_rocksdb_Options_createStatistics(
    JNIEnv* env, jobject jobj, jlong jOptHandle) {
  reinterpret_cast<rocksdb::Options*>(jOptHandle)->statistics =
      rocksdb::CreateDBStatistics();
}

/*
 * Class:     org_rocksdb_Options
 * Method:    statisticsPtr
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_statisticsPtr(
    JNIEnv* env, jobject jobj, jlong jOptHandle) {
  auto st = reinterpret_cast<rocksdb::Options*>(jOptHandle)->statistics.get();
  return reinterpret_cast<jlong>(st);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    maxWriteBufferNumber
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_maxWriteBufferNumber(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->max_write_buffer_number;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    errorIfExists
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_errorIfExists(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->error_if_exists;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setErrorIfExists
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setErrorIfExists(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean error_if_exists) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->error_if_exists =
      static_cast<bool>(error_if_exists);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    paranoidChecks
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_paranoidChecks(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->paranoid_checks;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setParanoidChecks
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setParanoidChecks(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean paranoid_checks) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->paranoid_checks =
      static_cast<bool>(paranoid_checks);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setEnv
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setEnv(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jenv) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->env =
      reinterpret_cast<rocksdb::Env*>(jenv);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMaxTotalWalSize
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setMaxTotalWalSize(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong jmax_total_wal_size) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->max_total_wal_size =
      static_cast<jlong>(jmax_total_wal_size);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    maxTotalWalSize
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_maxTotalWalSize(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->
      max_total_wal_size;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    maxOpenFiles
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_maxOpenFiles(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->max_open_files;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMaxOpenFiles
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setMaxOpenFiles(
    JNIEnv* env, jobject jobj, jlong jhandle, jint max_open_files) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->max_open_files =
      static_cast<int>(max_open_files);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    disableDataSync
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_disableDataSync(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->disableDataSync;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setDisableDataSync
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setDisableDataSync(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean disableDataSync) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->disableDataSync =
      static_cast<bool>(disableDataSync);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    useFsync
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_useFsync(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->use_fsync;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setUseFsync
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setUseFsync(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean use_fsync) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->use_fsync =
      static_cast<bool>(use_fsync);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    dbLogDir
 * Signature: (J)Ljava/lang/String
 */
jstring Java_org_rocksdb_Options_dbLogDir(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return env->NewStringUTF(
      reinterpret_cast<rocksdb::Options*>(jhandle)->db_log_dir.c_str());
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setDbLogDir
 * Signature: (JLjava/lang/String)V
 */
void Java_org_rocksdb_Options_setDbLogDir(
    JNIEnv* env, jobject jobj, jlong jhandle, jstring jdb_log_dir) {
  const char* log_dir = env->GetStringUTFChars(jdb_log_dir, 0);
  reinterpret_cast<rocksdb::Options*>(jhandle)->db_log_dir.assign(log_dir);
  env->ReleaseStringUTFChars(jdb_log_dir, log_dir);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    walDir
 * Signature: (J)Ljava/lang/String
 */
jstring Java_org_rocksdb_Options_walDir(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return env->NewStringUTF(
      reinterpret_cast<rocksdb::Options*>(jhandle)->wal_dir.c_str());
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setWalDir
 * Signature: (JLjava/lang/String)V
 */
void Java_org_rocksdb_Options_setWalDir(
    JNIEnv* env, jobject jobj, jlong jhandle, jstring jwal_dir) {
  const char* wal_dir = env->GetStringUTFChars(jwal_dir, 0);
  reinterpret_cast<rocksdb::Options*>(jhandle)->wal_dir.assign(wal_dir);
  env->ReleaseStringUTFChars(jwal_dir, wal_dir);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    deleteObsoleteFilesPeriodMicros
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_deleteObsoleteFilesPeriodMicros(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)
      ->delete_obsolete_files_period_micros;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setDeleteObsoleteFilesPeriodMicros
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setDeleteObsoleteFilesPeriodMicros(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong micros) {
  reinterpret_cast<rocksdb::Options*>(jhandle)
      ->delete_obsolete_files_period_micros =
          static_cast<int64_t>(micros);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    maxBackgroundCompactions
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_maxBackgroundCompactions(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->max_background_compactions;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMaxBackgroundCompactions
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setMaxBackgroundCompactions(
    JNIEnv* env, jobject jobj, jlong jhandle, jint max) {
  reinterpret_cast<rocksdb::Options*>(jhandle)
      ->max_background_compactions = static_cast<int>(max);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    maxBackgroundFlushes
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_maxBackgroundFlushes(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->max_background_flushes;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMaxBackgroundFlushes
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setMaxBackgroundFlushes(
    JNIEnv* env, jobject jobj, jlong jhandle, jint max_background_flushes) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->max_background_flushes =
      static_cast<int>(max_background_flushes);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    maxLogFileSize
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_maxLogFileSize(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->max_log_file_size;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMaxLogFileSize
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setMaxLogFileSize(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong max_log_file_size) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(max_log_file_size);
  if (s.ok()) {
    reinterpret_cast<rocksdb::Options*>(jhandle)->max_log_file_size =
        max_log_file_size;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_Options
 * Method:    logFileTimeToRoll
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_logFileTimeToRoll(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->log_file_time_to_roll;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setLogFileTimeToRoll
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setLogFileTimeToRoll(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong log_file_time_to_roll) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(
      log_file_time_to_roll);
  if (s.ok()) {
    reinterpret_cast<rocksdb::Options*>(jhandle)->log_file_time_to_roll =
        log_file_time_to_roll;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_Options
 * Method:    keepLogFileNum
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_keepLogFileNum(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->keep_log_file_num;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setKeepLogFileNum
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setKeepLogFileNum(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong keep_log_file_num) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(keep_log_file_num);
  if (s.ok()) {
    reinterpret_cast<rocksdb::Options*>(jhandle)->keep_log_file_num =
        keep_log_file_num;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_Options
 * Method:    maxManifestFileSize
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_maxManifestFileSize(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->max_manifest_file_size;
}

/*
 * Method:    memTableFactoryName
 * Signature: (J)Ljava/lang/String
 */
jstring Java_org_rocksdb_Options_memTableFactoryName(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto opt = reinterpret_cast<rocksdb::Options*>(jhandle);
  rocksdb::MemTableRepFactory* tf = opt->memtable_factory.get();

  // Should never be nullptr.
  // Default memtable factory is SkipListFactory
  assert(tf);

  // temporarly fix for the historical typo
  if (strcmp(tf->Name(), "HashLinkListRepFactory") == 0) {
    return env->NewStringUTF("HashLinkedListRepFactory");
  }

  return env->NewStringUTF(tf->Name());
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMaxManifestFileSize
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setMaxManifestFileSize(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong max_manifest_file_size) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->max_manifest_file_size =
      static_cast<int64_t>(max_manifest_file_size);
}

/*
 * Method:    setMemTableFactory
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setMemTableFactory(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jfactory_handle) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->memtable_factory.reset(
      reinterpret_cast<rocksdb::MemTableRepFactory*>(jfactory_handle));
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setRateLimiter
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setRateLimiter(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jrate_limiter_handle) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->rate_limiter.reset(
      reinterpret_cast<rocksdb::RateLimiter*>(jrate_limiter_handle));
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setLogger
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setLogger(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jlogger_handle) {
std::shared_ptr<rocksdb::LoggerJniCallback> *pLogger =
      reinterpret_cast<std::shared_ptr<rocksdb::LoggerJniCallback> *>(
          jlogger_handle);
  reinterpret_cast<rocksdb::Options*>(jhandle)->info_log = *pLogger;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setInfoLogLevel
 * Signature: (JB)V
 */
void Java_org_rocksdb_Options_setInfoLogLevel(
    JNIEnv* env, jobject jobj, jlong jhandle, jbyte jlog_level) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->info_log_level =
      static_cast<rocksdb::InfoLogLevel>(jlog_level);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    infoLogLevel
 * Signature: (J)B
 */
jbyte Java_org_rocksdb_Options_infoLogLevel(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return static_cast<jbyte>(
      reinterpret_cast<rocksdb::Options*>(jhandle)->info_log_level);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    tableCacheNumshardbits
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_tableCacheNumshardbits(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->table_cache_numshardbits;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setTableCacheNumshardbits
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setTableCacheNumshardbits(
    JNIEnv* env, jobject jobj, jlong jhandle, jint table_cache_numshardbits) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->table_cache_numshardbits =
      static_cast<int>(table_cache_numshardbits);
}

/*
 * Method:    useFixedLengthPrefixExtractor
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_useFixedLengthPrefixExtractor(
    JNIEnv* env, jobject jobj, jlong jhandle, jint jprefix_length) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->prefix_extractor.reset(
      rocksdb::NewFixedPrefixTransform(
          static_cast<int>(jprefix_length)));
}

/*
 * Method:    useCappedPrefixExtractor
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_useCappedPrefixExtractor(
    JNIEnv* env, jobject jobj, jlong jhandle, jint jprefix_length) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->prefix_extractor.reset(
      rocksdb::NewCappedPrefixTransform(
          static_cast<int>(jprefix_length)));
}

/*
 * Class:     org_rocksdb_Options
 * Method:    walTtlSeconds
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_walTtlSeconds(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->WAL_ttl_seconds;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setWalTtlSeconds
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setWalTtlSeconds(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong WAL_ttl_seconds) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->WAL_ttl_seconds =
      static_cast<int64_t>(WAL_ttl_seconds);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    walTtlSeconds
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_walSizeLimitMB(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->WAL_size_limit_MB;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setWalSizeLimitMB
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setWalSizeLimitMB(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong WAL_size_limit_MB) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->WAL_size_limit_MB =
      static_cast<int64_t>(WAL_size_limit_MB);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    manifestPreallocationSize
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_manifestPreallocationSize(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)
      ->manifest_preallocation_size;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setManifestPreallocationSize
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setManifestPreallocationSize(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong preallocation_size) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(preallocation_size);
  if (s.ok()) {
    reinterpret_cast<rocksdb::Options*>(jhandle)->manifest_preallocation_size =
        preallocation_size;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_Options
 * Method:    allowOsBuffer
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_allowOsBuffer(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->allow_os_buffer;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setAllowOsBuffer
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setAllowOsBuffer(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean allow_os_buffer) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->allow_os_buffer =
      static_cast<bool>(allow_os_buffer);
}

/*
 * Method:    setTableFactory
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setTableFactory(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jfactory_handle) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->table_factory.reset(
      reinterpret_cast<rocksdb::TableFactory*>(jfactory_handle));
}

/*
 * Class:     org_rocksdb_Options
 * Method:    allowMmapReads
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_allowMmapReads(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->allow_mmap_reads;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setAllowMmapReads
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setAllowMmapReads(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean allow_mmap_reads) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->allow_mmap_reads =
      static_cast<bool>(allow_mmap_reads);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    allowMmapWrites
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_allowMmapWrites(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->allow_mmap_writes;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setAllowMmapWrites
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setAllowMmapWrites(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean allow_mmap_writes) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->allow_mmap_writes =
      static_cast<bool>(allow_mmap_writes);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    isFdCloseOnExec
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_isFdCloseOnExec(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->is_fd_close_on_exec;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setIsFdCloseOnExec
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setIsFdCloseOnExec(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean is_fd_close_on_exec) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->is_fd_close_on_exec =
      static_cast<bool>(is_fd_close_on_exec);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    statsDumpPeriodSec
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_statsDumpPeriodSec(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->stats_dump_period_sec;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setStatsDumpPeriodSec
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setStatsDumpPeriodSec(
    JNIEnv* env, jobject jobj, jlong jhandle, jint stats_dump_period_sec) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->stats_dump_period_sec =
      static_cast<int>(stats_dump_period_sec);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    adviseRandomOnOpen
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_adviseRandomOnOpen(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->advise_random_on_open;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setAdviseRandomOnOpen
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setAdviseRandomOnOpen(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean advise_random_on_open) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->advise_random_on_open =
      static_cast<bool>(advise_random_on_open);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    useAdaptiveMutex
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_useAdaptiveMutex(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->use_adaptive_mutex;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setUseAdaptiveMutex
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setUseAdaptiveMutex(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean use_adaptive_mutex) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->use_adaptive_mutex =
      static_cast<bool>(use_adaptive_mutex);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    bytesPerSync
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_bytesPerSync(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->bytes_per_sync;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setBytesPerSync
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setBytesPerSync(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong bytes_per_sync) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->bytes_per_sync =
      static_cast<int64_t>(bytes_per_sync);
}

/*
 * Method:    tableFactoryName
 * Signature: (J)Ljava/lang/String
 */
jstring Java_org_rocksdb_Options_tableFactoryName(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto opt = reinterpret_cast<rocksdb::Options*>(jhandle);
  rocksdb::TableFactory* tf = opt->table_factory.get();

  // Should never be nullptr.
  // Default memtable factory is SkipListFactory
  assert(tf);

  return env->NewStringUTF(tf->Name());
}


/*
 * Class:     org_rocksdb_Options
 * Method:    minWriteBufferNumberToMerge
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_minWriteBufferNumberToMerge(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->min_write_buffer_number_to_merge;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMinWriteBufferNumberToMerge
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setMinWriteBufferNumberToMerge(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jmin_write_buffer_number_to_merge) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->min_write_buffer_number_to_merge =
          static_cast<int>(jmin_write_buffer_number_to_merge);
}
/*
 * Class:     org_rocksdb_Options
 * Method:    maxWriteBufferNumberToMaintain
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_maxWriteBufferNumberToMaintain(JNIEnv* env,
                                                             jobject jobj,
                                                             jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)
      ->max_write_buffer_number_to_maintain;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMaxWriteBufferNumberToMaintain
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setMaxWriteBufferNumberToMaintain(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jmax_write_buffer_number_to_maintain) {
  reinterpret_cast<rocksdb::Options*>(jhandle)
      ->max_write_buffer_number_to_maintain =
      static_cast<int>(jmax_write_buffer_number_to_maintain);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setCompressionType
 * Signature: (JB)V
 */
void Java_org_rocksdb_Options_setCompressionType(
    JNIEnv* env, jobject jobj, jlong jhandle, jbyte compression) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->compression =
      static_cast<rocksdb::CompressionType>(compression);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    compressionType
 * Signature: (J)B
 */
jbyte Java_org_rocksdb_Options_compressionType(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->compression;
}

/*
 * Helper method to convert a Java list to a CompressionType
 * vector.
 */
std::vector<rocksdb::CompressionType> rocksdb_compression_vector_helper(
    JNIEnv* env, jobject jcompressionLevels) {
  std::vector<rocksdb::CompressionType> compressionLevels;
  // iterate over compressionLevels
  jobject iteratorObj = env->CallObjectMethod(
        jcompressionLevels, rocksdb::ListJni::getIteratorMethod(env));
  while (env->CallBooleanMethod(
    iteratorObj, rocksdb::ListJni::getHasNextMethod(env)) == JNI_TRUE) {
    // get compression
    jobject jcompression_obj = env->CallObjectMethod(iteratorObj,
        rocksdb::ListJni::getNextMethod(env));
    jbyte jcompression = env->CallByteMethod(jcompression_obj,
        rocksdb::ByteJni::getByteValueMethod(env));
    compressionLevels.push_back(static_cast<rocksdb::CompressionType>(
        jcompression));
  }
  return compressionLevels;
}

/*
 * Helper method to convert a CompressionType vector to a Java
 * List.
 */
jobject rocksdb_compression_list_helper(JNIEnv* env,
    std::vector<rocksdb::CompressionType> compressionLevels) {
  jclass jListClazz = env->FindClass("java/util/ArrayList");
  jmethodID midList = rocksdb::ListJni::getArrayListConstructorMethodId(
      env, jListClazz);
  jobject jcompressionLevels = env->NewObject(jListClazz,
    midList, compressionLevels.size());
  // insert in java list
  for (std::vector<rocksdb::CompressionType>::size_type i = 0;
        i != compressionLevels.size(); i++) {
    jclass jByteClazz = env->FindClass("java/lang/Byte");
    jmethodID midByte = env->GetMethodID(jByteClazz, "<init>", "(B)V");
    jobject obj = env->NewObject(jByteClazz, midByte,
        compressionLevels[i]);
    env->CallBooleanMethod(jcompressionLevels,
        rocksdb::ListJni::getListAddMethodId(env), obj);
  }
  return jcompressionLevels;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setCompressionPerLevel
 * Signature: (JLjava/util/List;)V
 */
void Java_org_rocksdb_Options_setCompressionPerLevel(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jobject jcompressionLevels) {
  auto* options = reinterpret_cast<rocksdb::Options*>(jhandle);
  std::vector<rocksdb::CompressionType> compressionLevels =
      rocksdb_compression_vector_helper(env, jcompressionLevels);
  options->compression_per_level = compressionLevels;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    compressionPerLevel
 * Signature: (J)Ljava/util/List;
 */
jobject Java_org_rocksdb_Options_compressionPerLevel(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto* options = reinterpret_cast<rocksdb::Options*>(jhandle);
  return rocksdb_compression_list_helper(env,
      options->compression_per_level);
}


/*
 * Class:     org_rocksdb_Options
 * Method:    setCompactionStyle
 * Signature: (JB)V
 */
void Java_org_rocksdb_Options_setCompactionStyle(
    JNIEnv* env, jobject jobj, jlong jhandle, jbyte compaction_style) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->compaction_style =
      static_cast<rocksdb::CompactionStyle>(compaction_style);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    compactionStyle
 * Signature: (J)B
 */
jbyte Java_org_rocksdb_Options_compactionStyle(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->compaction_style;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMaxTableFilesSizeFIFO
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setMaxTableFilesSizeFIFO(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jmax_table_files_size) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->compaction_options_fifo.max_table_files_size =
    static_cast<long>(jmax_table_files_size);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    maxTableFilesSizeFIFO
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_maxTableFilesSizeFIFO(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->compaction_options_fifo.max_table_files_size;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    numLevels
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_numLevels(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->num_levels;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setNumLevels
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setNumLevels(
    JNIEnv* env, jobject jobj, jlong jhandle, jint jnum_levels) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->num_levels =
      static_cast<int>(jnum_levels);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    levelZeroFileNumCompactionTrigger
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_levelZeroFileNumCompactionTrigger(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->level0_file_num_compaction_trigger;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setLevelZeroFileNumCompactionTrigger
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setLevelZeroFileNumCompactionTrigger(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jlevel0_file_num_compaction_trigger) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->level0_file_num_compaction_trigger =
          static_cast<int>(jlevel0_file_num_compaction_trigger);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    levelZeroSlowdownWritesTrigger
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_levelZeroSlowdownWritesTrigger(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->level0_slowdown_writes_trigger;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setLevelSlowdownWritesTrigger
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setLevelZeroSlowdownWritesTrigger(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jlevel0_slowdown_writes_trigger) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->level0_slowdown_writes_trigger =
          static_cast<int>(jlevel0_slowdown_writes_trigger);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    levelZeroStopWritesTrigger
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_levelZeroStopWritesTrigger(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->level0_stop_writes_trigger;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setLevelStopWritesTrigger
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setLevelZeroStopWritesTrigger(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jlevel0_stop_writes_trigger) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->level0_stop_writes_trigger =
      static_cast<int>(jlevel0_stop_writes_trigger);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    targetFileSizeBase
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_targetFileSizeBase(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->target_file_size_base;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setTargetFileSizeBase
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setTargetFileSizeBase(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong jtarget_file_size_base) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->target_file_size_base =
      static_cast<uint64_t>(jtarget_file_size_base);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    targetFileSizeMultiplier
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_targetFileSizeMultiplier(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->target_file_size_multiplier;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setTargetFileSizeMultiplier
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setTargetFileSizeMultiplier(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jtarget_file_size_multiplier) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->target_file_size_multiplier =
          static_cast<int>(jtarget_file_size_multiplier);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    maxBytesForLevelBase
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_maxBytesForLevelBase(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->max_bytes_for_level_base;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMaxBytesForLevelBase
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setMaxBytesForLevelBase(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong jmax_bytes_for_level_base) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->max_bytes_for_level_base =
          static_cast<int64_t>(jmax_bytes_for_level_base);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    levelCompactionDynamicLevelBytes
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_levelCompactionDynamicLevelBytes(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->level_compaction_dynamic_level_bytes;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setLevelCompactionDynamicLevelBytes
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setLevelCompactionDynamicLevelBytes(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jboolean jenable_dynamic_level_bytes) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->level_compaction_dynamic_level_bytes =
          (jenable_dynamic_level_bytes);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    maxBytesForLevelMultiplier
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_maxBytesForLevelMultiplier(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->max_bytes_for_level_multiplier;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMaxBytesForLevelMultiplier
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setMaxBytesForLevelMultiplier(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jmax_bytes_for_level_multiplier) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->max_bytes_for_level_multiplier =
          static_cast<int>(jmax_bytes_for_level_multiplier);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    expandedCompactionFactor
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_expandedCompactionFactor(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->expanded_compaction_factor;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setExpandedCompactionFactor
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setExpandedCompactionFactor(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jexpanded_compaction_factor) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->expanded_compaction_factor =
          static_cast<int>(jexpanded_compaction_factor);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    sourceCompactionFactor
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_sourceCompactionFactor(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->source_compaction_factor;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setSourceCompactionFactor
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setSourceCompactionFactor(
    JNIEnv* env, jobject jobj, jlong jhandle,
        jint jsource_compaction_factor) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->source_compaction_factor =
          static_cast<int>(jsource_compaction_factor);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    maxGrandparentOverlapFactor
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_maxGrandparentOverlapFactor(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->max_grandparent_overlap_factor;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMaxGrandparentOverlapFactor
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setMaxGrandparentOverlapFactor(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jmax_grandparent_overlap_factor) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->max_grandparent_overlap_factor =
          static_cast<int>(jmax_grandparent_overlap_factor);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    softRateLimit
 * Signature: (J)D
 */
jdouble Java_org_rocksdb_Options_softRateLimit(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->soft_rate_limit;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setSoftRateLimit
 * Signature: (JD)V
 */
void Java_org_rocksdb_Options_setSoftRateLimit(
    JNIEnv* env, jobject jobj, jlong jhandle, jdouble jsoft_rate_limit) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->soft_rate_limit =
      static_cast<double>(jsoft_rate_limit);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    hardRateLimit
 * Signature: (J)D
 */
jdouble Java_org_rocksdb_Options_hardRateLimit(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->hard_rate_limit;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setHardRateLimit
 * Signature: (JD)V
 */
void Java_org_rocksdb_Options_setHardRateLimit(
    JNIEnv* env, jobject jobj, jlong jhandle, jdouble jhard_rate_limit) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->hard_rate_limit =
      static_cast<double>(jhard_rate_limit);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    rateLimitDelayMaxMilliseconds
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_rateLimitDelayMaxMilliseconds(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->rate_limit_delay_max_milliseconds;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setRateLimitDelayMaxMilliseconds
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setRateLimitDelayMaxMilliseconds(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jrate_limit_delay_max_milliseconds) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->rate_limit_delay_max_milliseconds =
          static_cast<int>(jrate_limit_delay_max_milliseconds);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    arenaBlockSize
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_arenaBlockSize(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->arena_block_size;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setArenaBlockSize
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setArenaBlockSize(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jarena_block_size) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(jarena_block_size);
  if (s.ok()) {
    reinterpret_cast<rocksdb::Options*>(jhandle)->arena_block_size =
        jarena_block_size;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_Options
 * Method:    disableAutoCompactions
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_disableAutoCompactions(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->disable_auto_compactions;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setDisableAutoCompactions
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setDisableAutoCompactions(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jboolean jdisable_auto_compactions) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->disable_auto_compactions =
          static_cast<bool>(jdisable_auto_compactions);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    purgeRedundantKvsWhileFlush
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_purgeRedundantKvsWhileFlush(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->purge_redundant_kvs_while_flush;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setPurgeRedundantKvsWhileFlush
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setPurgeRedundantKvsWhileFlush(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jboolean jpurge_redundant_kvs_while_flush) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->purge_redundant_kvs_while_flush =
          static_cast<bool>(jpurge_redundant_kvs_while_flush);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    verifyChecksumsInCompaction
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_verifyChecksumsInCompaction(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->verify_checksums_in_compaction;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setVerifyChecksumsInCompaction
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setVerifyChecksumsInCompaction(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jboolean jverify_checksums_in_compaction) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->verify_checksums_in_compaction =
          static_cast<bool>(jverify_checksums_in_compaction);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    filterDeletes
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_filterDeletes(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->filter_deletes;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setFilterDeletes
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setFilterDeletes(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean jfilter_deletes) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->filter_deletes =
      static_cast<bool>(jfilter_deletes);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    maxSequentialSkipInIterations
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_maxSequentialSkipInIterations(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->max_sequential_skip_in_iterations;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMaxSequentialSkipInIterations
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setMaxSequentialSkipInIterations(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong jmax_sequential_skip_in_iterations) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->max_sequential_skip_in_iterations =
          static_cast<int64_t>(jmax_sequential_skip_in_iterations);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    inplaceUpdateSupport
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_inplaceUpdateSupport(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->inplace_update_support;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setInplaceUpdateSupport
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setInplaceUpdateSupport(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jboolean jinplace_update_support) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->inplace_update_support =
          static_cast<bool>(jinplace_update_support);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    inplaceUpdateNumLocks
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_inplaceUpdateNumLocks(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->inplace_update_num_locks;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setInplaceUpdateNumLocks
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setInplaceUpdateNumLocks(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong jinplace_update_num_locks) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(
      jinplace_update_num_locks);
  if (s.ok()) {
    reinterpret_cast<rocksdb::Options*>(jhandle)->inplace_update_num_locks =
        jinplace_update_num_locks;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_Options
 * Method:    memtablePrefixBloomBits
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_memtablePrefixBloomBits(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->memtable_prefix_bloom_bits;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMemtablePrefixBloomBits
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setMemtablePrefixBloomBits(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jmemtable_prefix_bloom_bits) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->memtable_prefix_bloom_bits =
          static_cast<int32_t>(jmemtable_prefix_bloom_bits);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    memtablePrefixBloomProbes
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_memtablePrefixBloomProbes(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->memtable_prefix_bloom_probes;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMemtablePrefixBloomProbes
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setMemtablePrefixBloomProbes(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jmemtable_prefix_bloom_probes) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->memtable_prefix_bloom_probes =
          static_cast<int32_t>(jmemtable_prefix_bloom_probes);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    bloomLocality
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_bloomLocality(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->bloom_locality;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setBloomLocality
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setBloomLocality(
    JNIEnv* env, jobject jobj, jlong jhandle, jint jbloom_locality) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->bloom_locality =
      static_cast<int32_t>(jbloom_locality);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    maxSuccessiveMerges
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Options_maxSuccessiveMerges(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(jhandle)->max_successive_merges;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMaxSuccessiveMerges
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_setMaxSuccessiveMerges(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong jmax_successive_merges) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(
      jmax_successive_merges);
  if (s.ok()) {
    reinterpret_cast<rocksdb::Options*>(jhandle)->max_successive_merges =
        jmax_successive_merges;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_Options
 * Method:    minPartialMergeOperands
 * Signature: (J)I
 */
jint Java_org_rocksdb_Options_minPartialMergeOperands(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->min_partial_merge_operands;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setMinPartialMergeOperands
 * Signature: (JI)V
 */
void Java_org_rocksdb_Options_setMinPartialMergeOperands(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jmin_partial_merge_operands) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->min_partial_merge_operands =
          static_cast<int32_t>(jmin_partial_merge_operands);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    optimizeFiltersForHits
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_Options_optimizeFiltersForHits(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::Options*>(
      jhandle)->optimize_filters_for_hits;
}

/*
 * Class:     org_rocksdb_Options
 * Method:    setOptimizeFiltersForHits
 * Signature: (JZ)V
 */
void Java_org_rocksdb_Options_setOptimizeFiltersForHits(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jboolean joptimize_filters_for_hits) {
  reinterpret_cast<rocksdb::Options*>(
      jhandle)->optimize_filters_for_hits =
          static_cast<bool>(joptimize_filters_for_hits);
}

/*
 * Method:    optimizeForPointLookup
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_optimizeForPointLookup(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong block_cache_size_mb) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->
      OptimizeForPointLookup(block_cache_size_mb);
}

/*
 * Method:    optimizeLevelStyleCompaction
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_optimizeLevelStyleCompaction(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong memtable_memory_budget) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->
      OptimizeLevelStyleCompaction(memtable_memory_budget);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    optimizeUniversalStyleCompaction
 * Signature: (JJ)V
 */
void Java_org_rocksdb_Options_optimizeUniversalStyleCompaction(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong memtable_memory_budget) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->
      OptimizeUniversalStyleCompaction(memtable_memory_budget);
}

/*
 * Class:     org_rocksdb_Options
 * Method:    prepareForBulkLoad
 * Signature: (J)V
 */
void Java_org_rocksdb_Options_prepareForBulkLoad(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  reinterpret_cast<rocksdb::Options*>(jhandle)->
      PrepareForBulkLoad();
}

//////////////////////////////////////////////////////////////////////////////
// rocksdb::ColumnFamilyOptions

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    newColumnFamilyOptions
 * Signature: ()V
 */
void Java_org_rocksdb_ColumnFamilyOptions_newColumnFamilyOptions(
    JNIEnv* env, jobject jobj) {
  rocksdb::ColumnFamilyOptions* op = new rocksdb::ColumnFamilyOptions();
  rocksdb::ColumnFamilyOptionsJni::setHandle(env, jobj, op);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    getColumnFamilyOptionsFromProps
 * Signature: (Ljava/util/String;)J
 */
jlong Java_org_rocksdb_ColumnFamilyOptions_getColumnFamilyOptionsFromProps(
    JNIEnv* env, jclass jclazz, jstring jopt_string) {
  jlong ret_value = 0;
  rocksdb::ColumnFamilyOptions* cf_options =
      new rocksdb::ColumnFamilyOptions();
  const char* opt_string = env->GetStringUTFChars(jopt_string, 0);
  rocksdb::Status status = rocksdb::GetColumnFamilyOptionsFromString(
      rocksdb::ColumnFamilyOptions(), opt_string, cf_options);
  env->ReleaseStringUTFChars(jopt_string, opt_string);
  // Check if ColumnFamilyOptions creation was possible.
  if (status.ok()) {
    ret_value = reinterpret_cast<jlong>(cf_options);
  } else {
    // if operation failed the ColumnFamilyOptions need to be deleted
    // again to prevent a memory leak.
    delete cf_options;
  }
  return ret_value;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    disposeInternal
 * Signature: (J)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_disposeInternal(
    JNIEnv* env, jobject jobj, jlong handle) {
  delete reinterpret_cast<rocksdb::ColumnFamilyOptions*>(handle);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    optimizeForPointLookup
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_optimizeForPointLookup(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong block_cache_size_mb) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      OptimizeForPointLookup(block_cache_size_mb);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    optimizeLevelStyleCompaction
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_optimizeLevelStyleCompaction(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong memtable_memory_budget) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      OptimizeLevelStyleCompaction(memtable_memory_budget);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    optimizeUniversalStyleCompaction
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_optimizeUniversalStyleCompaction(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong memtable_memory_budget) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      OptimizeUniversalStyleCompaction(memtable_memory_budget);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setComparatorHandle
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setComparatorHandle__JI(
    JNIEnv* env, jobject jobj, jlong jhandle, jint builtinComparator) {
  switch (builtinComparator) {
    case 1:
      reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->comparator =
          rocksdb::ReverseBytewiseComparator();
      break;
    default:
      reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->comparator =
          rocksdb::BytewiseComparator();
      break;
  }
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setComparatorHandle
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setComparatorHandle__JJ(
    JNIEnv* env, jobject jobj, jlong jopt_handle, jlong jcomparator_handle) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jopt_handle)->comparator =
      reinterpret_cast<rocksdb::Comparator*>(jcomparator_handle);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMergeOperatorName
 * Signature: (JJjava/lang/String)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMergeOperatorName(
    JNIEnv* env, jobject jobj, jlong jhandle, jstring jop_name) {
  auto options = reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle);
  const char* op_name = env->GetStringUTFChars(jop_name, 0);
  options->merge_operator = rocksdb::MergeOperators::CreateFromStringId(
        op_name);
  env->ReleaseStringUTFChars(jop_name, op_name);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMergeOperator
 * Signature: (JJjava/lang/String)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMergeOperator(
  JNIEnv* env, jobject jobj, jlong jhandle, jlong mergeOperatorHandle) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->merge_operator =
    *(reinterpret_cast<std::shared_ptr<rocksdb::MergeOperator>*>
      (mergeOperatorHandle));
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setCompactionFilterHandle
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setCompactionFilterHandle__JJ(
    JNIEnv* env, jobject jobj, jlong jopt_handle,
    jlong jcompactionfilter_handle) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jopt_handle)->
      compaction_filter = reinterpret_cast<rocksdb::CompactionFilter*>
        (jcompactionfilter_handle);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setWriteBufferSize
 * Signature: (JJ)I
 */
void Java_org_rocksdb_ColumnFamilyOptions_setWriteBufferSize(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jwrite_buffer_size) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(jwrite_buffer_size);
  if (s.ok()) {
    reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
        write_buffer_size = jwrite_buffer_size;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    writeBufferSize
 * Signature: (J)J
 */
jlong Java_org_rocksdb_ColumnFamilyOptions_writeBufferSize(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      write_buffer_size;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMaxWriteBufferNumber
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMaxWriteBufferNumber(
    JNIEnv* env, jobject jobj, jlong jhandle, jint jmax_write_buffer_number) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      max_write_buffer_number = jmax_write_buffer_number;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    maxWriteBufferNumber
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_maxWriteBufferNumber(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      max_write_buffer_number;
}

/*
 * Method:    setMemTableFactory
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMemTableFactory(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jfactory_handle) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      memtable_factory.reset(
      reinterpret_cast<rocksdb::MemTableRepFactory*>(jfactory_handle));
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    memTableFactoryName
 * Signature: (J)Ljava/lang/String
 */
jstring Java_org_rocksdb_ColumnFamilyOptions_memTableFactoryName(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto opt = reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle);
  rocksdb::MemTableRepFactory* tf = opt->memtable_factory.get();

  // Should never be nullptr.
  // Default memtable factory is SkipListFactory
  assert(tf);

  // temporarly fix for the historical typo
  if (strcmp(tf->Name(), "HashLinkListRepFactory") == 0) {
    return env->NewStringUTF("HashLinkedListRepFactory");
  }

  return env->NewStringUTF(tf->Name());
}

/*
 * Method:    useFixedLengthPrefixExtractor
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_useFixedLengthPrefixExtractor(
    JNIEnv* env, jobject jobj, jlong jhandle, jint jprefix_length) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      prefix_extractor.reset(rocksdb::NewFixedPrefixTransform(
          static_cast<int>(jprefix_length)));
}

/*
 * Method:    useCappedPrefixExtractor
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_useCappedPrefixExtractor(
    JNIEnv* env, jobject jobj, jlong jhandle, jint jprefix_length) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      prefix_extractor.reset(rocksdb::NewCappedPrefixTransform(
          static_cast<int>(jprefix_length)));
}

/*
 * Method:    setTableFactory
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setTableFactory(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jfactory_handle) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      table_factory.reset(reinterpret_cast<rocksdb::TableFactory*>(
      jfactory_handle));
}

/*
 * Method:    tableFactoryName
 * Signature: (J)Ljava/lang/String
 */
jstring Java_org_rocksdb_ColumnFamilyOptions_tableFactoryName(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto opt = reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle);
  rocksdb::TableFactory* tf = opt->table_factory.get();

  // Should never be nullptr.
  // Default memtable factory is SkipListFactory
  assert(tf);

  return env->NewStringUTF(tf->Name());
}


/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    minWriteBufferNumberToMerge
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_minWriteBufferNumberToMerge(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->min_write_buffer_number_to_merge;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMinWriteBufferNumberToMerge
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMinWriteBufferNumberToMerge(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jmin_write_buffer_number_to_merge) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->min_write_buffer_number_to_merge =
          static_cast<int>(jmin_write_buffer_number_to_merge);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    maxWriteBufferNumberToMaintain
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_maxWriteBufferNumberToMaintain(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)
      ->max_write_buffer_number_to_maintain;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMaxWriteBufferNumberToMaintain
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMaxWriteBufferNumberToMaintain(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jmax_write_buffer_number_to_maintain) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)
      ->max_write_buffer_number_to_maintain =
      static_cast<int>(jmax_write_buffer_number_to_maintain);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setCompressionType
 * Signature: (JB)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setCompressionType(
    JNIEnv* env, jobject jobj, jlong jhandle, jbyte compression) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      compression = static_cast<rocksdb::CompressionType>(compression);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    compressionType
 * Signature: (J)B
 */
jbyte Java_org_rocksdb_ColumnFamilyOptions_compressionType(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      compression;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setCompressionPerLevel
 * Signature: (JLjava/util/List;)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setCompressionPerLevel(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jobject jcompressionLevels) {
  auto* options = reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle);
  std::vector<rocksdb::CompressionType> compressionLevels =
      rocksdb_compression_vector_helper(env, jcompressionLevels);
  options->compression_per_level = compressionLevels;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    compressionPerLevel
 * Signature: (J)Ljava/util/List;
 */
jobject Java_org_rocksdb_ColumnFamilyOptions_compressionPerLevel(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto* options = reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle);
  return rocksdb_compression_list_helper(env,
      options->compression_per_level);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setCompactionStyle
 * Signature: (JB)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setCompactionStyle(
    JNIEnv* env, jobject jobj, jlong jhandle, jbyte compaction_style) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->compaction_style =
      static_cast<rocksdb::CompactionStyle>(compaction_style);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    compactionStyle
 * Signature: (J)B
 */
jbyte Java_org_rocksdb_ColumnFamilyOptions_compactionStyle(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>
      (jhandle)->compaction_style;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMaxTableFilesSizeFIFO
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMaxTableFilesSizeFIFO(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jmax_table_files_size) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->compaction_options_fifo.max_table_files_size =
    static_cast<long>(jmax_table_files_size);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    maxTableFilesSizeFIFO
 * Signature: (J)J
 */
jlong Java_org_rocksdb_ColumnFamilyOptions_maxTableFilesSizeFIFO(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->compaction_options_fifo.max_table_files_size;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    numLevels
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_numLevels(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->num_levels;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setNumLevels
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setNumLevels(
    JNIEnv* env, jobject jobj, jlong jhandle, jint jnum_levels) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->num_levels =
      static_cast<int>(jnum_levels);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    levelZeroFileNumCompactionTrigger
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_levelZeroFileNumCompactionTrigger(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->level0_file_num_compaction_trigger;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setLevelZeroFileNumCompactionTrigger
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setLevelZeroFileNumCompactionTrigger(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jlevel0_file_num_compaction_trigger) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->level0_file_num_compaction_trigger =
          static_cast<int>(jlevel0_file_num_compaction_trigger);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    levelZeroSlowdownWritesTrigger
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_levelZeroSlowdownWritesTrigger(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->level0_slowdown_writes_trigger;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setLevelSlowdownWritesTrigger
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setLevelZeroSlowdownWritesTrigger(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jlevel0_slowdown_writes_trigger) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->level0_slowdown_writes_trigger =
          static_cast<int>(jlevel0_slowdown_writes_trigger);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    levelZeroStopWritesTrigger
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_levelZeroStopWritesTrigger(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->level0_stop_writes_trigger;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setLevelStopWritesTrigger
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setLevelZeroStopWritesTrigger(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jlevel0_stop_writes_trigger) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      level0_stop_writes_trigger = static_cast<int>(
      jlevel0_stop_writes_trigger);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    maxMemCompactionLevel
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_maxMemCompactionLevel(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return 0;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMaxMemCompactionLevel
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMaxMemCompactionLevel(
    JNIEnv* env, jobject jobj, jlong jhandle, jint jmax_mem_compaction_level) {}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    targetFileSizeBase
 * Signature: (J)J
 */
jlong Java_org_rocksdb_ColumnFamilyOptions_targetFileSizeBase(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      target_file_size_base;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setTargetFileSizeBase
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setTargetFileSizeBase(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong jtarget_file_size_base) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      target_file_size_base = static_cast<uint64_t>(jtarget_file_size_base);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    targetFileSizeMultiplier
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_targetFileSizeMultiplier(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->target_file_size_multiplier;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setTargetFileSizeMultiplier
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setTargetFileSizeMultiplier(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jtarget_file_size_multiplier) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->target_file_size_multiplier =
          static_cast<int>(jtarget_file_size_multiplier);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    maxBytesForLevelBase
 * Signature: (J)J
 */
jlong Java_org_rocksdb_ColumnFamilyOptions_maxBytesForLevelBase(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->max_bytes_for_level_base;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMaxBytesForLevelBase
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMaxBytesForLevelBase(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong jmax_bytes_for_level_base) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->max_bytes_for_level_base =
          static_cast<int64_t>(jmax_bytes_for_level_base);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    levelCompactionDynamicLevelBytes
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_ColumnFamilyOptions_levelCompactionDynamicLevelBytes(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->level_compaction_dynamic_level_bytes;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setLevelCompactionDynamicLevelBytes
 * Signature: (JZ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setLevelCompactionDynamicLevelBytes(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jboolean jenable_dynamic_level_bytes) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->level_compaction_dynamic_level_bytes =
          (jenable_dynamic_level_bytes);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    maxBytesForLevelMultiplier
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_maxBytesForLevelMultiplier(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->max_bytes_for_level_multiplier;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMaxBytesForLevelMultiplier
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMaxBytesForLevelMultiplier(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jmax_bytes_for_level_multiplier) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->max_bytes_for_level_multiplier =
          static_cast<int>(jmax_bytes_for_level_multiplier);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    expandedCompactionFactor
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_expandedCompactionFactor(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->expanded_compaction_factor;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setExpandedCompactionFactor
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setExpandedCompactionFactor(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jexpanded_compaction_factor) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->expanded_compaction_factor =
          static_cast<int>(jexpanded_compaction_factor);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    sourceCompactionFactor
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_sourceCompactionFactor(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->source_compaction_factor;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setSourceCompactionFactor
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setSourceCompactionFactor(
    JNIEnv* env, jobject jobj, jlong jhandle,
        jint jsource_compaction_factor) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->source_compaction_factor =
          static_cast<int>(jsource_compaction_factor);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    maxGrandparentOverlapFactor
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_maxGrandparentOverlapFactor(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->max_grandparent_overlap_factor;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMaxGrandparentOverlapFactor
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMaxGrandparentOverlapFactor(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jmax_grandparent_overlap_factor) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->max_grandparent_overlap_factor =
          static_cast<int>(jmax_grandparent_overlap_factor);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    softRateLimit
 * Signature: (J)D
 */
jdouble Java_org_rocksdb_ColumnFamilyOptions_softRateLimit(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      soft_rate_limit;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setSoftRateLimit
 * Signature: (JD)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setSoftRateLimit(
    JNIEnv* env, jobject jobj, jlong jhandle, jdouble jsoft_rate_limit) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->soft_rate_limit =
      static_cast<double>(jsoft_rate_limit);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    hardRateLimit
 * Signature: (J)D
 */
jdouble Java_org_rocksdb_ColumnFamilyOptions_hardRateLimit(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      hard_rate_limit;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setHardRateLimit
 * Signature: (JD)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setHardRateLimit(
    JNIEnv* env, jobject jobj, jlong jhandle, jdouble jhard_rate_limit) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->hard_rate_limit =
      static_cast<double>(jhard_rate_limit);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    rateLimitDelayMaxMilliseconds
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_rateLimitDelayMaxMilliseconds(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->rate_limit_delay_max_milliseconds;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setRateLimitDelayMaxMilliseconds
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setRateLimitDelayMaxMilliseconds(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jrate_limit_delay_max_milliseconds) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->rate_limit_delay_max_milliseconds =
          static_cast<int>(jrate_limit_delay_max_milliseconds);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    arenaBlockSize
 * Signature: (J)J
 */
jlong Java_org_rocksdb_ColumnFamilyOptions_arenaBlockSize(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      arena_block_size;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setArenaBlockSize
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setArenaBlockSize(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jarena_block_size) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(jarena_block_size);
  if (s.ok()) {
    reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
        arena_block_size = jarena_block_size;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    disableAutoCompactions
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_ColumnFamilyOptions_disableAutoCompactions(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->disable_auto_compactions;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setDisableAutoCompactions
 * Signature: (JZ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setDisableAutoCompactions(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jboolean jdisable_auto_compactions) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->disable_auto_compactions =
          static_cast<bool>(jdisable_auto_compactions);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    purgeRedundantKvsWhileFlush
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_ColumnFamilyOptions_purgeRedundantKvsWhileFlush(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->purge_redundant_kvs_while_flush;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setPurgeRedundantKvsWhileFlush
 * Signature: (JZ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setPurgeRedundantKvsWhileFlush(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jboolean jpurge_redundant_kvs_while_flush) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->purge_redundant_kvs_while_flush =
          static_cast<bool>(jpurge_redundant_kvs_while_flush);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    verifyChecksumsInCompaction
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_ColumnFamilyOptions_verifyChecksumsInCompaction(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->verify_checksums_in_compaction;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setVerifyChecksumsInCompaction
 * Signature: (JZ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setVerifyChecksumsInCompaction(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jboolean jverify_checksums_in_compaction) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->verify_checksums_in_compaction =
          static_cast<bool>(jverify_checksums_in_compaction);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    filterDeletes
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_ColumnFamilyOptions_filterDeletes(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      filter_deletes;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setFilterDeletes
 * Signature: (JZ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setFilterDeletes(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean jfilter_deletes) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->filter_deletes =
      static_cast<bool>(jfilter_deletes);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    maxSequentialSkipInIterations
 * Signature: (J)J
 */
jlong Java_org_rocksdb_ColumnFamilyOptions_maxSequentialSkipInIterations(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->max_sequential_skip_in_iterations;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMaxSequentialSkipInIterations
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMaxSequentialSkipInIterations(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong jmax_sequential_skip_in_iterations) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->max_sequential_skip_in_iterations =
          static_cast<int64_t>(jmax_sequential_skip_in_iterations);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    inplaceUpdateSupport
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_ColumnFamilyOptions_inplaceUpdateSupport(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->inplace_update_support;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setInplaceUpdateSupport
 * Signature: (JZ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setInplaceUpdateSupport(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jboolean jinplace_update_support) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->inplace_update_support =
          static_cast<bool>(jinplace_update_support);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    inplaceUpdateNumLocks
 * Signature: (J)J
 */
jlong Java_org_rocksdb_ColumnFamilyOptions_inplaceUpdateNumLocks(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->inplace_update_num_locks;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setInplaceUpdateNumLocks
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setInplaceUpdateNumLocks(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong jinplace_update_num_locks) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(
      jinplace_update_num_locks);
  if (s.ok()) {
    reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
        inplace_update_num_locks = jinplace_update_num_locks;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    memtablePrefixBloomBits
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_memtablePrefixBloomBits(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->memtable_prefix_bloom_bits;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMemtablePrefixBloomBits
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMemtablePrefixBloomBits(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jmemtable_prefix_bloom_bits) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->memtable_prefix_bloom_bits =
          static_cast<int32_t>(jmemtable_prefix_bloom_bits);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    memtablePrefixBloomProbes
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_memtablePrefixBloomProbes(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->memtable_prefix_bloom_probes;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMemtablePrefixBloomProbes
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMemtablePrefixBloomProbes(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jmemtable_prefix_bloom_probes) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->memtable_prefix_bloom_probes =
          static_cast<int32_t>(jmemtable_prefix_bloom_probes);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    bloomLocality
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_bloomLocality(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      bloom_locality;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setBloomLocality
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setBloomLocality(
    JNIEnv* env, jobject jobj, jlong jhandle, jint jbloom_locality) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->bloom_locality =
      static_cast<int32_t>(jbloom_locality);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    maxSuccessiveMerges
 * Signature: (J)J
 */
jlong Java_org_rocksdb_ColumnFamilyOptions_maxSuccessiveMerges(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
      max_successive_merges;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMaxSuccessiveMerges
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMaxSuccessiveMerges(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong jmax_successive_merges) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(
      jmax_successive_merges);
  if (s.ok()) {
    reinterpret_cast<rocksdb::ColumnFamilyOptions*>(jhandle)->
        max_successive_merges = jmax_successive_merges;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    minPartialMergeOperands
 * Signature: (J)I
 */
jint Java_org_rocksdb_ColumnFamilyOptions_minPartialMergeOperands(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->min_partial_merge_operands;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setMinPartialMergeOperands
 * Signature: (JI)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setMinPartialMergeOperands(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jint jmin_partial_merge_operands) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->min_partial_merge_operands =
          static_cast<int32_t>(jmin_partial_merge_operands);
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    optimizeFiltersForHits
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_ColumnFamilyOptions_optimizeFiltersForHits(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->optimize_filters_for_hits;
}

/*
 * Class:     org_rocksdb_ColumnFamilyOptions
 * Method:    setOptimizeFiltersForHits
 * Signature: (JZ)V
 */
void Java_org_rocksdb_ColumnFamilyOptions_setOptimizeFiltersForHits(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jboolean joptimize_filters_for_hits) {
  reinterpret_cast<rocksdb::ColumnFamilyOptions*>(
      jhandle)->optimize_filters_for_hits =
          static_cast<bool>(joptimize_filters_for_hits);
}

/////////////////////////////////////////////////////////////////////
// rocksdb::DBOptions

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    newDBOptions
 * Signature: ()V
 */
void Java_org_rocksdb_DBOptions_newDBOptions(JNIEnv* env,
    jobject jobj) {
  rocksdb::DBOptions* dbop = new rocksdb::DBOptions();
  rocksdb::DBOptionsJni::setHandle(env, jobj, dbop);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    getDBOptionsFromProps
 * Signature: (Ljava/util/String;)J
 */
jlong Java_org_rocksdb_DBOptions_getDBOptionsFromProps(
    JNIEnv* env, jclass jclazz, jstring jopt_string) {
  jlong ret_value = 0;
  rocksdb::DBOptions* db_options =
      new rocksdb::DBOptions();
  const char* opt_string = env->GetStringUTFChars(jopt_string, 0);
  rocksdb::Status status = rocksdb::GetDBOptionsFromString(
      rocksdb::DBOptions(), opt_string, db_options);
  env->ReleaseStringUTFChars(jopt_string, opt_string);
  // Check if DBOptions creation was possible.
  if (status.ok()) {
    ret_value = reinterpret_cast<jlong>(db_options);
  } else {
    // if operation failed the DBOptions need to be deleted
    // again to prevent a memory leak.
    delete db_options;
  }
  return ret_value;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    disposeInternal
 * Signature: (J)V
 */
void Java_org_rocksdb_DBOptions_disposeInternal(
    JNIEnv* env, jobject jobj, jlong handle) {
  delete reinterpret_cast<rocksdb::DBOptions*>(handle);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setIncreaseParallelism
 * Signature: (JI)V
 */
void Java_org_rocksdb_DBOptions_setIncreaseParallelism(
    JNIEnv * env, jobject jobj, jlong jhandle, jint totalThreads) {
  reinterpret_cast<rocksdb::DBOptions*>
      (jhandle)->IncreaseParallelism(static_cast<int>(totalThreads));
}


/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setCreateIfMissing
 * Signature: (JZ)V
 */
void Java_org_rocksdb_DBOptions_setCreateIfMissing(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean flag) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->
      create_if_missing = flag;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    createIfMissing
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_DBOptions_createIfMissing(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->create_if_missing;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setCreateMissingColumnFamilies
 * Signature: (JZ)V
 */
void Java_org_rocksdb_DBOptions_setCreateMissingColumnFamilies(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean flag) {
  reinterpret_cast<rocksdb::DBOptions*>
      (jhandle)->create_missing_column_families = flag;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    createMissingColumnFamilies
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_DBOptions_createMissingColumnFamilies(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>
      (jhandle)->create_missing_column_families;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setErrorIfExists
 * Signature: (JZ)V
 */
void Java_org_rocksdb_DBOptions_setErrorIfExists(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean error_if_exists) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->error_if_exists =
      static_cast<bool>(error_if_exists);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    errorIfExists
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_DBOptions_errorIfExists(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->error_if_exists;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setParanoidChecks
 * Signature: (JZ)V
 */
void Java_org_rocksdb_DBOptions_setParanoidChecks(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean paranoid_checks) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->paranoid_checks =
      static_cast<bool>(paranoid_checks);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    paranoidChecks
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_DBOptions_paranoidChecks(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->paranoid_checks;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setRateLimiter
 * Signature: (JJ)V
 */
void Java_org_rocksdb_DBOptions_setRateLimiter(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jrate_limiter_handle) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->rate_limiter.reset(
      reinterpret_cast<rocksdb::RateLimiter*>(jrate_limiter_handle));
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setLogger
 * Signature: (JJ)V
 */
void Java_org_rocksdb_DBOptions_setLogger(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jlogger_handle) {
  std::shared_ptr<rocksdb::LoggerJniCallback> *pLogger =
      reinterpret_cast<std::shared_ptr<rocksdb::LoggerJniCallback> *>(
          jlogger_handle);
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->info_log = *pLogger;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setInfoLogLevel
 * Signature: (JB)V
 */
void Java_org_rocksdb_DBOptions_setInfoLogLevel(
    JNIEnv* env, jobject jobj, jlong jhandle, jbyte jlog_level) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->info_log_level =
    static_cast<rocksdb::InfoLogLevel>(jlog_level);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    infoLogLevel
 * Signature: (J)B
 */
jbyte Java_org_rocksdb_DBOptions_infoLogLevel(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return static_cast<jbyte>(
      reinterpret_cast<rocksdb::DBOptions*>(jhandle)->info_log_level);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setMaxTotalWalSize
 * Signature: (JJ)V
 */
void Java_org_rocksdb_DBOptions_setMaxTotalWalSize(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jlong jmax_total_wal_size) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->max_total_wal_size =
      static_cast<jlong>(jmax_total_wal_size);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    maxTotalWalSize
 * Signature: (J)J
 */
jlong Java_org_rocksdb_DBOptions_maxTotalWalSize(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->
      max_total_wal_size;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setMaxOpenFiles
 * Signature: (JI)V
 */
void Java_org_rocksdb_DBOptions_setMaxOpenFiles(
    JNIEnv* env, jobject jobj, jlong jhandle, jint max_open_files) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->max_open_files =
      static_cast<int>(max_open_files);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    maxOpenFiles
 * Signature: (J)I
 */
jint Java_org_rocksdb_DBOptions_maxOpenFiles(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->max_open_files;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    createStatistics
 * Signature: (J)V
 */
void Java_org_rocksdb_DBOptions_createStatistics(
    JNIEnv* env, jobject jobj, jlong jOptHandle) {
  reinterpret_cast<rocksdb::DBOptions*>(jOptHandle)->statistics =
      rocksdb::CreateDBStatistics();
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    statisticsPtr
 * Signature: (J)J
 */
jlong Java_org_rocksdb_DBOptions_statisticsPtr(
    JNIEnv* env, jobject jobj, jlong jOptHandle) {
  auto st = reinterpret_cast<rocksdb::DBOptions*>(jOptHandle)->
      statistics.get();
  return reinterpret_cast<jlong>(st);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setDisableDataSync
 * Signature: (JZ)V
 */
void Java_org_rocksdb_DBOptions_setDisableDataSync(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean disableDataSync) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->disableDataSync =
      static_cast<bool>(disableDataSync);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    disableDataSync
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_DBOptions_disableDataSync(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->disableDataSync;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setUseFsync
 * Signature: (JZ)V
 */
void Java_org_rocksdb_DBOptions_setUseFsync(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean use_fsync) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->use_fsync =
      static_cast<bool>(use_fsync);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    useFsync
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_DBOptions_useFsync(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->use_fsync;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setDbLogDir
 * Signature: (JLjava/lang/String)V
 */
void Java_org_rocksdb_DBOptions_setDbLogDir(
    JNIEnv* env, jobject jobj, jlong jhandle, jstring jdb_log_dir) {
  const char* log_dir = env->GetStringUTFChars(jdb_log_dir, 0);
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->db_log_dir.assign(log_dir);
  env->ReleaseStringUTFChars(jdb_log_dir, log_dir);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    dbLogDir
 * Signature: (J)Ljava/lang/String
 */
jstring Java_org_rocksdb_DBOptions_dbLogDir(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return env->NewStringUTF(
      reinterpret_cast<rocksdb::DBOptions*>(jhandle)->db_log_dir.c_str());
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setWalDir
 * Signature: (JLjava/lang/String)V
 */
void Java_org_rocksdb_DBOptions_setWalDir(
    JNIEnv* env, jobject jobj, jlong jhandle, jstring jwal_dir) {
  const char* wal_dir = env->GetStringUTFChars(jwal_dir, 0);
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->wal_dir.assign(wal_dir);
  env->ReleaseStringUTFChars(jwal_dir, wal_dir);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    walDir
 * Signature: (J)Ljava/lang/String
 */
jstring Java_org_rocksdb_DBOptions_walDir(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return env->NewStringUTF(
      reinterpret_cast<rocksdb::DBOptions*>(jhandle)->wal_dir.c_str());
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setDeleteObsoleteFilesPeriodMicros
 * Signature: (JJ)V
 */
void Java_org_rocksdb_DBOptions_setDeleteObsoleteFilesPeriodMicros(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong micros) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)
      ->delete_obsolete_files_period_micros =
          static_cast<int64_t>(micros);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    deleteObsoleteFilesPeriodMicros
 * Signature: (J)J
 */
jlong Java_org_rocksdb_DBOptions_deleteObsoleteFilesPeriodMicros(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)
      ->delete_obsolete_files_period_micros;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setMaxBackgroundCompactions
 * Signature: (JI)V
 */
void Java_org_rocksdb_DBOptions_setMaxBackgroundCompactions(
    JNIEnv* env, jobject jobj, jlong jhandle, jint max) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)
      ->max_background_compactions = static_cast<int>(max);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    maxBackgroundCompactions
 * Signature: (J)I
 */
jint Java_org_rocksdb_DBOptions_maxBackgroundCompactions(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(
      jhandle)->max_background_compactions;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setMaxBackgroundFlushes
 * Signature: (JI)V
 */
void Java_org_rocksdb_DBOptions_setMaxBackgroundFlushes(
    JNIEnv* env, jobject jobj, jlong jhandle, jint max_background_flushes) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->max_background_flushes =
      static_cast<int>(max_background_flushes);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    maxBackgroundFlushes
 * Signature: (J)I
 */
jint Java_org_rocksdb_DBOptions_maxBackgroundFlushes(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->
      max_background_flushes;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setMaxLogFileSize
 * Signature: (JJ)V
 */
void Java_org_rocksdb_DBOptions_setMaxLogFileSize(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong max_log_file_size) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(max_log_file_size);
  if (s.ok()) {
    reinterpret_cast<rocksdb::DBOptions*>(jhandle)->max_log_file_size =
        max_log_file_size;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    maxLogFileSize
 * Signature: (J)J
 */
jlong Java_org_rocksdb_DBOptions_maxLogFileSize(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->max_log_file_size;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setLogFileTimeToRoll
 * Signature: (JJ)V
 */
void Java_org_rocksdb_DBOptions_setLogFileTimeToRoll(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong log_file_time_to_roll) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(
      log_file_time_to_roll);
  if (s.ok()) {
    reinterpret_cast<rocksdb::DBOptions*>(jhandle)->log_file_time_to_roll =
        log_file_time_to_roll;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    logFileTimeToRoll
 * Signature: (J)J
 */
jlong Java_org_rocksdb_DBOptions_logFileTimeToRoll(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->log_file_time_to_roll;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setKeepLogFileNum
 * Signature: (JJ)V
 */
void Java_org_rocksdb_DBOptions_setKeepLogFileNum(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong keep_log_file_num) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(keep_log_file_num);
  if (s.ok()) {
    reinterpret_cast<rocksdb::DBOptions*>(jhandle)->keep_log_file_num =
        keep_log_file_num;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    keepLogFileNum
 * Signature: (J)J
 */
jlong Java_org_rocksdb_DBOptions_keepLogFileNum(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->keep_log_file_num;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setMaxManifestFileSize
 * Signature: (JJ)V
 */
void Java_org_rocksdb_DBOptions_setMaxManifestFileSize(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong max_manifest_file_size) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->max_manifest_file_size =
      static_cast<int64_t>(max_manifest_file_size);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    maxManifestFileSize
 * Signature: (J)J
 */
jlong Java_org_rocksdb_DBOptions_maxManifestFileSize(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->
      max_manifest_file_size;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setTableCacheNumshardbits
 * Signature: (JI)V
 */
void Java_org_rocksdb_DBOptions_setTableCacheNumshardbits(
    JNIEnv* env, jobject jobj, jlong jhandle, jint table_cache_numshardbits) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->table_cache_numshardbits =
      static_cast<int>(table_cache_numshardbits);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    tableCacheNumshardbits
 * Signature: (J)I
 */
jint Java_org_rocksdb_DBOptions_tableCacheNumshardbits(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->
      table_cache_numshardbits;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setWalTtlSeconds
 * Signature: (JJ)V
 */
void Java_org_rocksdb_DBOptions_setWalTtlSeconds(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong WAL_ttl_seconds) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->WAL_ttl_seconds =
      static_cast<int64_t>(WAL_ttl_seconds);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    walTtlSeconds
 * Signature: (J)J
 */
jlong Java_org_rocksdb_DBOptions_walTtlSeconds(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->WAL_ttl_seconds;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setWalSizeLimitMB
 * Signature: (JJ)V
 */
void Java_org_rocksdb_DBOptions_setWalSizeLimitMB(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong WAL_size_limit_MB) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->WAL_size_limit_MB =
      static_cast<int64_t>(WAL_size_limit_MB);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    walTtlSeconds
 * Signature: (J)J
 */
jlong Java_org_rocksdb_DBOptions_walSizeLimitMB(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->WAL_size_limit_MB;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setManifestPreallocationSize
 * Signature: (JJ)V
 */
void Java_org_rocksdb_DBOptions_setManifestPreallocationSize(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong preallocation_size) {
  rocksdb::Status s = rocksdb::check_if_jlong_fits_size_t(preallocation_size);
  if (s.ok()) {
    reinterpret_cast<rocksdb::DBOptions*>(jhandle)->
        manifest_preallocation_size = preallocation_size;
  } else {
    rocksdb::IllegalArgumentExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    manifestPreallocationSize
 * Signature: (J)J
 */
jlong Java_org_rocksdb_DBOptions_manifestPreallocationSize(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)
      ->manifest_preallocation_size;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setAllowOsBuffer
 * Signature: (JZ)V
 */
void Java_org_rocksdb_DBOptions_setAllowOsBuffer(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean allow_os_buffer) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->allow_os_buffer =
      static_cast<bool>(allow_os_buffer);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    allowOsBuffer
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_DBOptions_allowOsBuffer(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->allow_os_buffer;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setAllowMmapReads
 * Signature: (JZ)V
 */
void Java_org_rocksdb_DBOptions_setAllowMmapReads(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean allow_mmap_reads) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->allow_mmap_reads =
      static_cast<bool>(allow_mmap_reads);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    allowMmapReads
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_DBOptions_allowMmapReads(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->allow_mmap_reads;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setAllowMmapWrites
 * Signature: (JZ)V
 */
void Java_org_rocksdb_DBOptions_setAllowMmapWrites(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean allow_mmap_writes) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->allow_mmap_writes =
      static_cast<bool>(allow_mmap_writes);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    allowMmapWrites
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_DBOptions_allowMmapWrites(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->allow_mmap_writes;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setIsFdCloseOnExec
 * Signature: (JZ)V
 */
void Java_org_rocksdb_DBOptions_setIsFdCloseOnExec(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean is_fd_close_on_exec) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->is_fd_close_on_exec =
      static_cast<bool>(is_fd_close_on_exec);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    isFdCloseOnExec
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_DBOptions_isFdCloseOnExec(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->is_fd_close_on_exec;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setStatsDumpPeriodSec
 * Signature: (JI)V
 */
void Java_org_rocksdb_DBOptions_setStatsDumpPeriodSec(
    JNIEnv* env, jobject jobj, jlong jhandle, jint stats_dump_period_sec) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->stats_dump_period_sec =
      static_cast<int>(stats_dump_period_sec);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    statsDumpPeriodSec
 * Signature: (J)I
 */
jint Java_org_rocksdb_DBOptions_statsDumpPeriodSec(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->stats_dump_period_sec;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setAdviseRandomOnOpen
 * Signature: (JZ)V
 */
void Java_org_rocksdb_DBOptions_setAdviseRandomOnOpen(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean advise_random_on_open) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->advise_random_on_open =
      static_cast<bool>(advise_random_on_open);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    adviseRandomOnOpen
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_DBOptions_adviseRandomOnOpen(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->advise_random_on_open;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setUseAdaptiveMutex
 * Signature: (JZ)V
 */
void Java_org_rocksdb_DBOptions_setUseAdaptiveMutex(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean use_adaptive_mutex) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->use_adaptive_mutex =
      static_cast<bool>(use_adaptive_mutex);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    useAdaptiveMutex
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_DBOptions_useAdaptiveMutex(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->use_adaptive_mutex;
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    setBytesPerSync
 * Signature: (JJ)V
 */
void Java_org_rocksdb_DBOptions_setBytesPerSync(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong bytes_per_sync) {
  reinterpret_cast<rocksdb::DBOptions*>(jhandle)->bytes_per_sync =
      static_cast<int64_t>(bytes_per_sync);
}

/*
 * Class:     org_rocksdb_DBOptions
 * Method:    bytesPerSync
 * Signature: (J)J
 */
jlong Java_org_rocksdb_DBOptions_bytesPerSync(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::DBOptions*>(jhandle)->bytes_per_sync;
}

//////////////////////////////////////////////////////////////////////////////
// rocksdb::WriteOptions

/*
 * Class:     org_rocksdb_WriteOptions
 * Method:    newWriteOptions
 * Signature: ()V
 */
void Java_org_rocksdb_WriteOptions_newWriteOptions(
    JNIEnv* env, jobject jwrite_options) {
  rocksdb::WriteOptions* op = new rocksdb::WriteOptions();
  rocksdb::WriteOptionsJni::setHandle(env, jwrite_options, op);
}

/*
 * Class:     org_rocksdb_WriteOptions
 * Method:    disposeInternal
 * Signature: ()V
 */
void Java_org_rocksdb_WriteOptions_disposeInternal(
    JNIEnv* env, jobject jwrite_options, jlong jhandle) {
  auto write_options = reinterpret_cast<rocksdb::WriteOptions*>(jhandle);
  delete write_options;

  rocksdb::WriteOptionsJni::setHandle(env, jwrite_options, nullptr);
}

/*
 * Class:     org_rocksdb_WriteOptions
 * Method:    setSync
 * Signature: (JZ)V
 */
void Java_org_rocksdb_WriteOptions_setSync(
  JNIEnv* env, jobject jwrite_options, jlong jhandle, jboolean jflag) {
  reinterpret_cast<rocksdb::WriteOptions*>(jhandle)->sync = jflag;
}

/*
 * Class:     org_rocksdb_WriteOptions
 * Method:    sync
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_WriteOptions_sync(
    JNIEnv* env, jobject jwrite_options, jlong jhandle) {
  return reinterpret_cast<rocksdb::WriteOptions*>(jhandle)->sync;
}

/*
 * Class:     org_rocksdb_WriteOptions
 * Method:    setDisableWAL
 * Signature: (JZ)V
 */
void Java_org_rocksdb_WriteOptions_setDisableWAL(
    JNIEnv* env, jobject jwrite_options, jlong jhandle, jboolean jflag) {
  reinterpret_cast<rocksdb::WriteOptions*>(jhandle)->disableWAL = jflag;
}

/*
 * Class:     org_rocksdb_WriteOptions
 * Method:    disableWAL
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_WriteOptions_disableWAL(
    JNIEnv* env, jobject jwrite_options, jlong jhandle) {
  return reinterpret_cast<rocksdb::WriteOptions*>(jhandle)->disableWAL;
}

/////////////////////////////////////////////////////////////////////
// rocksdb::ReadOptions

/*
 * Class:     org_rocksdb_ReadOptions
 * Method:    newReadOptions
 * Signature: ()V
 */
void Java_org_rocksdb_ReadOptions_newReadOptions(
    JNIEnv* env, jobject jobj) {
  auto read_opt = new rocksdb::ReadOptions();
  rocksdb::ReadOptionsJni::setHandle(env, jobj, read_opt);
}

/*
 * Class:     org_rocksdb_ReadOptions
 * Method:    disposeInternal
 * Signature: (J)V
 */
void Java_org_rocksdb_ReadOptions_disposeInternal(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  delete reinterpret_cast<rocksdb::ReadOptions*>(jhandle);
  rocksdb::ReadOptionsJni::setHandle(env, jobj, nullptr);
}

/*
 * Class:     org_rocksdb_ReadOptions
 * Method:    setVerifyChecksums
 * Signature: (JZ)V
 */
void Java_org_rocksdb_ReadOptions_setVerifyChecksums(
    JNIEnv* env, jobject jobj, jlong jhandle,
    jboolean jverify_checksums) {
  reinterpret_cast<rocksdb::ReadOptions*>(jhandle)->verify_checksums =
      static_cast<bool>(jverify_checksums);
}

/*
 * Class:     org_rocksdb_ReadOptions
 * Method:    verifyChecksums
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_ReadOptions_verifyChecksums(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ReadOptions*>(
      jhandle)->verify_checksums;
}

/*
 * Class:     org_rocksdb_ReadOptions
 * Method:    setFillCache
 * Signature: (JZ)V
 */
void Java_org_rocksdb_ReadOptions_setFillCache(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean jfill_cache) {
  reinterpret_cast<rocksdb::ReadOptions*>(jhandle)->fill_cache =
      static_cast<bool>(jfill_cache);
}

/*
 * Class:     org_rocksdb_ReadOptions
 * Method:    fillCache
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_ReadOptions_fillCache(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ReadOptions*>(jhandle)->fill_cache;
}

/*
 * Class:     org_rocksdb_ReadOptions
 * Method:    setTailing
 * Signature: (JZ)V
 */
void Java_org_rocksdb_ReadOptions_setTailing(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean jtailing) {
  reinterpret_cast<rocksdb::ReadOptions*>(jhandle)->tailing =
      static_cast<bool>(jtailing);
}

/*
 * Class:     org_rocksdb_ReadOptions
 * Method:    tailing
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_ReadOptions_tailing(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ReadOptions*>(jhandle)->tailing;
}

/*
 * Class:     org_rocksdb_ReadOptions
 * Method:    setSnapshot
 * Signature: (JJ)V
 */
void Java_org_rocksdb_ReadOptions_setSnapshot(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jsnapshot) {
  reinterpret_cast<rocksdb::ReadOptions*>(jhandle)->snapshot =
      reinterpret_cast<rocksdb::Snapshot*>(jsnapshot);
}

/*
 * Class:     org_rocksdb_ReadOptions
 * Method:    snapshot
 * Signature: (J)J
 */
jlong Java_org_rocksdb_ReadOptions_snapshot(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto& snapshot =
      reinterpret_cast<rocksdb::ReadOptions*>(jhandle)->snapshot;
  return reinterpret_cast<jlong>(snapshot);
}

/////////////////////////////////////////////////////////////////////
// rocksdb::ComparatorOptions

/*
 * Class:     org_rocksdb_ComparatorOptions
 * Method:    newComparatorOptions
 * Signature: ()V
 */
void Java_org_rocksdb_ComparatorOptions_newComparatorOptions(
    JNIEnv* env, jobject jobj) {
  auto comparator_opt = new rocksdb::ComparatorJniCallbackOptions();
  rocksdb::ComparatorOptionsJni::setHandle(env, jobj, comparator_opt);
}

/*
 * Class:     org_rocksdb_ComparatorOptions
 * Method:    useAdaptiveMutex
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_ComparatorOptions_useAdaptiveMutex(
    JNIEnv * env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::ComparatorJniCallbackOptions*>(jhandle)
    ->use_adaptive_mutex;
}

/*
 * Class:     org_rocksdb_ComparatorOptions
 * Method:    setUseAdaptiveMutex
 * Signature: (JZ)V
 */
void Java_org_rocksdb_ComparatorOptions_setUseAdaptiveMutex(
    JNIEnv * env, jobject jobj, jlong jhandle, jboolean juse_adaptive_mutex) {
  reinterpret_cast<rocksdb::ComparatorJniCallbackOptions*>(jhandle)
    ->use_adaptive_mutex = static_cast<bool>(juse_adaptive_mutex);
}

/*
 * Class:     org_rocksdb_ComparatorOptions
 * Method:    disposeInternal
 * Signature: (J)V
 */
void Java_org_rocksdb_ComparatorOptions_disposeInternal(
    JNIEnv * env, jobject jobj, jlong jhandle) {
  delete reinterpret_cast<rocksdb::ComparatorJniCallbackOptions*>(jhandle);
  rocksdb::ComparatorOptionsJni::setHandle(env, jobj, nullptr);
}

/////////////////////////////////////////////////////////////////////
// rocksdb::FlushOptions

/*
 * Class:     org_rocksdb_FlushOptions
 * Method:    newFlushOptions
 * Signature: ()V
 */
void Java_org_rocksdb_FlushOptions_newFlushOptions(
    JNIEnv* env, jobject jobj) {
  auto flush_opt = new rocksdb::FlushOptions();
  rocksdb::FlushOptionsJni::setHandle(env, jobj, flush_opt);
}

/*
 * Class:     org_rocksdb_FlushOptions
 * Method:    setWaitForFlush
 * Signature: (JZ)V
 */
void Java_org_rocksdb_FlushOptions_setWaitForFlush(
    JNIEnv * env, jobject jobj, jlong jhandle, jboolean jwait) {
  reinterpret_cast<rocksdb::FlushOptions*>(jhandle)
    ->wait = static_cast<bool>(jwait);
}

/*
 * Class:     org_rocksdb_FlushOptions
 * Method:    waitForFlush
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_FlushOptions_waitForFlush(
    JNIEnv * env, jobject jobj, jlong jhandle) {
  return reinterpret_cast<rocksdb::FlushOptions*>(jhandle)
    ->wait;
}

/*
 * Class:     org_rocksdb_FlushOptions
 * Method:    disposeInternal
 * Signature: (J)V
 */
void Java_org_rocksdb_FlushOptions_disposeInternal(
    JNIEnv * env, jobject jobj, jlong jhandle) {
  delete reinterpret_cast<rocksdb::FlushOptions*>(jhandle);
  rocksdb::FlushOptionsJni::setHandle(env, jobj, nullptr);
}
