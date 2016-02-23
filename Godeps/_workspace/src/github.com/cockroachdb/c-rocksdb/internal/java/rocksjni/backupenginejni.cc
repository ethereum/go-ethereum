// Copyright (c) 2015, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// This file implements the "bridge" between Java and C++ and enables
// calling C++ rocksdb::BackupEngine methods from the Java side.

#include <jni.h>
#include <vector>

#include "include/org_rocksdb_BackupEngine.h"
#include "rocksdb/utilities/backupable_db.h"
#include "rocksjni/portal.h"

/*
 * Class:     org_rocksdb_BackupEngine
 * Method:    open
 * Signature: (JJ)V
 */
void Java_org_rocksdb_BackupEngine_open(
    JNIEnv* env, jobject jbe, jlong env_handle,
    jlong backupable_db_options_handle) {
  auto* rocks_env = reinterpret_cast<rocksdb::Env*>(env_handle);
  auto* backupable_db_options =
      reinterpret_cast<rocksdb::BackupableDBOptions*>(
      backupable_db_options_handle);
  rocksdb::BackupEngine* backup_engine;
  auto status = rocksdb::BackupEngine::Open(rocks_env,
      *backupable_db_options, &backup_engine);

  if (status.ok()) {
    rocksdb::BackupEngineJni::setHandle(env, jbe, backup_engine);
    return;
  }

  rocksdb::RocksDBExceptionJni::ThrowNew(env, status);
}

/*
 * Class:     org_rocksdb_BackupEngine
 * Method:    createNewBackup
 * Signature: (JJZ)V
 */
void Java_org_rocksdb_BackupEngine_createNewBackup(
    JNIEnv* env, jobject jbe, jlong jbe_handle, jlong db_handle,
    jboolean jflush_before_backup) {
  auto* db = reinterpret_cast<rocksdb::DB*>(db_handle);
  auto* backup_engine = reinterpret_cast<rocksdb::BackupEngine*>(jbe_handle);
  auto status = backup_engine->CreateNewBackup(db,
      static_cast<bool>(jflush_before_backup));

  if (status.ok()) {
    return;
  }

  rocksdb::RocksDBExceptionJni::ThrowNew(env, status);
}

/*
 * Class:     org_rocksdb_BackupEngine
 * Method:    getBackupInfo
 * Signature: (J)Ljava/util/List;
 */
jobject Java_org_rocksdb_BackupEngine_getBackupInfo(
    JNIEnv* env, jobject jbe, jlong jbe_handle) {
  auto* backup_engine = reinterpret_cast<rocksdb::BackupEngine*>(jbe_handle);
  std::vector<rocksdb::BackupInfo> backup_infos;
  backup_engine->GetBackupInfo(&backup_infos);
  return rocksdb::BackupInfoListJni::getBackupInfo(env, backup_infos);
}

/*
 * Class:     org_rocksdb_BackupEngine
 * Method:    getCorruptedBackups
 * Signature: (J)[I
 */
