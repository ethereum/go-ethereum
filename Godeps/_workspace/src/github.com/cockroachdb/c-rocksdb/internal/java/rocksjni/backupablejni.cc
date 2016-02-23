// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// This file implements the "bridge" between Java and C++ and enables
// calling c++ rocksdb::BackupableDB and rocksdb::BackupableDBOptions methods
// from Java side.

#include <stdio.h>
#include <stdlib.h>
#include <jni.h>
#include <string>
#include <vector>

#include "include/org_rocksdb_BackupableDB.h"
#include "include/org_rocksdb_BackupableDBOptions.h"
#include "rocksjni/portal.h"
#include "rocksdb/utilities/backupable_db.h"

/*
 * Class:     org_rocksdb_BackupableDB
 * Method:    open
 * Signature: (JJ)V
 */
void Java_org_rocksdb_BackupableDB_open(
    JNIEnv* env, jobject jbdb, jlong jdb_handle, jlong jopt_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto opt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jopt_handle);
  auto bdb = new rocksdb::BackupableDB(db, *opt);

  // as BackupableDB extends RocksDB on the java side, we can reuse
  // the RocksDB portal here.
  rocksdb::RocksDBJni::setHandle(env, jbdb, bdb);
}

/*
 * Class:     org_rocksdb_BackupableDB
 * Method:    createNewBackup
 * Signature: (JZ)V
 */
