// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// This file implements the "bridge" between Java and C++ and enables
// calling c++ rocksdb::DB methods from Java side.

#include <jni.h>
#include <stdio.h>
#include <stdlib.h>
#include <memory>
#include <string>
#include <vector>

#include "include/org_rocksdb_RocksDB.h"
#include "rocksdb/db.h"
#include "rocksdb/cache.h"
#include "rocksdb/types.h"
#include "rocksjni/portal.h"

//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB::Open

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    open
 * Signature: (JLjava/lang/String;)V
 */
void Java_org_rocksdb_RocksDB_open__JLjava_lang_String_2(
    JNIEnv* env, jobject jdb, jlong jopt_handle, jstring jdb_path) {
  auto opt = reinterpret_cast<rocksdb::Options*>(jopt_handle);
  rocksdb::DB* db = nullptr;
  const char* db_path = env->GetStringUTFChars(jdb_path, 0);
  rocksdb::Status s = rocksdb::DB::Open(*opt, db_path, &db);
  env->ReleaseStringUTFChars(jdb_path, db_path);

  if (s.ok()) {
    rocksdb::RocksDBJni::setHandle(env, jdb, db);
    return;
  }
  rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    openROnly
 * Signature: (JLjava/lang/String;)V
 */
void Java_org_rocksdb_RocksDB_openROnly__JLjava_lang_String_2(
    JNIEnv* env, jobject jdb, jlong jopt_handle, jstring jdb_path) {
  auto opt = reinterpret_cast<rocksdb::Options*>(jopt_handle);
  rocksdb::DB* db = nullptr;
  const char* db_path = env->GetStringUTFChars(jdb_path, 0);
  rocksdb::Status s = rocksdb::DB::OpenForReadOnly(*opt,
      db_path, &db);
  env->ReleaseStringUTFChars(jdb_path, db_path);

  if (s.ok()) {
    rocksdb::RocksDBJni::setHandle(env, jdb, db);
    return;
  }
  rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    openROnly
 * Signature: (JLjava/lang/String;Ljava/util/List;I)Ljava/util/List;
 */
jobject
    Java_org_rocksdb_RocksDB_openROnly__JLjava_lang_String_2Ljava_util_List_2I(
    JNIEnv* env, jobject jdb, jlong jopt_handle, jstring jdb_path,
    jobject jcfdesc_list, jint jcfdesc_count) {
  auto opt = reinterpret_cast<rocksdb::Options*>(jopt_handle);
  rocksdb::DB* db = nullptr;
  const char* db_path = env->GetStringUTFChars(jdb_path, 0);

  std::vector<jbyte*> cfnames_to_free;
  std::vector<jbyteArray> jcfnames_for_free;

  std::vector<rocksdb::ColumnFamilyDescriptor> column_families;
  std::vector<rocksdb::ColumnFamilyHandle* > handles;
  // get iterator for ColumnFamilyDescriptors
  jobject iteratorObj = env->CallObjectMethod(
      jcfdesc_list, rocksdb::ListJni::getIteratorMethod(env));

  // iterate over ColumnFamilyDescriptors
  while (env->CallBooleanMethod(
      iteratorObj, rocksdb::ListJni::getHasNextMethod(env)) == JNI_TRUE) {
      // get ColumnFamilyDescriptor
      jobject jcf_descriptor = env->CallObjectMethod(iteratorObj,
          rocksdb::ListJni::getNextMethod(env));
      // get ColumnFamilyName
      jbyteArray cf_name_in_byte_array = static_cast<jbyteArray>(
          env->CallObjectMethod(jcf_descriptor,
          rocksdb::ColumnFamilyDescriptorJni::getColumnFamilyNameMethod(
              env)));
      // get CF Options
      jobject jcf_opt_obj = env->CallObjectMethod(jcf_descriptor,
          rocksdb::ColumnFamilyDescriptorJni::getColumnFamilyOptionsMethod(
              env));
      rocksdb::ColumnFamilyOptions* cfOptions =
          rocksdb::ColumnFamilyOptionsJni::getHandle(env, jcf_opt_obj);

      jbyte* cfname = env->GetByteArrayElements(cf_name_in_byte_array, 0);
      const int len = env->GetArrayLength(cf_name_in_byte_array);

      // free allocated cfnames after call to open
      cfnames_to_free.push_back(cfname);
      jcfnames_for_free.push_back(cf_name_in_byte_array);
      column_families.push_back(rocksdb::ColumnFamilyDescriptor(
          std::string(reinterpret_cast<char *>(cfname), len), *cfOptions));
  }

  rocksdb::Status s = rocksdb::DB::OpenForReadOnly(*opt,
      db_path, column_families, &handles, &db);
  env->ReleaseStringUTFChars(jdb_path, db_path);
  // free jbyte allocations
  for (std::vector<jbyte*>::size_type i = 0;
      i != cfnames_to_free.size(); i++) {
    // free  cfnames
    env->ReleaseByteArrayElements(jcfnames_for_free[i], cfnames_to_free[i], 0);
  }

  // check if open operation was successful
  if (s.ok()) {
    rocksdb::RocksDBJni::setHandle(env, jdb, db);
    jclass jListClazz = env->FindClass("java/util/ArrayList");
    jmethodID midList = rocksdb::ListJni::getArrayListConstructorMethodId(
        env, jListClazz);
    jobject jcfhandle_list = env->NewObject(jListClazz,
        midList, handles.size());
    // insert in java list
    for (std::vector<rocksdb::ColumnFamilyHandle*>::size_type i = 0;
        i != handles.size(); i++) {
      // jlong must be converted to Long due to collections restrictions
      jclass jLongClazz = env->FindClass("java/lang/Long");
      jmethodID midLong = env->GetMethodID(jLongClazz, "<init>", "(J)V");
      jobject obj = env->NewObject(jLongClazz, midLong,
          reinterpret_cast<jlong>(handles[i]));
      env->CallBooleanMethod(jcfhandle_list,
          rocksdb::ListJni::getListAddMethodId(env), obj);
    }

    return jcfhandle_list;
  }
  rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  return nullptr;
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    open
 * Signature: (JLjava/lang/String;Ljava/util/List;I)Ljava/util/List;
 */
jobject Java_org_rocksdb_RocksDB_open__JLjava_lang_String_2Ljava_util_List_2I(
    JNIEnv* env, jobject jdb, jlong jopt_handle, jstring jdb_path,
    jobject jcfdesc_list, jint jcfdesc_count) {
  auto opt = reinterpret_cast<rocksdb::Options*>(jopt_handle);
  rocksdb::DB* db = nullptr;
  const char* db_path = env->GetStringUTFChars(jdb_path, 0);

  std::vector<jbyte*> cfnames_to_free;
  std::vector<jbyteArray> jcfnames_for_free;

  std::vector<rocksdb::ColumnFamilyDescriptor> column_families;
  std::vector<rocksdb::ColumnFamilyHandle* > handles;
  // get iterator for ColumnFamilyDescriptors
  jobject iteratorObj = env->CallObjectMethod(
      jcfdesc_list, rocksdb::ListJni::getIteratorMethod(env));

  // iterate over ColumnFamilyDescriptors
  while (env->CallBooleanMethod(
      iteratorObj, rocksdb::ListJni::getHasNextMethod(env)) == JNI_TRUE) {
      // get ColumnFamilyDescriptor
      jobject jcf_descriptor = env->CallObjectMethod(iteratorObj,
          rocksdb::ListJni::getNextMethod(env));
      // get ColumnFamilyName
      jbyteArray cf_name_in_byte_array = static_cast<jbyteArray>(
          env->CallObjectMethod(jcf_descriptor,
          rocksdb::ColumnFamilyDescriptorJni::getColumnFamilyNameMethod(
              env)));
      // get CF Options
      jobject jcf_opt_obj = env->CallObjectMethod(jcf_descriptor,
          rocksdb::ColumnFamilyDescriptorJni::getColumnFamilyOptionsMethod(
              env));
      rocksdb::ColumnFamilyOptions* cfOptions =
          rocksdb::ColumnFamilyOptionsJni::getHandle(env, jcf_opt_obj);

      jbyte* cfname = env->GetByteArrayElements(cf_name_in_byte_array, 0);
      const int len = env->GetArrayLength(cf_name_in_byte_array);

      // free allocated cfnames after call to open
      cfnames_to_free.push_back(cfname);
      jcfnames_for_free.push_back(cf_name_in_byte_array);
      column_families.push_back(rocksdb::ColumnFamilyDescriptor(
          std::string(reinterpret_cast<char *>(cfname), len), *cfOptions));
  }

  rocksdb::Status s = rocksdb::DB::Open(*opt, db_path, column_families,
      &handles, &db);
  env->ReleaseStringUTFChars(jdb_path, db_path);
  // free jbyte allocations
  for (std::vector<jbyte*>::size_type i = 0;
      i != cfnames_to_free.size(); i++) {
    // free  cfnames
    env->ReleaseByteArrayElements(jcfnames_for_free[i], cfnames_to_free[i], 0);
  }

  // check if open operation was successful
  if (s.ok()) {
    rocksdb::RocksDBJni::setHandle(env, jdb, db);
    jclass jListClazz = env->FindClass("java/util/ArrayList");
    jmethodID midList = rocksdb::ListJni::getArrayListConstructorMethodId(
        env, jListClazz);
    jobject jcfhandle_list = env->NewObject(jListClazz,
        midList, handles.size());
    // insert in java list
    for (std::vector<rocksdb::ColumnFamilyHandle*>::size_type i = 0;
        i != handles.size(); i++) {
      // jlong must be converted to Long due to collections restrictions
      jclass jLongClazz = env->FindClass("java/lang/Long");
      jmethodID midLong = env->GetMethodID(jLongClazz, "<init>", "(J)V");
      jobject obj = env->NewObject(jLongClazz, midLong,
          reinterpret_cast<jlong>(handles[i]));
      env->CallBooleanMethod(jcfhandle_list,
          rocksdb::ListJni::getListAddMethodId(env), obj);
    }

    return jcfhandle_list;
  }
  rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  return nullptr;
}

//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB::ListColumnFamilies

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    listColumnFamilies
 * Signature: (JLjava/lang/String;)Ljava/util/List;
 */
jobject Java_org_rocksdb_RocksDB_listColumnFamilies(
    JNIEnv* env, jclass jclazz, jlong jopt_handle, jstring jdb_path) {
  std::vector<std::string> column_family_names;
  auto opt = reinterpret_cast<rocksdb::Options*>(jopt_handle);
  const char* db_path = env->GetStringUTFChars(jdb_path, 0);
  jobject jvalue_list = nullptr;

  rocksdb::Status s = rocksdb::DB::ListColumnFamilies(*opt, db_path,
      &column_family_names);
  env->ReleaseStringUTFChars(jdb_path, db_path);
  if (s.ok()) {
    // Don't reuse class pointer
    jclass jListClazz = env->FindClass("java/util/ArrayList");
    jmethodID mid = rocksdb::ListJni::getArrayListConstructorMethodId(env,
        jListClazz);
    jvalue_list = env->NewObject(jListClazz, mid, column_family_names.size());

    for (std::vector<std::string>::size_type i = 0;
        i < column_family_names.size(); i++) {
      jbyteArray jcf_value =
          env->NewByteArray(static_cast<jsize>(column_family_names[i].size()));
      env->SetByteArrayRegion(
          jcf_value, 0, static_cast<jsize>(column_family_names[i].size()),
          reinterpret_cast<const jbyte*>(column_family_names[i].data()));
      env->CallBooleanMethod(jvalue_list,
          rocksdb::ListJni::getListAddMethodId(env), jcf_value);
    }
  }
  return jvalue_list;
}

//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB::Put

void rocksdb_put_helper(
    JNIEnv* env, rocksdb::DB* db, const rocksdb::WriteOptions& write_options,
    rocksdb::ColumnFamilyHandle* cf_handle, jbyteArray jkey, jint jkey_len,
    jbyteArray jentry_value, jint jentry_value_len) {

  jbyte* key = env->GetByteArrayElements(jkey, 0);
  jbyte* value = env->GetByteArrayElements(jentry_value, 0);
  rocksdb::Slice key_slice(reinterpret_cast<char*>(key), jkey_len);
  rocksdb::Slice value_slice(reinterpret_cast<char*>(value),
      jentry_value_len);

  rocksdb::Status s;
  if (cf_handle != nullptr) {
    s = db->Put(write_options, cf_handle, key_slice, value_slice);
  } else {
    // backwards compatibility
    s = db->Put(write_options, key_slice, value_slice);
  }

  // trigger java unref on key and value.
  // by passing JNI_ABORT, it will simply release the reference without
  // copying the result back to the java byte array.
  env->ReleaseByteArrayElements(jkey, key, JNI_ABORT);
  env->ReleaseByteArrayElements(jentry_value, value, JNI_ABORT);

  if (s.ok()) {
    return;
  }
  rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    put
 * Signature: (J[BI[BI)V
 */
void Java_org_rocksdb_RocksDB_put__J_3BI_3BI(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jbyteArray jkey, jint jkey_len,
    jbyteArray jentry_value, jint jentry_value_len) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  static const rocksdb::WriteOptions default_write_options =
      rocksdb::WriteOptions();

  rocksdb_put_helper(env, db, default_write_options, nullptr,
                     jkey, jkey_len,
                     jentry_value, jentry_value_len);
}
/*
 * Class:     org_rocksdb_RocksDB
 * Method:    put
 * Signature: (J[BI[BIJ)V
 */
void Java_org_rocksdb_RocksDB_put__J_3BI_3BIJ(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jbyteArray jkey, jint jkey_len,
    jbyteArray jentry_value, jint jentry_value_len, jlong jcf_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  static const rocksdb::WriteOptions default_write_options =
      rocksdb::WriteOptions();
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  if (cf_handle != nullptr) {
    rocksdb_put_helper(env, db, default_write_options, cf_handle,
        jkey, jkey_len, jentry_value, jentry_value_len);
  } else {
    rocksdb::RocksDBExceptionJni::ThrowNew(env,
        rocksdb::Status::InvalidArgument("Invalid ColumnFamilyHandle."));
  }
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    put
 * Signature: (JJ[BI[BI)V
 */
void Java_org_rocksdb_RocksDB_put__JJ_3BI_3BI(
    JNIEnv* env, jobject jdb,
    jlong jdb_handle, jlong jwrite_options_handle,
    jbyteArray jkey, jint jkey_len,
    jbyteArray jentry_value, jint jentry_value_len) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto write_options = reinterpret_cast<rocksdb::WriteOptions*>(
      jwrite_options_handle);

  rocksdb_put_helper(env, db, *write_options, nullptr,
                     jkey, jkey_len,
                     jentry_value, jentry_value_len);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    put
 * Signature: (JJ[BI[BIJ)V
 */
void Java_org_rocksdb_RocksDB_put__JJ_3BI_3BIJ(
    JNIEnv* env, jobject jdb,
    jlong jdb_handle, jlong jwrite_options_handle,
    jbyteArray jkey, jint jkey_len,
    jbyteArray jentry_value, jint jentry_value_len, jlong jcf_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto write_options = reinterpret_cast<rocksdb::WriteOptions*>(
      jwrite_options_handle);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  if (cf_handle != nullptr) {
    rocksdb_put_helper(env, db, *write_options, cf_handle,
        jkey, jkey_len, jentry_value, jentry_value_len);
  } else {
    rocksdb::RocksDBExceptionJni::ThrowNew(env,
        rocksdb::Status::InvalidArgument("Invalid ColumnFamilyHandle."));
  }
}

//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB::Write
/*
 * Class:     org_rocksdb_RocksDB
 * Method:    write0
 * Signature: (JJ)V
 */
void Java_org_rocksdb_RocksDB_write0(
    JNIEnv* env, jobject jdb,
    jlong jwrite_options_handle, jlong jwb_handle) {
  rocksdb::DB* db = rocksdb::RocksDBJni::getHandle(env, jdb);
  auto* write_options = reinterpret_cast<rocksdb::WriteOptions*>(
      jwrite_options_handle);
  auto* wb = reinterpret_cast<rocksdb::WriteBatch*>(jwb_handle);

  rocksdb::Status s = db->Write(*write_options, wb);

  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    write1
 * Signature: (JJ)V
 */
void Java_org_rocksdb_RocksDB_write1(
    JNIEnv* env, jobject jdb,
    jlong jwrite_options_handle, jlong jwbwi_handle) {
  rocksdb::DB* db = rocksdb::RocksDBJni::getHandle(env, jdb);
  auto* write_options = reinterpret_cast<rocksdb::WriteOptions*>(
      jwrite_options_handle);
  auto* wbwi = reinterpret_cast<rocksdb::WriteBatchWithIndex*>(jwbwi_handle);
  auto* wb = wbwi->GetWriteBatch();

  rocksdb::Status s = db->Write(*write_options, wb);

  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB::KeyMayExist
jboolean key_may_exist_helper(JNIEnv* env, rocksdb::DB* db,
    const rocksdb::ReadOptions& read_opt,
    rocksdb::ColumnFamilyHandle* cf_handle, jbyteArray jkey, jint jkey_len,
    jobject jstring_buffer) {
  std::string value;
  bool value_found = false;
  jboolean isCopy;
  jbyte* key = env->GetByteArrayElements(jkey, &isCopy);
  rocksdb::Slice key_slice(reinterpret_cast<char*>(key), jkey_len);
  bool keyMayExist;
  if (cf_handle != nullptr) {
    keyMayExist = db->KeyMayExist(read_opt, cf_handle, key_slice,
        &value, &value_found);
  } else {
    keyMayExist = db->KeyMayExist(read_opt, key_slice,
        &value, &value_found);
  }

  if (value_found && !value.empty()) {
    jclass clazz = env->GetObjectClass(jstring_buffer);
    jmethodID mid = env->GetMethodID(clazz, "append",
        "(Ljava/lang/String;)Ljava/lang/StringBuffer;");
    jstring new_value_str = env->NewStringUTF(value.c_str());
    env->CallObjectMethod(jstring_buffer, mid, new_value_str);
  }
  env->ReleaseByteArrayElements(jkey, key, JNI_ABORT);
  return static_cast<jboolean>(keyMayExist);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    keyMayExist
 * Signature: ([BILjava/lang/StringBuffer;)Z
 */
jboolean Java_org_rocksdb_RocksDB_keyMayExist___3BILjava_lang_StringBuffer_2(
    JNIEnv* env, jobject jdb, jbyteArray jkey, jint jkey_len,
    jobject jstring_buffer) {
  rocksdb::DB* db = rocksdb::RocksDBJni::getHandle(env, jdb);
  return key_may_exist_helper(env, db, rocksdb::ReadOptions(),
      nullptr, jkey, jkey_len, jstring_buffer);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    keyMayExist
 * Signature: ([BIJLjava/lang/StringBuffer;)Z
 */
jboolean Java_org_rocksdb_RocksDB_keyMayExist___3BIJLjava_lang_StringBuffer_2(
    JNIEnv* env, jobject jdb, jbyteArray jkey, jint jkey_len,
    jlong jcf_handle, jobject jstring_buffer) {
  rocksdb::DB* db = rocksdb::RocksDBJni::getHandle(env, jdb);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(
      jcf_handle);
  if (cf_handle != nullptr) {
    return key_may_exist_helper(env, db, rocksdb::ReadOptions(),
        cf_handle, jkey, jkey_len, jstring_buffer);
  } else {
    rocksdb::RocksDBExceptionJni::ThrowNew(env,
        rocksdb::Status::InvalidArgument("Invalid ColumnFamilyHandle."));
  }
  return true;
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    keyMayExist
 * Signature: (J[BILjava/lang/StringBuffer;)Z
 */
jboolean Java_org_rocksdb_RocksDB_keyMayExist__J_3BILjava_lang_StringBuffer_2(
    JNIEnv* env, jobject jdb, jlong jread_options_handle,
    jbyteArray jkey, jint jkey_len, jobject jstring_buffer) {
  rocksdb::DB* db = rocksdb::RocksDBJni::getHandle(env, jdb);
  auto& read_options = *reinterpret_cast<rocksdb::ReadOptions*>(
      jread_options_handle);
  return key_may_exist_helper(env, db, read_options,
      nullptr, jkey, jkey_len, jstring_buffer);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    keyMayExist
 * Signature: (J[BIJLjava/lang/StringBuffer;)Z
 */
jboolean Java_org_rocksdb_RocksDB_keyMayExist__J_3BIJLjava_lang_StringBuffer_2(
    JNIEnv* env, jobject jdb, jlong jread_options_handle,
    jbyteArray jkey, jint jkey_len, jlong jcf_handle, jobject jstring_buffer) {
  rocksdb::DB* db = rocksdb::RocksDBJni::getHandle(env, jdb);
  auto& read_options = *reinterpret_cast<rocksdb::ReadOptions*>(
      jread_options_handle);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(
      jcf_handle);
  if (cf_handle != nullptr) {
    return key_may_exist_helper(env, db, read_options, cf_handle,
        jkey, jkey_len, jstring_buffer);
  } else {
    rocksdb::RocksDBExceptionJni::ThrowNew(env,
        rocksdb::Status::InvalidArgument("Invalid ColumnFamilyHandle."));
  }
  return true;
}

//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB::Get

jbyteArray rocksdb_get_helper(
    JNIEnv* env, rocksdb::DB* db, const rocksdb::ReadOptions& read_opt,
    rocksdb::ColumnFamilyHandle* column_family_handle, jbyteArray jkey,
    jint jkey_len) {
  jboolean isCopy;
  jbyte* key = env->GetByteArrayElements(jkey, &isCopy);
  rocksdb::Slice key_slice(
      reinterpret_cast<char*>(key), jkey_len);

  std::string value;
  rocksdb::Status s;
  if (column_family_handle != nullptr) {
    s = db->Get(read_opt, column_family_handle, key_slice, &value);
  } else {
    // backwards compatibility
    s = db->Get(read_opt, key_slice, &value);
  }

  // trigger java unref on key.
  // by passing JNI_ABORT, it will simply release the reference without
  // copying the result back to the java byte array.
  env->ReleaseByteArrayElements(jkey, key, JNI_ABORT);

  if (s.IsNotFound()) {
    return nullptr;
  }

  if (s.ok()) {
    jbyteArray jret_value = env->NewByteArray(static_cast<jsize>(value.size()));
    env->SetByteArrayRegion(jret_value, 0, static_cast<jsize>(value.size()),
                            reinterpret_cast<const jbyte*>(value.c_str()));
    return jret_value;
  }
  rocksdb::RocksDBExceptionJni::ThrowNew(env, s);

  return nullptr;
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    get
 * Signature: (J[BI)[B
 */
jbyteArray Java_org_rocksdb_RocksDB_get__J_3BI(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jbyteArray jkey, jint jkey_len) {
  return rocksdb_get_helper(env,
      reinterpret_cast<rocksdb::DB*>(jdb_handle),
      rocksdb::ReadOptions(), nullptr,
      jkey, jkey_len);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    get
 * Signature: (J[BIJ)[B
 */
jbyteArray Java_org_rocksdb_RocksDB_get__J_3BIJ(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jbyteArray jkey, jint jkey_len, jlong jcf_handle) {
  auto db_handle = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  if (cf_handle != nullptr) {
    return rocksdb_get_helper(env, db_handle, rocksdb::ReadOptions(),
        cf_handle, jkey, jkey_len);
  } else {
    rocksdb::RocksDBExceptionJni::ThrowNew(env,
        rocksdb::Status::InvalidArgument("Invalid ColumnFamilyHandle."));
    // will never be evaluated
    return env->NewByteArray(0);
  }
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    get
 * Signature: (JJ[BI)[B
 */
jbyteArray Java_org_rocksdb_RocksDB_get__JJ_3BI(
    JNIEnv* env, jobject jdb, jlong jdb_handle, jlong jropt_handle,
    jbyteArray jkey, jint jkey_len) {
  return rocksdb_get_helper(env,
      reinterpret_cast<rocksdb::DB*>(jdb_handle),
      *reinterpret_cast<rocksdb::ReadOptions*>(jropt_handle), nullptr,
      jkey, jkey_len);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    get
 * Signature: (JJ[BIJ)[B
 */
jbyteArray Java_org_rocksdb_RocksDB_get__JJ_3BIJ(
    JNIEnv* env, jobject jdb, jlong jdb_handle, jlong jropt_handle,
    jbyteArray jkey, jint jkey_len, jlong jcf_handle) {
  auto db_handle = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto& ro_opt = *reinterpret_cast<rocksdb::ReadOptions*>(jropt_handle);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  if (cf_handle != nullptr) {
    return rocksdb_get_helper(env, db_handle, ro_opt, cf_handle,
        jkey, jkey_len);
  } else {
    rocksdb::RocksDBExceptionJni::ThrowNew(env,
        rocksdb::Status::InvalidArgument("Invalid ColumnFamilyHandle."));
    // will never be evaluated
    return env->NewByteArray(0);
  }
}

jint rocksdb_get_helper(
    JNIEnv* env, rocksdb::DB* db, const rocksdb::ReadOptions& read_options,
    rocksdb::ColumnFamilyHandle* column_family_handle, jbyteArray jkey,
    jint jkey_len, jbyteArray jentry_value, jint jentry_value_len) {
  static const int kNotFound = -1;
  static const int kStatusError = -2;

  jbyte* key = env->GetByteArrayElements(jkey, 0);
  rocksdb::Slice key_slice(
      reinterpret_cast<char*>(key), jkey_len);

  // TODO(yhchiang): we might save one memory allocation here by adding
  // a DB::Get() function which takes preallocated jbyte* as input.
  std::string cvalue;
  rocksdb::Status s;
  if (column_family_handle != nullptr) {
    s = db->Get(read_options, column_family_handle, key_slice, &cvalue);
  } else {
    // backwards compatibility
    s = db->Get(read_options, key_slice, &cvalue);
  }

  // trigger java unref on key.
  // by passing JNI_ABORT, it will simply release the reference without
  // copying the result back to the java byte array.
  env->ReleaseByteArrayElements(jkey, key, JNI_ABORT);

  if (s.IsNotFound()) {
    return kNotFound;
  } else if (!s.ok()) {
    // Here since we are throwing a Java exception from c++ side.
    // As a result, c++ does not know calling this function will in fact
    // throwing an exception.  As a result, the execution flow will
    // not stop here, and codes after this throw will still be
    // executed.
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);

    // Return a dummy const value to avoid compilation error, although
    // java side might not have a chance to get the return value :)
    return kStatusError;
  }

  int cvalue_len = static_cast<int>(cvalue.size());
  int length = std::min(jentry_value_len, cvalue_len);

  env->SetByteArrayRegion(
      jentry_value, 0, length,
      reinterpret_cast<const jbyte*>(cvalue.c_str()));
  return cvalue_len;
}

// cf multi get
jobject multi_get_helper(JNIEnv* env, jobject jdb, rocksdb::DB* db,
    const rocksdb::ReadOptions& rOpt, jobject jkey_list, jint jkeys_count,
    jobject jcfhandle_list) {
  std::vector<rocksdb::Slice> keys;
  std::vector<jbyte*> keys_to_free;
  std::vector<rocksdb::ColumnFamilyHandle*> cf_handles;

  if (jcfhandle_list != nullptr) {
    // get cf iterator
    jobject cfIteratorObj = env->CallObjectMethod(
        jcfhandle_list, rocksdb::ListJni::getIteratorMethod(env));

    // iterate over keys and convert java byte array to slice
    while (env->CallBooleanMethod(
        cfIteratorObj, rocksdb::ListJni::getHasNextMethod(env)) == JNI_TRUE) {
      jobject jobj = (jbyteArray) env->CallObjectMethod(
          cfIteratorObj, rocksdb::ListJni::getNextMethod(env));
      rocksdb::ColumnFamilyHandle* cfHandle =
          rocksdb::ColumnFamilyHandleJni::getHandle(env, jobj);
      cf_handles.push_back(cfHandle);
    }
  }

  // Process key list
  // get iterator
  jobject iteratorObj = env->CallObjectMethod(
      jkey_list, rocksdb::ListJni::getIteratorMethod(env));

  // iterate over keys and convert java byte array to slice
  while (env->CallBooleanMethod(
      iteratorObj, rocksdb::ListJni::getHasNextMethod(env)) == JNI_TRUE) {
    jbyteArray jkey = (jbyteArray) env->CallObjectMethod(
       iteratorObj, rocksdb::ListJni::getNextMethod(env));
    jint key_length = env->GetArrayLength(jkey);

    jbyte* key = new jbyte[key_length];
    env->GetByteArrayRegion(jkey, 0, key_length, key);
    // store allocated jbyte to free it after multiGet call
    keys_to_free.push_back(key);

    rocksdb::Slice key_slice(
      reinterpret_cast<char*>(key), key_length);
    keys.push_back(key_slice);
  }

  std::vector<std::string> values;
  std::vector<rocksdb::Status> s;
  if (cf_handles.size() == 0) {
    s = db->MultiGet(rOpt, keys, &values);
  } else {
    s = db->MultiGet(rOpt, cf_handles, keys, &values);
  }

  // Don't reuse class pointer
  jclass jclazz = env->FindClass("java/util/ArrayList");
  jmethodID mid = rocksdb::ListJni::getArrayListConstructorMethodId(
      env, jclazz);
  jobject jvalue_list = env->NewObject(jclazz, mid, jkeys_count);

  // insert in java list
  for (std::vector<rocksdb::Status>::size_type i = 0; i != s.size(); i++) {
    if (s[i].ok()) {
      jbyteArray jentry_value =
          env->NewByteArray(static_cast<jsize>(values[i].size()));
      env->SetByteArrayRegion(
          jentry_value, 0, static_cast<jsize>(values[i].size()),
          reinterpret_cast<const jbyte*>(values[i].c_str()));
      env->CallBooleanMethod(
          jvalue_list, rocksdb::ListJni::getListAddMethodId(env),
              jentry_value);
    } else {
      env->CallBooleanMethod(
          jvalue_list, rocksdb::ListJni::getListAddMethodId(env), nullptr);
    }
  }
  // free up allocated byte arrays
  for (std::vector<jbyte*>::size_type i = 0; i != keys_to_free.size(); i++) {
    delete[] keys_to_free[i];
  }
  keys_to_free.clear();
  return jvalue_list;
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    multiGet
 * Signature: (JLjava/util/List;I)Ljava/util/List;
 */
jobject Java_org_rocksdb_RocksDB_multiGet__JLjava_util_List_2I(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jobject jkey_list, jint jkeys_count) {
  return multi_get_helper(env, jdb, reinterpret_cast<rocksdb::DB*>(jdb_handle),
      rocksdb::ReadOptions(), jkey_list, jkeys_count, nullptr);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    multiGet
 * Signature: (JLjava/util/List;ILjava/util/List;)Ljava/util/List;
 */
jobject
    Java_org_rocksdb_RocksDB_multiGet__JLjava_util_List_2ILjava_util_List_2(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jobject jkey_list, jint jkeys_count, jobject jcfhandle_list) {
  return multi_get_helper(env, jdb, reinterpret_cast<rocksdb::DB*>(jdb_handle),
      rocksdb::ReadOptions(), jkey_list, jkeys_count, jcfhandle_list);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    multiGet
 * Signature: (JJLjava/util/List;I)Ljava/util/List;
 */
jobject Java_org_rocksdb_RocksDB_multiGet__JJLjava_util_List_2I(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jlong jropt_handle, jobject jkey_list, jint jkeys_count) {
  return multi_get_helper(env, jdb, reinterpret_cast<rocksdb::DB*>(jdb_handle),
      *reinterpret_cast<rocksdb::ReadOptions*>(jropt_handle), jkey_list,
      jkeys_count, nullptr);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    multiGet
 * Signature: (JJLjava/util/List;ILjava/util/List;)Ljava/util/List;
 */
jobject
    Java_org_rocksdb_RocksDB_multiGet__JJLjava_util_List_2ILjava_util_List_2(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jlong jropt_handle, jobject jkey_list, jint jkeys_count,
    jobject jcfhandle_list) {
  return multi_get_helper(env, jdb, reinterpret_cast<rocksdb::DB*>(jdb_handle),
      *reinterpret_cast<rocksdb::ReadOptions*>(jropt_handle), jkey_list,
      jkeys_count, jcfhandle_list);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    get
 * Signature: (J[BI[BI)I
 */
jint Java_org_rocksdb_RocksDB_get__J_3BI_3BI(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jbyteArray jkey, jint jkey_len,
    jbyteArray jentry_value, jint jentry_value_len) {
  return rocksdb_get_helper(env,
      reinterpret_cast<rocksdb::DB*>(jdb_handle),
      rocksdb::ReadOptions(), nullptr,
      jkey, jkey_len, jentry_value, jentry_value_len);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    get
 * Signature: (J[BI[BIJ)I
 */
jint Java_org_rocksdb_RocksDB_get__J_3BI_3BIJ(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jbyteArray jkey, jint jkey_len,
    jbyteArray jentry_value, jint jentry_value_len, jlong jcf_handle) {
  auto db_handle = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  if (cf_handle != nullptr) {
    return rocksdb_get_helper(env, db_handle, rocksdb::ReadOptions(), cf_handle,
        jkey, jkey_len, jentry_value, jentry_value_len);
  } else {
    rocksdb::RocksDBExceptionJni::ThrowNew(env,
        rocksdb::Status::InvalidArgument("Invalid ColumnFamilyHandle."));
    // will never be evaluated
    return 0;
  }
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    get
 * Signature: (JJ[BI[BI)I
 */
jint Java_org_rocksdb_RocksDB_get__JJ_3BI_3BI(
    JNIEnv* env, jobject jdb, jlong jdb_handle, jlong jropt_handle,
    jbyteArray jkey, jint jkey_len,
    jbyteArray jentry_value, jint jentry_value_len) {
  return rocksdb_get_helper(env,
      reinterpret_cast<rocksdb::DB*>(jdb_handle),
      *reinterpret_cast<rocksdb::ReadOptions*>(jropt_handle),
      nullptr, jkey, jkey_len, jentry_value, jentry_value_len);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    get
 * Signature: (JJ[BI[BIJ)I
 */
jint Java_org_rocksdb_RocksDB_get__JJ_3BI_3BIJ(
    JNIEnv* env, jobject jdb, jlong jdb_handle, jlong jropt_handle,
    jbyteArray jkey, jint jkey_len,
    jbyteArray jentry_value, jint jentry_value_len, jlong jcf_handle) {
  auto db_handle = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto& ro_opt = *reinterpret_cast<rocksdb::ReadOptions*>(jropt_handle);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  if (cf_handle != nullptr) {
    return rocksdb_get_helper(env, db_handle, ro_opt, cf_handle, jkey,
        jkey_len, jentry_value, jentry_value_len);
  } else {
    rocksdb::RocksDBExceptionJni::ThrowNew(env,
        rocksdb::Status::InvalidArgument("Invalid ColumnFamilyHandle."));
    // will never be evaluated
    return 0;
  }
}
//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB::Delete()
void rocksdb_remove_helper(
    JNIEnv* env, rocksdb::DB* db, const rocksdb::WriteOptions& write_options,
    rocksdb::ColumnFamilyHandle* cf_handle, jbyteArray jkey, jint jkey_len) {
  jbyte* key = env->GetByteArrayElements(jkey, 0);
  rocksdb::Slice key_slice(reinterpret_cast<char*>(key), jkey_len);

  rocksdb::Status s;
  if (cf_handle != nullptr) {
    s = db->Delete(write_options, cf_handle, key_slice);
  } else {
    // backwards compatibility
    s = db->Delete(write_options, key_slice);
  }
  // trigger java unref on key and value.
  // by passing JNI_ABORT, it will simply release the reference without
  // copying the result back to the java byte array.
  env->ReleaseByteArrayElements(jkey, key, JNI_ABORT);

  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
  return;
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    remove
 * Signature: (J[BI)V
 */
void Java_org_rocksdb_RocksDB_remove__J_3BI(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jbyteArray jkey, jint jkey_len) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  static const rocksdb::WriteOptions default_write_options =
      rocksdb::WriteOptions();
  rocksdb_remove_helper(env, db, default_write_options, nullptr,
      jkey, jkey_len);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    remove
 * Signature: (J[BIJ)V
 */
void Java_org_rocksdb_RocksDB_remove__J_3BIJ(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jbyteArray jkey, jint jkey_len, jlong jcf_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  static const rocksdb::WriteOptions default_write_options =
      rocksdb::WriteOptions();
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  if (cf_handle != nullptr) {
    rocksdb_remove_helper(env, db, default_write_options, cf_handle,
        jkey, jkey_len);
  } else {
    rocksdb::RocksDBExceptionJni::ThrowNew(env,
        rocksdb::Status::InvalidArgument("Invalid ColumnFamilyHandle."));
  }
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    remove
 * Signature: (JJ[BIJ)V
 */
void Java_org_rocksdb_RocksDB_remove__JJ_3BI(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jlong jwrite_options, jbyteArray jkey, jint jkey_len) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto write_options = reinterpret_cast<rocksdb::WriteOptions*>(jwrite_options);
  rocksdb_remove_helper(env, db, *write_options, nullptr, jkey, jkey_len);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    remove
 * Signature: (JJ[BIJ)V
 */
void Java_org_rocksdb_RocksDB_remove__JJ_3BIJ(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jlong jwrite_options, jbyteArray jkey, jint jkey_len,
    jlong jcf_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto write_options = reinterpret_cast<rocksdb::WriteOptions*>(jwrite_options);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  if (cf_handle != nullptr) {
    rocksdb_remove_helper(env, db, *write_options, cf_handle, jkey, jkey_len);
  } else {
    rocksdb::RocksDBExceptionJni::ThrowNew(env,
        rocksdb::Status::InvalidArgument("Invalid ColumnFamilyHandle."));
  }
}
//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB::Merge

void rocksdb_merge_helper(
    JNIEnv* env, rocksdb::DB* db, const rocksdb::WriteOptions& write_options,
    rocksdb::ColumnFamilyHandle* cf_handle, jbyteArray jkey, jint jkey_len,
    jbyteArray jentry_value, jint jentry_value_len) {

  jbyte* key = env->GetByteArrayElements(jkey, 0);
  jbyte* value = env->GetByteArrayElements(jentry_value, 0);
  rocksdb::Slice key_slice(reinterpret_cast<char*>(key), jkey_len);
  rocksdb::Slice value_slice(reinterpret_cast<char*>(value),
      jentry_value_len);

  rocksdb::Status s;
  if (cf_handle != nullptr) {
    s = db->Merge(write_options, cf_handle, key_slice, value_slice);
  } else {
    s = db->Merge(write_options, key_slice, value_slice);
  }

  // trigger java unref on key and value.
  // by passing JNI_ABORT, it will simply release the reference without
  // copying the result back to the java byte array.
  env->ReleaseByteArrayElements(jkey, key, JNI_ABORT);
  env->ReleaseByteArrayElements(jentry_value, value, JNI_ABORT);

  if (s.ok()) {
    return;
  }
  rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    merge
 * Signature: (J[BI[BI)V
 */
void Java_org_rocksdb_RocksDB_merge__J_3BI_3BI(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jbyteArray jkey, jint jkey_len,
    jbyteArray jentry_value, jint jentry_value_len) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  static const rocksdb::WriteOptions default_write_options =
      rocksdb::WriteOptions();

  rocksdb_merge_helper(env, db, default_write_options,
      nullptr, jkey, jkey_len, jentry_value, jentry_value_len);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    merge
 * Signature: (J[BI[BIJ)V
 */
void Java_org_rocksdb_RocksDB_merge__J_3BI_3BIJ(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jbyteArray jkey, jint jkey_len,
    jbyteArray jentry_value, jint jentry_value_len, jlong jcf_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  static const rocksdb::WriteOptions default_write_options =
      rocksdb::WriteOptions();
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  if (cf_handle != nullptr) {
    rocksdb_merge_helper(env, db, default_write_options,
        cf_handle, jkey, jkey_len, jentry_value, jentry_value_len);
  } else {
    rocksdb::RocksDBExceptionJni::ThrowNew(env,
        rocksdb::Status::InvalidArgument("Invalid ColumnFamilyHandle."));
  }
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    merge
 * Signature: (JJ[BI[BI)V
 */
void Java_org_rocksdb_RocksDB_merge__JJ_3BI_3BI(
    JNIEnv* env, jobject jdb,
    jlong jdb_handle, jlong jwrite_options_handle,
    jbyteArray jkey, jint jkey_len,
    jbyteArray jentry_value, jint jentry_value_len) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto write_options = reinterpret_cast<rocksdb::WriteOptions*>(
      jwrite_options_handle);

  rocksdb_merge_helper(env, db, *write_options,
      nullptr, jkey, jkey_len, jentry_value, jentry_value_len);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    merge
 * Signature: (JJ[BI[BIJ)V
 */
void Java_org_rocksdb_RocksDB_merge__JJ_3BI_3BIJ(
    JNIEnv* env, jobject jdb,
    jlong jdb_handle, jlong jwrite_options_handle,
    jbyteArray jkey, jint jkey_len,
    jbyteArray jentry_value, jint jentry_value_len, jlong jcf_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto write_options = reinterpret_cast<rocksdb::WriteOptions*>(
      jwrite_options_handle);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  if (cf_handle != nullptr) {
    rocksdb_merge_helper(env, db, *write_options,
        cf_handle, jkey, jkey_len, jentry_value, jentry_value_len);
  } else {
    rocksdb::RocksDBExceptionJni::ThrowNew(env,
        rocksdb::Status::InvalidArgument("Invalid ColumnFamilyHandle."));
  }
}

//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB::~DB()

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    disposeInternal
 * Signature: (J)V
 */
void Java_org_rocksdb_RocksDB_disposeInternal(
    JNIEnv* env, jobject java_db, jlong jhandle) {
  delete reinterpret_cast<rocksdb::DB*>(jhandle);
}

jlong rocksdb_iterator_helper(
    rocksdb::DB* db, rocksdb::ReadOptions read_options,
    rocksdb::ColumnFamilyHandle* cf_handle) {
  rocksdb::Iterator* iterator = nullptr;
  if (cf_handle != nullptr) {
    iterator = db->NewIterator(read_options, cf_handle);
  } else {
    iterator = db->NewIterator(read_options);
  }
  return reinterpret_cast<jlong>(iterator);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    iterator
 * Signature: (J)J
 */
jlong Java_org_rocksdb_RocksDB_iterator__J(
    JNIEnv* env, jobject jdb, jlong db_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(db_handle);
  return rocksdb_iterator_helper(db, rocksdb::ReadOptions(),
      nullptr);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    iterator
 * Signature: (JJ)J
 */
jlong Java_org_rocksdb_RocksDB_iterator__JJ(
    JNIEnv* env, jobject jdb, jlong db_handle,
    jlong jread_options_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(db_handle);
  auto& read_options = *reinterpret_cast<rocksdb::ReadOptions*>(
      jread_options_handle);
  return rocksdb_iterator_helper(db, read_options,
      nullptr);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    iteratorCF
 * Signature: (JJ)J
 */
jlong Java_org_rocksdb_RocksDB_iteratorCF__JJ(
    JNIEnv* env, jobject jdb, jlong db_handle, jlong jcf_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(db_handle);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  return rocksdb_iterator_helper(db, rocksdb::ReadOptions(),
        cf_handle);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    iteratorCF
 * Signature: (JJJ)J
 */
jlong Java_org_rocksdb_RocksDB_iteratorCF__JJJ(
    JNIEnv* env, jobject jdb, jlong db_handle, jlong jcf_handle,
    jlong jread_options_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(db_handle);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  auto& read_options = *reinterpret_cast<rocksdb::ReadOptions*>(
      jread_options_handle);
  return rocksdb_iterator_helper(db, read_options,
        cf_handle);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    iterators
 * Signature: (JLjava/util/List;J)[J
 */
jlongArray Java_org_rocksdb_RocksDB_iterators(
    JNIEnv* env, jobject jdb, jlong db_handle, jobject jcfhandle_list,
    jlong jread_options_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(db_handle);
  auto& read_options = *reinterpret_cast<rocksdb::ReadOptions*>(
        jread_options_handle);
  std::vector<rocksdb::ColumnFamilyHandle*> cf_handles;
  std::vector<rocksdb::Iterator*> iterators;

  if (jcfhandle_list != nullptr) {
    // get cf iterator
    jobject cfIteratorObj = env->CallObjectMethod(
        jcfhandle_list, rocksdb::ListJni::getIteratorMethod(env));

    // iterate over keys and convert java byte array to slice
    while (env->CallBooleanMethod(
        cfIteratorObj, rocksdb::ListJni::getHasNextMethod(env)) == JNI_TRUE) {
      jobject jobj = (jbyteArray) env->CallObjectMethod(
          cfIteratorObj, rocksdb::ListJni::getNextMethod(env));
      rocksdb::ColumnFamilyHandle* cfHandle =
          rocksdb::ColumnFamilyHandleJni::getHandle(env, jobj);
      cf_handles.push_back(cfHandle);
    }
  }

  rocksdb::Status s = db->NewIterators(read_options,
      cf_handles, &iterators);
  if (s.ok()) {
    jlongArray jLongArray =
        env->NewLongArray(static_cast<jsize>(iterators.size()));
    for (std::vector<rocksdb::Iterator*>::size_type i = 0; i < iterators.size();
         i++) {
      env->SetLongArrayRegion(jLongArray, static_cast<jsize>(i), 1,
                              reinterpret_cast<const jlong*>(&iterators[i]));
    }
    return jLongArray;
  }
  rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  return env->NewLongArray(0);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    getDefaultColumnFamily
 * Signature: (J)J
 */
jlong Java_org_rocksdb_RocksDB_getDefaultColumnFamily(
    JNIEnv* env, jobject jobj, jlong jdb_handle) {
  auto* db_handle = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto* cf_handle = db_handle->DefaultColumnFamily();
  return reinterpret_cast<jlong>(cf_handle);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    createColumnFamily
 * Signature: (JLorg/rocksdb/ColumnFamilyDescriptor;)J;
 */
jlong Java_org_rocksdb_RocksDB_createColumnFamily(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jobject jcf_descriptor) {
  rocksdb::ColumnFamilyHandle* handle;
  auto db_handle = reinterpret_cast<rocksdb::DB*>(jdb_handle);

  // get ColumnFamilyName
  jbyteArray byteArray = static_cast<jbyteArray>(env->CallObjectMethod(
      jcf_descriptor,
      rocksdb::ColumnFamilyDescriptorJni::getColumnFamilyNameMethod(
          env)));
  // get CF Options
  jobject jcf_opt_obj = env->CallObjectMethod(jcf_descriptor,
      rocksdb::ColumnFamilyDescriptorJni::getColumnFamilyOptionsMethod(
          env));
  rocksdb::ColumnFamilyOptions* cfOptions =
      rocksdb::ColumnFamilyOptionsJni::getHandle(env, jcf_opt_obj);

  jbyte* cfname = env->GetByteArrayElements(byteArray, 0);
  const int len = env->GetArrayLength(byteArray);

  rocksdb::Status s = db_handle->CreateColumnFamily(
      *cfOptions, std::string(reinterpret_cast<char *>(cfname), len), &handle);
  env->ReleaseByteArrayElements(byteArray, cfname, 0);

  if (s.ok()) {
    return reinterpret_cast<jlong>(handle);
  }
  rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  return 0;
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    dropColumnFamily
 * Signature: (JJ)V;
 */
void Java_org_rocksdb_RocksDB_dropColumnFamily(
    JNIEnv* env, jobject jdb, jlong jdb_handle, jlong jcf_handle) {
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  auto db_handle = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  rocksdb::Status s = db_handle->DropColumnFamily(cf_handle);
  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Method:    getSnapshot
 * Signature: (J)J
 */
jlong Java_org_rocksdb_RocksDB_getSnapshot(
    JNIEnv* env, jobject jdb, jlong db_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(db_handle);
  const rocksdb::Snapshot* snapshot = db->GetSnapshot();
  return reinterpret_cast<jlong>(snapshot);
}

/*
 * Method:    releaseSnapshot
 * Signature: (JJ)V
 */
void Java_org_rocksdb_RocksDB_releaseSnapshot(
    JNIEnv* env, jobject jdb, jlong db_handle, jlong snapshot_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(db_handle);
  auto snapshot = reinterpret_cast<rocksdb::Snapshot*>(snapshot_handle);
  db->ReleaseSnapshot(snapshot);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    getProperty0
 * Signature: (JLjava/lang/String;I)Ljava/lang/String;
 */
jstring Java_org_rocksdb_RocksDB_getProperty0__JLjava_lang_String_2I(
    JNIEnv* env, jobject jdb, jlong db_handle, jstring jproperty,
    jint jproperty_len) {
  auto db = reinterpret_cast<rocksdb::DB*>(db_handle);

  const char* property = env->GetStringUTFChars(jproperty, 0);
  rocksdb::Slice property_slice(property, jproperty_len);

  std::string property_value;
  bool retCode = db->GetProperty(property_slice, &property_value);
  env->ReleaseStringUTFChars(jproperty, property);

  if (!retCode) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, rocksdb::Status::NotFound());
  }

  return env->NewStringUTF(property_value.data());
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    getProperty0
 * Signature: (JJLjava/lang/String;I)Ljava/lang/String;
 */
jstring Java_org_rocksdb_RocksDB_getProperty0__JJLjava_lang_String_2I(
    JNIEnv* env, jobject jdb, jlong db_handle, jlong jcf_handle,
    jstring jproperty, jint jproperty_len) {
  auto db = reinterpret_cast<rocksdb::DB*>(db_handle);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);

  const char* property = env->GetStringUTFChars(jproperty, 0);
  rocksdb::Slice property_slice(property, jproperty_len);

  std::string property_value;
  bool retCode = db->GetProperty(cf_handle, property_slice, &property_value);
  env->ReleaseStringUTFChars(jproperty, property);

  if (!retCode) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, rocksdb::Status::NotFound());
  }

  return env->NewStringUTF(property_value.data());
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    getLongProperty
 * Signature: (JLjava/lang/String;I)L;
 */
jlong Java_org_rocksdb_RocksDB_getLongProperty__JLjava_lang_String_2I(
    JNIEnv* env, jobject jdb, jlong db_handle, jstring jproperty,
    jint jproperty_len) {
  auto db = reinterpret_cast<rocksdb::DB*>(db_handle);

  const char* property = env->GetStringUTFChars(jproperty, 0);
  rocksdb::Slice property_slice(property, jproperty_len);

  uint64_t property_value = 0;
  bool retCode = db->GetIntProperty(property_slice, &property_value);
  env->ReleaseStringUTFChars(jproperty, property);

  if (!retCode) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, rocksdb::Status::NotFound());
  }
  return property_value;
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    getLongProperty
 * Signature: (JJLjava/lang/String;I)L;
 */
jlong Java_org_rocksdb_RocksDB_getLongProperty__JJLjava_lang_String_2I(
    JNIEnv* env, jobject jdb, jlong db_handle, jlong jcf_handle,
    jstring jproperty, jint jproperty_len) {
  auto db = reinterpret_cast<rocksdb::DB*>(db_handle);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);

  const char* property = env->GetStringUTFChars(jproperty, 0);
  rocksdb::Slice property_slice(property, jproperty_len);

  uint64_t property_value;
  bool retCode = db->GetIntProperty(cf_handle, property_slice, &property_value);
  env->ReleaseStringUTFChars(jproperty, property);

  if (!retCode) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, rocksdb::Status::NotFound());
  }
  return property_value;
}

//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB::Flush

void rocksdb_flush_helper(
    JNIEnv* env, rocksdb::DB* db, const rocksdb::FlushOptions& flush_options,
  rocksdb::ColumnFamilyHandle* column_family_handle) {
  rocksdb::Status s;
  if (column_family_handle != nullptr) {
    s = db->Flush(flush_options, column_family_handle);
  } else {
    s = db->Flush(flush_options);
  }
  if (!s.ok()) {
      rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    flush
 * Signature: (JJ)V
 */
void Java_org_rocksdb_RocksDB_flush__JJ(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jlong jflush_options) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto flush_options = reinterpret_cast<rocksdb::FlushOptions*>(jflush_options);
  rocksdb_flush_helper(env, db, *flush_options, nullptr);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    flush
 * Signature: (JJJ)V
 */
void Java_org_rocksdb_RocksDB_flush__JJJ(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
    jlong jflush_options, jlong jcf_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto flush_options = reinterpret_cast<rocksdb::FlushOptions*>(jflush_options);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  rocksdb_flush_helper(env, db, *flush_options, cf_handle);
}

//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB::CompactRange - Full

void rocksdb_compactrange_helper(JNIEnv* env, rocksdb::DB* db,
    rocksdb::ColumnFamilyHandle* cf_handle, jboolean jreduce_level,
    jint jtarget_level, jint jtarget_path_id) {

  rocksdb::Status s;
  rocksdb::CompactRangeOptions compact_options;
  compact_options.change_level = jreduce_level;
  compact_options.target_level = jtarget_level;
  compact_options.target_path_id = static_cast<uint32_t>(jtarget_path_id);
  if (cf_handle != nullptr) {
    s = db->CompactRange(compact_options, cf_handle, nullptr, nullptr);
  } else {
    // backwards compatibility
    s = db->CompactRange(compact_options, nullptr, nullptr);
  }

  if (s.ok()) {
    return;
  }
  rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    compactRange0
 * Signature: (JZII)V
 */
void Java_org_rocksdb_RocksDB_compactRange0__JZII(JNIEnv* env,
    jobject jdb, jlong jdb_handle, jboolean jreduce_level,
    jint jtarget_level, jint jtarget_path_id) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  rocksdb_compactrange_helper(env, db, nullptr, jreduce_level,
      jtarget_level, jtarget_path_id);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    compactRange
 * Signature: (JZIIJ)V
 */
void Java_org_rocksdb_RocksDB_compactRange__JZIIJ(
    JNIEnv* env, jobject jdb, jlong jdb_handle,
     jboolean jreduce_level, jint jtarget_level,
     jint jtarget_path_id, jlong jcf_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  rocksdb_compactrange_helper(env, db, cf_handle, jreduce_level,
      jtarget_level, jtarget_path_id);
}

//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB::CompactRange - Range

void rocksdb_compactrange_helper(JNIEnv* env, rocksdb::DB* db,
    rocksdb::ColumnFamilyHandle* cf_handle, jbyteArray jbegin, jint jbegin_len,
    jbyteArray jend, jint jend_len, jboolean jreduce_level, jint jtarget_level,
    jint jtarget_path_id) {

  jbyte* begin = env->GetByteArrayElements(jbegin, 0);
  jbyte* end = env->GetByteArrayElements(jend, 0);
  const rocksdb::Slice begin_slice(reinterpret_cast<char*>(begin), jbegin_len);
  const rocksdb::Slice end_slice(reinterpret_cast<char*>(end), jend_len);

  rocksdb::Status s;
  rocksdb::CompactRangeOptions compact_options;
  compact_options.change_level = jreduce_level;
  compact_options.target_level = jtarget_level;
  compact_options.target_path_id = static_cast<uint32_t>(jtarget_path_id);
  if (cf_handle != nullptr) {
    s = db->CompactRange(compact_options, cf_handle, &begin_slice, &end_slice);
  } else {
    // backwards compatibility
    s = db->CompactRange(compact_options, &begin_slice, &end_slice);
  }

  env->ReleaseByteArrayElements(jbegin, begin, JNI_ABORT);
  env->ReleaseByteArrayElements(jend, end, JNI_ABORT);

  if (s.ok()) {
    return;
  }
  rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    compactRange0
 * Signature: (J[BI[BIZII)V
 */
void Java_org_rocksdb_RocksDB_compactRange0__J_3BI_3BIZII(JNIEnv* env,
    jobject jdb, jlong jdb_handle, jbyteArray jbegin, jint jbegin_len,
    jbyteArray jend, jint jend_len, jboolean jreduce_level,
    jint jtarget_level, jint jtarget_path_id) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  rocksdb_compactrange_helper(env, db, nullptr, jbegin, jbegin_len,
      jend, jend_len, jreduce_level, jtarget_level, jtarget_path_id);
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    compactRange
 * Signature: (JJ[BI[BIZII)V
 */
void Java_org_rocksdb_RocksDB_compactRange__J_3BI_3BIZIIJ(
    JNIEnv* env, jobject jdb, jlong jdb_handle, jbyteArray jbegin,
    jint jbegin_len, jbyteArray jend, jint jend_len,
    jboolean jreduce_level, jint jtarget_level,
    jint jtarget_path_id, jlong jcf_handle) {
  auto db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  auto cf_handle = reinterpret_cast<rocksdb::ColumnFamilyHandle*>(jcf_handle);
  rocksdb_compactrange_helper(env, db, cf_handle, jbegin, jbegin_len,
      jend, jend_len, jreduce_level, jtarget_level, jtarget_path_id);
}

//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB::GetLatestSequenceNumber

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    getLatestSequenceNumber
 * Signature: (J)V
 */
jlong Java_org_rocksdb_RocksDB_getLatestSequenceNumber(JNIEnv* env,
    jobject jdb, jlong jdb_handle) {
  auto* db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  return db->GetLatestSequenceNumber();
}

//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB enable/disable file deletions

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    enableFileDeletions
 * Signature: (J)V
 */
void Java_org_rocksdb_RocksDB_disableFileDeletions(JNIEnv* env,
    jobject jdb, jlong jdb_handle) {
  auto* db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  rocksdb::Status s = db->DisableFileDeletions();
  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    enableFileDeletions
 * Signature: (JZ)V
 */
void Java_org_rocksdb_RocksDB_enableFileDeletions(JNIEnv* env,
    jobject jdb, jlong jdb_handle, jboolean jforce) {
  auto* db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  rocksdb::Status s = db->EnableFileDeletions(jforce);
  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

//////////////////////////////////////////////////////////////////////////////
// rocksdb::DB::GetUpdatesSince

/*
 * Class:     org_rocksdb_RocksDB
 * Method:    getUpdatesSince
 * Signature: (JJ)J
 */
jlong Java_org_rocksdb_RocksDB_getUpdatesSince(JNIEnv* env,
    jobject jdb, jlong jdb_handle, jlong jsequence_number) {
  auto* db = reinterpret_cast<rocksdb::DB*>(jdb_handle);
  rocksdb::SequenceNumber sequence_number =
      static_cast<rocksdb::SequenceNumber>(jsequence_number);
  std::unique_ptr<rocksdb::TransactionLogIterator> iter;
  rocksdb::Status s = db->GetUpdatesSince(sequence_number, &iter);
  if (s.ok()) {
    return reinterpret_cast<jlong>(iter.release());
  }
  rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  return 0;
}
