// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// This file implements the "bridge" between Java and C++ and enables
// calling c++ rocksdb::RestoreBackupableDB and rocksdb::RestoreOptions methods
// from Java side.

#include <stdio.h>
#include <stdlib.h>
#include <jni.h>
#include <string>

#include "include/org_rocksdb_RestoreOptions.h"
#include "include/org_rocksdb_RestoreBackupableDB.h"
#include "rocksjni/portal.h"
#include "rocksdb/utilities/backupable_db.h"
/*
 * Class:     org_rocksdb_RestoreOptions
 * Method:    newRestoreOptions
 * Signature: (Z)J
 */
jlong Java_org_rocksdb_RestoreOptions_newRestoreOptions(JNIEnv* env,
    jobject jobj, jboolean keep_log_files) {
  auto ropt = new rocksdb::RestoreOptions(keep_log_files);
  return reinterpret_cast<jlong>(ropt);
}

/*
 * Class:     org_rocksdb_RestoreOptions
 * Method:    dispose
 * Signature: (J)V
 */
void Java_org_rocksdb_RestoreOptions_dispose(JNIEnv* env, jobject jobj,
    jlong jhandle) {
  auto ropt = reinterpret_cast<rocksdb::RestoreOptions*>(jhandle);
  assert(ropt);
  delete ropt;
}

/*
 * Class:     org_rocksdb_RestoreBackupableDB
 * Method:    newRestoreBackupableDB
 * Signature: (J)J
 */
jlong Java_org_rocksdb_RestoreBackupableDB_newRestoreBackupableDB(JNIEnv* env,
    jobject jobj, jlong jopt_handle) {
  auto opt = reinterpret_cast<rocksdb::BackupableDBOptions*>(jopt_handle);
  auto rdb = new rocksdb::RestoreBackupableDB(rocksdb::Env::Default(), *opt);
  return reinterpret_cast<jlong>(rdb);
}

/*
 * Class:     org_rocksdb_RestoreBackupableDB
 * Method:    restoreDBFromBackup0
 * Signature: (JJLjava/lang/String;Ljava/lang/String;J)V
 */
void Java_org_rocksdb_RestoreBackupableDB_restoreDBFromBackup0(JNIEnv* env,
    jobject jobj, jlong jhandle, jlong jbackup_id, jstring jdb_dir,
    jstring jwal_dir, jlong jopt_handle) {
  auto opt = reinterpret_cast<rocksdb::RestoreOptions*>(jopt_handle);

  const char* cdb_dir = env->GetStringUTFChars(jdb_dir, 0);
  const char* cwal_dir = env->GetStringUTFChars(jwal_dir, 0);

  auto rdb = reinterpret_cast<rocksdb::RestoreBackupableDB*>(jhandle);
  rocksdb::Status s = rdb->RestoreDBFromBackup(
      static_cast<rocksdb::BackupID>(jbackup_id), cdb_dir, cwal_dir, *opt);

  env->ReleaseStringUTFChars(jdb_dir, cdb_dir);
  env->ReleaseStringUTFChars(jwal_dir, cwal_dir);

  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_RestoreBackupableDB
 * Method:    restoreDBFromLatestBackup0
 * Signature: (JLjava/lang/String;Ljava/lang/String;J)V
 */
void Java_org_rocksdb_RestoreBackupableDB_restoreDBFromLatestBackup0(
    JNIEnv* env, jobject jobj, jlong jhandle, jstring jdb_dir, jstring jwal_dir,
    jlong jopt_handle) {
  auto opt = reinterpret_cast<rocksdb::RestoreOptions*>(jopt_handle);

  const char* cdb_dir = env->GetStringUTFChars(jdb_dir, 0);
  const char* cwal_dir = env->GetStringUTFChars(jwal_dir, 0);

  auto rdb = reinterpret_cast<rocksdb::RestoreBackupableDB*>(jhandle);
  rocksdb::Status s =
      rdb->RestoreDBFromLatestBackup(cdb_dir, cwal_dir, *opt);

  env->ReleaseStringUTFChars(jdb_dir, cdb_dir);
  env->ReleaseStringUTFChars(jwal_dir, cwal_dir);

  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_RestoreBackupableDB
 * Method:    purgeOldBackups0
 * Signature: (JI)V
 */
void Java_org_rocksdb_RestoreBackupableDB_purgeOldBackups0(JNIEnv* env,
    jobject jobj, jlong jhandle, jint jnum_backups_to_keep) {
  auto rdb = reinterpret_cast<rocksdb::RestoreBackupableDB*>(jhandle);
  rocksdb::Status s = rdb->PurgeOldBackups(jnum_backups_to_keep);

  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_RestoreBackupableDB
 * Method:    deleteBackup0
 * Signature: (JI)V
 */
void Java_org_rocksdb_RestoreBackupableDB_deleteBackup0(JNIEnv* env,
    jobject jobj, jlong jhandle, jint jbackup_id) {
  auto rdb = reinterpret_cast<rocksdb::RestoreBackupableDB*>(jhandle);
  rocksdb::Status s = rdb->DeleteBackup(jbackup_id);

  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_RestoreBackupableDB
 * Method:    getBackupInfo
 * Signature: (J)Ljava/util/List;
 */
jobject Java_org_rocksdb_RestoreBackupableDB_getBackupInfo(
    JNIEnv* env, jobject jbdb, jlong jhandle) {
  std::vector<rocksdb::BackupInfo> backup_infos;
  reinterpret_cast<rocksdb::RestoreBackupableDB*>(jhandle)->
      GetBackupInfo(&backup_infos);
  return rocksdb::BackupInfoListJni::getBackupInfo(env,
      backup_infos);
}

/*
 * Class:     org_rocksdb_RestoreBackupableDB
 * Method:    getCorruptedBackups
 * Signature: (J)[I;
 */
jintArray Java_org_rocksdb_RestoreBackupableDB_getCorruptedBackups(
    JNIEnv* env, jobject jbdb, jlong jhandle) {
  std::vector<rocksdb::BackupID> backup_ids;
  reinterpret_cast<rocksdb::RestoreBackupableDB*>(jhandle)->
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
 * Class:     org_rocksdb_RestoreBackupableDB
 * Method:    garbageCollect
 * Signature: (J)V
 */
void Java_org_rocksdb_RestoreBackupableDB_garbageCollect(
    JNIEnv* env, jobject jobj, jlong jhandle) {
  auto db = reinterpret_cast<rocksdb::RestoreBackupableDB*>(
      jhandle);
  rocksdb::Status s = db->GarbageCollect();

  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_RestoreBackupableDB
 * Method:    dispose
 * Signature: (J)V
 */
void Java_org_rocksdb_RestoreBackupableDB_dispose(JNIEnv* env, jobject jobj,
    jlong jhandle) {
  auto ropt = reinterpret_cast<rocksdb::RestoreBackupableDB*>(jhandle);
  assert(ropt);
  delete ropt;
}