void Java_org_rocksdb_BackupableDB_createNewBackup(
    JNIEnv* env, jobject jbdb, jlong jhandle, jboolean jflag) {
  rocksdb::Status s =
      reinterpret_cast<rocksdb::BackupableDB*>(jhandle)->CreateNewBackup(jflag);
  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_BackupableDB
 * Method:    purgeOldBackups
 * Signature: (JI)V
 */
void Java_org_rocksdb_BackupableDB_purgeOldBackups(
    JNIEnv* env, jobject jbdb, jlong jhandle, jint jnumBackupsToKeep) {
  rocksdb::Status s =
      reinterpret_cast<rocksdb::BackupableDB*>(jhandle)->
      PurgeOldBackups(jnumBackupsToKeep);
  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_BackupableDB
 * Method:    deleteBackup0
 * Signature: (JI)V
 */
void Java_org_rocksdb_BackupableDB_deleteBackup0(JNIEnv* env,
    jobject jobj, jlong jhandle, jint jbackup_id) {
  auto rdb = reinterpret_cast<rocksdb::BackupableDB*>(jhandle);
  rocksdb::Status s = rdb->DeleteBackup(jbackup_id);

  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_BackupableDB
 * Method:    getBackupInfo
 * Signature: (J)Ljava/util/List;
 */
jobject Java_org_rocksdb_BackupableDB_getBackupInfo(
    JNIEnv* env, jobject jbdb, jlong jhandle) {
  std::vector<rocksdb::BackupInfo> backup_infos;
  reinterpret_cast<rocksdb::BackupableDB*>(jhandle)->
      GetBackupInfo(&backup_infos);
  return rocksdb::BackupInfoListJni::getBackupInfo(env,
      backup_infos);
}

/*
 * Class:     org_rocksdb_BackupableDB
 * Method:    getCorruptedBackups
 * Signature: (J)[I;
 */
jintArray Java_org_rocksdb_BackupableDB_getCorruptedBackups(
    JNIEnv* env, jobject jbdb, jlong jhandle) {
  std::vector<rocksdb::BackupID> backup_ids;
  reinterpret_cast<rocksdb::BackupableDB*>(jhandle)->
      GetCorruptedBackups(&backup_ids);
  // store backupids in int array
  const std::vector<rocksdb::BackupID>::size_type
      kIdSize = backup_ids.size();
  int int_backup_ids[kIdSize];
  for (std::vector<rocksdb::BackupID>::size_type i = 0;
      i != kIdSize; i++) {
    int_backup_ids[i] = backup_ids[i];
  }
  // Store ints in java array
  jintArray ret_backup_ids;
  // Its ok to loose precision here (64->32)
  jsize ret_backup_ids_size = static_cast<jsize>(kIdSize);
  ret_backup_ids = env->NewIntArray(ret_backup_ids_size);
  env->SetIntArrayRegion(ret_backup_ids, 0, ret_backup_ids_size,
      int_backup_ids);
  return ret_backup_ids;
}

/*
 * Class:     org_rocksdb_BackupableDB
 * Method:    garbageCollect
 * Signature: (J)V
 */
void Java_org_rocksdb_BackupableDB_garbageCollect(JNIEnv* env,
    jobject jobj, jlong jhandle) {
  auto db = reinterpret_cast<rocksdb::BackupableDB*>(jhandle);
  rocksdb::Status s = db->GarbageCollect();

  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

///////////////////////////////////////////////////////////////////////////
// BackupDBOptions

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    newBackupableDBOptions
 * Signature: (Ljava/lang/String;)V
 */
void Java_org_rocksdb_BackupableDBOptions_newBackupableDBOptions(
    JNIEnv* env, jobject jobj, jstring jpath) {
  const char* cpath = env->GetStringUTFChars(jpath, 0);
  auto bopt = new rocksdb::BackupableDBOptions(cpath);
  env->ReleaseStringUTFChars(jpath, cpath);
  rocksdb::BackupableDBOptionsJni::setHandle(env, jobj, bopt);
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    backupDir
 * Signature: (J)Ljava/lang/String;
 */
jstring Java_org_rocksdb_BackupableDBOptions_backupDir(
    JNIEnv* env, jobject jopt, jlong jhandle) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  return env->NewStringUTF(bopt->backup_dir.c_str());
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    setShareTableFiles
 * Signature: (JZ)V
 */
void Java_org_rocksdb_BackupableDBOptions_setShareTableFiles(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean flag) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  bopt->share_table_files = flag;
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    shareTableFiles
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_BackupableDBOptions_shareTableFiles(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  return bopt->share_table_files;
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    setSync
 * Signature: (JZ)V
 */
void Java_org_rocksdb_BackupableDBOptions_setSync(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean flag) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  bopt->sync = flag;
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    sync
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_BackupableDBOptions_sync(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  return bopt->sync;
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    setDestroyOldData
 * Signature: (JZ)V
 */
void Java_org_rocksdb_BackupableDBOptions_setDestroyOldData(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean flag) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  bopt->destroy_old_data = flag;
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    destroyOldData
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_BackupableDBOptions_destroyOldData(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  return bopt->destroy_old_data;
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    setBackupLogFiles
 * Signature: (JZ)V
 */
void Java_org_rocksdb_BackupableDBOptions_setBackupLogFiles(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean flag) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  bopt->backup_log_files = flag;
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    backupLogFiles
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_BackupableDBOptions_backupLogFiles(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  return bopt->backup_log_files;
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    setBackupRateLimit
 * Signature: (JJ)V
 */
void Java_org_rocksdb_BackupableDBOptions_setBackupRateLimit(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jbackup_rate_limit) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  bopt->backup_rate_limit = jbackup_rate_limit;
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    backupRateLimit
 * Signature: (J)J
 */
jlong Java_org_rocksdb_BackupableDBOptions_backupRateLimit(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  return bopt->backup_rate_limit;
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    setRestoreRateLimit
 * Signature: (JJ)V
 */
void Java_org_rocksdb_BackupableDBOptions_setRestoreRateLimit(
    JNIEnv* env, jobject jobj, jlong jhandle, jlong jrestore_rate_limit) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  bopt->restore_rate_limit = jrestore_rate_limit;
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    restoreRateLimit
 * Signature: (J)J
 */
jlong Java_org_rocksdb_BackupableDBOptions_restoreRateLimit(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  return bopt->restore_rate_limit;
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    setShareFilesWithChecksum
 * Signature: (JZ)V
 */
void Java_org_rocksdb_BackupableDBOptions_setShareFilesWithChecksum(
    JNIEnv* env, jobject jobj, jlong jhandle, jboolean flag) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  bopt->share_files_with_checksum = flag;
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    shareFilesWithChecksum
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_BackupableDBOptions_shareFilesWithChecksum(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  return bopt->share_files_with_checksum;
}

/*
 * Class:     org_rocksdb_BackupableDBOptions
 * Method:    disposeInternal
 * Signature: (J)V
 */
void Java_org_rocksdb_BackupableDBOptions_disposeInternal(
    JNIEnv* env, jobject jopt, jlong jhandle) {
  auto bopt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jhandle);
  assert(bopt);
  delete bopt;
  rocksdb::BackupableDBOptionsJni::setHandle(env, jopt, nullptr);
}
