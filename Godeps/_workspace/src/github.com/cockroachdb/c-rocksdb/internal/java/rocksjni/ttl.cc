// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// This file implements the "bridge" between Java and C++ and enables
// calling c++ rocksdb::TtlDB methods.
// from Java side.

#include <jni.h>
#include <stdio.h>
#include <stdlib.h>
#include <string>
#include <vector>

#include "include/org_rocksdb_TtlDB.h"
#include "rocksdb/utilities/db_ttl.h"
#include "rocksjni/portal.h"

/*
 * Class:     org_rocksdb_TtlDB
 * Method:    open
 * Signature: (JLjava/lang/String;IZ)V
 */
void Java_org_rocksdb_TtlDB_open(JNIEnv* env,
    jobject jttldb, jlong joptions_handle, jstring jdb_path,
    jint jttl, jboolean jread_only) {
  auto* opt = reinterpret_cast<rocksdb::Options*>(joptions_handle);
  rocksdb::DBWithTTL* db = nullptr;
  const char* db_path = env->GetStringUTFChars(jdb_path, 0);
  rocksdb::Status s = rocksdb::DBWithTTL::Open(*opt, db_path, &db,
      jttl, jread_only);
  env->ReleaseStringUTFChars(jdb_path, db_path);

  // as TTLDB extends RocksDB on the java side, we can reuse
  // the RocksDB portal here.
  if (s.ok()) {
      rocksdb::RocksDBJni::setHandle(env, jttldb, db);
      return;
  }
  rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
}

/*
 * Class:     org_rocksdb_TtlDB
 * Method:    openCF
 * Signature: (JLjava/lang/String;Ljava/util/List;
 *    ILjava/util/List;Z)Ljava/util/List;
 */
jobject
    Java_org_rocksdb_TtlDB_openCF(
    JNIEnv* env, jobject jdb, jlong jopt_handle, jstring jdb_path,
    jobject jcfdesc_list, jint jcfdesc_count, jobject jttl_list,
    jboolean jread_only) {
  auto* opt = reinterpret_cast<rocksdb::Options*>(jopt_handle);
  rocksdb::DBWithTTL* db = nullptr;
  const char* db_path = env->GetStringUTFChars(jdb_path, 0);

  std::vector<jbyte*> cfnames_to_free;
  std::vector<jbyteArray> jcfnames_for_free;

  std::vector<rocksdb::ColumnFamilyDescriptor> column_families;
  std::vector<int32_t> ttl_values;
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

      // free allocated cfnames after call to open
      cfnames_to_free.push_back(cfname);
      jcfnames_for_free.push_back(byteArray);
      column_families.push_back(rocksdb::ColumnFamilyDescriptor(
          std::string(reinterpret_cast<char *>(cfname), len), *cfOptions));
  }
  // get iterator for TTL values
  iteratorObj = env->CallObjectMethod(
        jttl_list, rocksdb::ListJni::getIteratorMethod(env));
  // iterate over TTL values
  while (env->CallBooleanMethod(
      iteratorObj, rocksdb::ListJni::getHasNextMethod(env)) == JNI_TRUE) {
     // get TTL object
     jobject jttl_object = env->CallObjectMethod(iteratorObj,
       rocksdb::ListJni::getNextMethod(env));
     // get Integer value
     jclass jIntClazz = env->FindClass("java/lang/Integer");
     jmethodID getVal = env->GetMethodID(jIntClazz, "intValue", "()I");
     ttl_values.push_back(env->CallIntMethod(jttl_object, getVal));
  }
  rocksdb::Status s = rocksdb::DBWithTTL::Open(*opt, db_path, column_families,
      &handles, &db, ttl_values, jread_only);

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
 * Class:     org_rocksdb_TtlDB
 * Method:    createColumnFamilyWithTtl
 * Signature: (JLorg/rocksdb/ColumnFamilyDescriptor;I)J;
 */
jlong Java_org_rocksdb_TtlDB_createColumnFamilyWithTtl(
    JNIEnv* env, jobject jobj, jlong jdb_handle,
    jobject jcf_descriptor, jint jttl) {
  rocksdb::ColumnFamilyHandle* handle;
  auto* db_handle = reinterpret_cast<rocksdb::DBWithTTL*>(jdb_handle);

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

  rocksdb::Status s = db_handle->CreateColumnFamilyWithTtl(
      *cfOptions, std::string(reinterpret_cast<char *>(cfname),
          len), &handle, jttl);
  env->ReleaseByteArrayElements(byteArray, cfname, 0);

  if (s.ok()) {
    return reinterpret_cast<jlong>(handle);
  }
  rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  return 0;
}