jintArray Java_org_rocksdb_BackupEngine_getCorruptedBackups(
    JNIEnv* env, jobject jbe, jlong jbe_handle) {
  auto* backup_engine = reinterpret_cast<rocksdb::BackupEngine*>(jbe_handle);
  std::vector<rocksdb::BackupID> backup_ids;
  backup_engine->GetCorruptedBackups(&backup_ids);
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
 * Class:     org_rocksdb_BackupEngine
 * Method:    garbageCollect
 * Signature: (J)V
 */
void Java_org_rocksdb_BackupEngine_garbageCollect(
    JNIEnv* env, jobject jbe, jlong jbe_handle) {
  auto* backup_engine = reinterpret_cast<rocksdb::BackupEngine*>(jbe_handle);
  auto status = backup_engine->GarbageCollect();

  if (status.ok()) {
    return;
  }

  rocksdb::RocksDBExceptionJni::ThrowNew(env, status);
}

/*
 * Class:     org_rocksdb_BackupEngine
 * Method:    purgeOldBackups
 * Signature: (JI)V
 */
void Java_org_rocksdb_BackupEngine_purgeOldBackups(
    JNIEnv* env, jobject jbe, jlong jbe_handle, jint jnum_backups_to_keep) {
  auto* backup_engine = reinterpret_cast<rocksdb::BackupEngine*>(jbe_handle);
  auto status =
      backup_engine->
          PurgeOldBackups(static_cast<uint32_t>(jnum_backups_to_keep));

  if (status.ok()) {
    return;
  }

  rocksdb::RocksDBExceptionJni::ThrowNew(env, status);
}

/*
 * Class:     org_rocksdb_BackupEngine
 * Method:    deleteBackup
 * Signature: (JI)V
 */
void Java_org_rocksdb_BackupEngine_deleteBackup(
    JNIEnv* env, jobject jbe, jlong jbe_handle, jint jbackup_id) {
  auto* backup_engine = reinterpret_cast<rocksdb::BackupEngine*>(jbe_handle);
  auto status =
      backup_engine->DeleteBackup(static_cast<rocksdb::BackupID>(jbackup_id));

  if (status.ok()) {
    return;
  }

  rocksdb::RocksDBExceptionJni::ThrowNew(env, status);
}

/*
 * Class:     org_rocksdb_BackupEngine
 * Method:    restoreDbFromBackup
 * Signature: (JILjava/lang/String;Ljava/lang/String;J)V
 */
void Java_org_rocksdb_BackupEngine_restoreDbFromBackup(
    JNIEnv* env, jobject jbe, jlong jbe_handle, jint jbackup_id,
    jstring jdb_dir, jstring jwal_dir, jlong jrestore_options_handle) {
  auto* backup_engine = reinterpret_cast<rocksdb::BackupEngine*>(jbe_handle);
  const char* db_dir = env->GetStringUTFChars(jdb_dir, 0);
  const char* wal_dir = env->GetStringUTFChars(jwal_dir, 0);
  auto* restore_options =
      reinterpret_cast<rocksdb::RestoreOptions*>(jrestore_options_handle);
  auto status =
      backup_engine->RestoreDBFromBackup(
          static_cast<rocksdb::BackupID>(jbackup_id), db_dir, wal_dir,
          *restore_options);
  env->ReleaseStringUTFChars(jwal_dir, wal_dir);
  env->ReleaseStringUTFChars(jdb_dir, db_dir);

  if (status.ok()) {
    return;
  }

  rocksdb::RocksDBExceptionJni::ThrowNew(env, status);
}

/*
 * Class:     org_rocksdb_BackupEngine
 * Method:    restoreDbFromLatestBackup
 * Signature: (JLjava/lang/String;Ljava/lang/String;J)V
 */
void Java_org_rocksdb_BackupEngine_restoreDbFromLatestBackup(
    JNIEnv* env, jobject jbe, jlong jbe_handle, jstring jdb_dir,
    jstring jwal_dir, jlong jrestore_options_handle) {
  auto* backup_engine = reinterpret_cast<rocksdb::BackupEngine*>(jbe_handle);
  const char* db_dir = env->GetStringUTFChars(jdb_dir, 0);
  const char* wal_dir = env->GetStringUTFChars(jwal_dir, 0);
  auto* restore_options =
      reinterpret_cast<rocksdb::RestoreOptions*>(jrestore_options_handle);
  auto status =
      backup_engine->RestoreDBFromLatestBackup(db_dir, wal_dir,
          *restore_options);
  env->ReleaseStringUTFChars(jwal_dir, wal_dir);
  env->ReleaseStringUTFChars(jdb_dir, db_dir);

  if (status.ok()) {
    return;
  }

  rocksdb::RocksDBExceptionJni::ThrowNew(env, status);
}

/*
 * Class:     org_rocksdb_BackupEngine
 * Method:    disposeInternal
 * Signature: (J)V
 */
void Java_org_rocksdb_BackupEngine_disposeInternal(
    JNIEnv* env, jobject jbe, jlong jbe_handle) {
  delete reinterpret_cast<rocksdb::BackupEngine*>(jbe_handle);
}
