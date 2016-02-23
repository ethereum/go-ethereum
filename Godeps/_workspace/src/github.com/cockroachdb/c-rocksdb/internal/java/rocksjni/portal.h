// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

// This file is designed for caching those frequently used IDs and provide
// efficient portal (i.e, a set of static functions) to access java code
// from c++.

#ifndef JAVA_ROCKSJNI_PORTAL_H_
#define JAVA_ROCKSJNI_PORTAL_H_

#include <jni.h>
#include <limits>
#include <string>
#include <vector>

#include "rocksdb/db.h"
#include "rocksdb/filter_policy.h"
#include "rocksdb/status.h"
#include "rocksdb/utilities/backupable_db.h"
#include "rocksdb/utilities/write_batch_with_index.h"
#include "rocksjni/comparatorjnicallback.h"
#include "rocksjni/loggerjnicallback.h"
#include "rocksjni/writebatchhandlerjnicallback.h"

namespace rocksdb {

// Detect if jlong overflows size_t
inline Status check_if_jlong_fits_size_t(const jlong& jvalue) {
  Status s = Status::OK();
  if (static_cast<uint64_t>(jvalue) > std::numeric_limits<size_t>::max()) {
    s = Status::InvalidArgument(Slice("jlong overflows 32 bit value."));
  }
  return s;
}

// Native class template
template<class PTR, class DERIVED> class RocksDBNativeClass {
 public:
  // Get the java class id
  static jclass getJClass(JNIEnv* env, const char* jclazz_name) {
    jclass jclazz = env->FindClass(jclazz_name);
    assert(jclazz != nullptr);
    return jclazz;
  }

  // Get the field id of the member variable to store
  // the ptr
  static jfieldID getHandleFieldID(JNIEnv* env) {
    static jfieldID fid = env->GetFieldID(
        DERIVED::getJClass(env), "nativeHandle_", "J");
    assert(fid != nullptr);
    return fid;
  }

  // Get the pointer from Java
  static PTR getHandle(JNIEnv* env, jobject jobj) {
    return reinterpret_cast<PTR>(
        env->GetLongField(jobj, getHandleFieldID(env)));
  }

  // Pass the pointer to the java side.
  static void setHandle(JNIEnv* env, jobject jdb, PTR ptr) {
    env->SetLongField(
        jdb, getHandleFieldID(env),
        reinterpret_cast<jlong>(ptr));
  }
};

// Java Exception template
template<class DERIVED> class RocksDBJavaException {
 public:
  // Get the java class id
  static jclass getJClass(JNIEnv* env, const char* jclazz_name) {
    jclass jclazz = env->FindClass(jclazz_name);
    assert(jclazz != nullptr);
    return jclazz;
  }

  // Create and throw a java exception by converting the input
  // Status.
  //
  // In case s.ok() is true, then this function will not throw any
  // exception.
  static void ThrowNew(JNIEnv* env, Status s) {
    if (s.ok()) {
      return;
    }
    jstring msg = env->NewStringUTF(s.ToString().c_str());
    // get the constructor id of org.rocksdb.RocksDBException
    static jmethodID mid = env->GetMethodID(
        DERIVED::getJClass(env), "<init>", "(Ljava/lang/String;)V");
    assert(mid != nullptr);

    env->Throw((jthrowable)env->NewObject(DERIVED::getJClass(env),
        mid, msg));
  }
};

// The portal class for org.rocksdb.RocksDB
class RocksDBJni : public RocksDBNativeClass<rocksdb::DB*, RocksDBJni> {
 public:
  // Get the java class id of org.rocksdb.RocksDB.
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env, "org/rocksdb/RocksDB");
  }
};

// The portal class for org.rocksdb.RocksDBException
class RocksDBExceptionJni :
    public RocksDBJavaException<RocksDBExceptionJni> {
 public:
  // Get the java class id of java.lang.IllegalArgumentException
  static jclass getJClass(JNIEnv* env) {
    return RocksDBJavaException::getJClass(env,
        "org/rocksdb/RocksDBException");
  }
};

// The portal class for java.lang.IllegalArgumentException
class IllegalArgumentExceptionJni :
    public RocksDBJavaException<IllegalArgumentExceptionJni> {
 public:
  // Get the java class id of java.lang.IllegalArgumentException
  static jclass getJClass(JNIEnv* env) {
    return RocksDBJavaException::getJClass(env,
        "java/lang/IllegalArgumentException");
  }
};


// The portal class for org.rocksdb.Options
class OptionsJni : public RocksDBNativeClass<
    rocksdb::Options*, OptionsJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env, "org/rocksdb/Options");
  }
};

// The portal class for org.rocksdb.DBOptions
class DBOptionsJni : public RocksDBNativeClass<
    rocksdb::DBOptions*, DBOptionsJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env, "org/rocksdb/DBOptions");
  }
};

class ColumnFamilyDescriptorJni {
 public:
  // Get the java class id of org.rocksdb.ColumnFamilyDescriptor
  static jclass getColumnFamilyDescriptorClass(JNIEnv* env) {
    jclass jclazz = env->FindClass("org/rocksdb/ColumnFamilyDescriptor");
    assert(jclazz != nullptr);
    return jclazz;
  }

  // Get the java method id of columnFamilyName
  static jmethodID getColumnFamilyNameMethod(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(
        getColumnFamilyDescriptorClass(env),
        "columnFamilyName", "()[B");
    assert(mid != nullptr);
    return mid;
  }

  // Get the java method id of columnFamilyOptions
  static jmethodID getColumnFamilyOptionsMethod(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(
        getColumnFamilyDescriptorClass(env),
        "columnFamilyOptions", "()Lorg/rocksdb/ColumnFamilyOptions;");
    assert(mid != nullptr);
    return mid;
  }
};

// The portal class for org.rocksdb.ColumnFamilyOptions
class ColumnFamilyOptionsJni : public RocksDBNativeClass<
    rocksdb::ColumnFamilyOptions*, ColumnFamilyOptionsJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/ColumnFamilyOptions");
  }
};

// The portal class for org.rocksdb.WriteOptions
class WriteOptionsJni : public RocksDBNativeClass<
    rocksdb::WriteOptions*, WriteOptionsJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/WriteOptions");
  }
};

// The portal class for org.rocksdb.ReadOptions
class ReadOptionsJni : public RocksDBNativeClass<
    rocksdb::ReadOptions*, ReadOptionsJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/ReadOptions");
  }
};

// The portal class for org.rocksdb.ReadOptions
class WriteBatchJni : public RocksDBNativeClass<
    rocksdb::WriteBatch*, WriteBatchJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/WriteBatch");
  }
};

// The portal class for org.rocksdb.WriteBatch.Handler
class WriteBatchHandlerJni : public RocksDBNativeClass<
    const rocksdb::WriteBatchHandlerJniCallback*,
    WriteBatchHandlerJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/WriteBatch$Handler");
  }

  // Get the java method `put` of org.rocksdb.WriteBatch.Handler.
  static jmethodID getPutMethodId(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(
        getJClass(env), "put", "([B[B)V");
    assert(mid != nullptr);
    return mid;
  }

  // Get the java method `merge` of org.rocksdb.WriteBatch.Handler.
  static jmethodID getMergeMethodId(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(
        getJClass(env), "merge", "([B[B)V");
    assert(mid != nullptr);
    return mid;
  }

  // Get the java method `delete` of org.rocksdb.WriteBatch.Handler.
  static jmethodID getDeleteMethodId(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(
        getJClass(env), "delete", "([B)V");
    assert(mid != nullptr);
    return mid;
  }

  // Get the java method `logData` of org.rocksdb.WriteBatch.Handler.
  static jmethodID getLogDataMethodId(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(
        getJClass(env), "logData", "([B)V");
    assert(mid != nullptr);
    return mid;
  }

  // Get the java method `shouldContinue` of org.rocksdb.WriteBatch.Handler.
  static jmethodID getContinueMethodId(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(
        getJClass(env), "shouldContinue", "()Z");
    assert(mid != nullptr);
    return mid;
  }
};

// The portal class for org.rocksdb.WriteBatchWithIndex
class WriteBatchWithIndexJni : public RocksDBNativeClass<
    rocksdb::WriteBatchWithIndex*, WriteBatchWithIndexJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/WriteBatch");
  }
};

class HistogramDataJni {
 public:
  static jmethodID getConstructorMethodId(JNIEnv* env, jclass jclazz) {
    static jmethodID mid = env->GetMethodID(jclazz, "<init>", "(DDDDD)V");
    assert(mid != nullptr);
    return mid;
  }
};

// The portal class for org.rocksdb.WriteBatchWithIndex
class BackupableDBOptionsJni : public RocksDBNativeClass<
    rocksdb::BackupableDBOptions*, BackupableDBOptionsJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/BackupableDBOptions");
  }
};

class BackupEngineJni : public RocksDBNativeClass<
    rocksdb::BackupEngine*, BackupEngineJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/BackupEngine");
  }
};

// The portal class for org.rocksdb.RocksIterator
class IteratorJni : public RocksDBNativeClass<
    rocksdb::Iterator*, IteratorJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/RocksIterator");
  }
};

// The portal class for org.rocksdb.Filter
class FilterJni : public RocksDBNativeClass<
    std::shared_ptr<rocksdb::FilterPolicy>*, FilterJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/Filter");
  }
};

// The portal class for org.rocksdb.ColumnFamilyHandle
class ColumnFamilyHandleJni : public RocksDBNativeClass<
    rocksdb::ColumnFamilyHandle*, ColumnFamilyHandleJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/ColumnFamilyHandle");
  }
};

// The portal class for org.rocksdb.FlushOptions
class FlushOptionsJni : public RocksDBNativeClass<
    rocksdb::FlushOptions*, FlushOptionsJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/FlushOptions");
  }
};

// The portal class for org.rocksdb.ComparatorOptions
class ComparatorOptionsJni : public RocksDBNativeClass<
    rocksdb::ComparatorJniCallbackOptions*, ComparatorOptionsJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/ComparatorOptions");
  }
};

// The portal class for org.rocksdb.AbstractComparator
class AbstractComparatorJni : public RocksDBNativeClass<
    const rocksdb::BaseComparatorJniCallback*,
    AbstractComparatorJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/AbstractComparator");
  }

  // Get the java method `name` of org.rocksdb.Comparator.
  static jmethodID getNameMethodId(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(
        getJClass(env), "name", "()Ljava/lang/String;");
    assert(mid != nullptr);
    return mid;
  }

  // Get the java method `compare` of org.rocksdb.Comparator.
  static jmethodID getCompareMethodId(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(getJClass(env),
      "compare",
      "(Lorg/rocksdb/AbstractSlice;Lorg/rocksdb/AbstractSlice;)I");
    assert(mid != nullptr);
    return mid;
  }

  // Get the java method `findShortestSeparator` of org.rocksdb.Comparator.
  static jmethodID getFindShortestSeparatorMethodId(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(getJClass(env),
      "findShortestSeparator",
      "(Ljava/lang/String;Lorg/rocksdb/AbstractSlice;)Ljava/lang/String;");
    assert(mid != nullptr);
    return mid;
  }

  // Get the java method `findShortSuccessor` of org.rocksdb.Comparator.
  static jmethodID getFindShortSuccessorMethodId(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(getJClass(env),
      "findShortSuccessor",
      "(Ljava/lang/String;)Ljava/lang/String;");
    assert(mid != nullptr);
    return mid;
  }
};

// The portal class for org.rocksdb.AbstractSlice
class AbstractSliceJni : public RocksDBNativeClass<
    const rocksdb::Slice*, AbstractSliceJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/AbstractSlice");
  }
};

class SliceJni {
 public:
  // Get the java class id of org.rocksdb.Slice.
  static jclass getJClass(JNIEnv* env) {
    jclass jclazz = env->FindClass("org/rocksdb/Slice");
    assert(jclazz != nullptr);
    return jclazz;
  }

  static jobject construct0(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(getJClass(env), "<init>", "()V");
    assert(mid != nullptr);
    return env->NewObject(getJClass(env), mid);
  }
};

class DirectSliceJni {
 public:
  // Get the java class id of org.rocksdb.DirectSlice.
  static jclass getJClass(JNIEnv* env) {
    jclass jclazz = env->FindClass("org/rocksdb/DirectSlice");
    assert(jclazz != nullptr);
    return jclazz;
  }

  static jobject construct0(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(getJClass(env), "<init>", "()V");
    assert(mid != nullptr);
    return env->NewObject(getJClass(env), mid);
  }
};

class ListJni {
 public:
  // Get the java class id of java.util.List.
  static jclass getListClass(JNIEnv* env) {
    jclass jclazz = env->FindClass("java/util/List");
    assert(jclazz != nullptr);
    return jclazz;
  }

  // Get the java class id of java.util.ArrayList.
  static jclass getArrayListClass(JNIEnv* env) {
    jclass jclazz = env->FindClass("java/util/ArrayList");
    assert(jclazz != nullptr);
    return jclazz;
  }

  // Get the java class id of java.util.Iterator.
  static jclass getIteratorClass(JNIEnv* env) {
    jclass jclazz = env->FindClass("java/util/Iterator");
    assert(jclazz != nullptr);
    return jclazz;
  }

  // Get the java method id of java.util.List.iterator().
  static jmethodID getIteratorMethod(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(
        getListClass(env), "iterator", "()Ljava/util/Iterator;");
    assert(mid != nullptr);
    return mid;
  }

  // Get the java method id of java.util.Iterator.hasNext().
  static jmethodID getHasNextMethod(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(
        getIteratorClass(env), "hasNext", "()Z");
    assert(mid != nullptr);
    return mid;
  }

  // Get the java method id of java.util.Iterator.next().
  static jmethodID getNextMethod(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(
        getIteratorClass(env), "next", "()Ljava/lang/Object;");
    assert(mid != nullptr);
    return mid;
  }

  // Get the java method id of arrayList constructor.
  static jmethodID getArrayListConstructorMethodId(JNIEnv* env, jclass jclazz) {
    static jmethodID mid = env->GetMethodID(
        jclazz, "<init>", "(I)V");
    assert(mid != nullptr);
    return mid;
  }

  // Get the java method id of java.util.List.add().
  static jmethodID getListAddMethodId(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(
        getListClass(env), "add", "(Ljava/lang/Object;)Z");
    assert(mid != nullptr);
    return mid;
  }
};

class ByteJni {
 public:
  // Get the java class id of java.lang.Byte.
  static jclass getByteClass(JNIEnv* env) {
    jclass jclazz = env->FindClass("java/lang/Byte");
    assert(jclazz != nullptr);
    return jclazz;
  }

  // Get the java method id of java.lang.Byte.byteValue.
  static jmethodID getByteValueMethod(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(
        getByteClass(env), "byteValue", "()B");
    assert(mid != nullptr);
    return mid;
  }
};

class BackupInfoJni {
 public:
  // Get the java class id of org.rocksdb.BackupInfo.
  static jclass getJClass(JNIEnv* env) {
    jclass jclazz = env->FindClass("org/rocksdb/BackupInfo");
    assert(jclazz != nullptr);
    return jclazz;
  }

  static jobject construct0(JNIEnv* env, uint32_t backup_id, int64_t timestamp,
      uint64_t size, uint32_t number_files) {
    static jmethodID mid = env->GetMethodID(getJClass(env), "<init>",
        "(IJJI)V");
    assert(mid != nullptr);
    return env->NewObject(getJClass(env), mid,
        backup_id, timestamp, size, number_files);
  }
};

class BackupInfoListJni {
 public:
  static jobject getBackupInfo(JNIEnv* env,
      std::vector<BackupInfo> backup_infos) {
    jclass jclazz = env->FindClass("java/util/ArrayList");
    jmethodID mid = rocksdb::ListJni::getArrayListConstructorMethodId(
        env, jclazz);
    jobject jbackup_info_handle_list = env->NewObject(jclazz, mid,
        backup_infos.size());
    // insert in java list
    for (std::vector<rocksdb::BackupInfo>::size_type i = 0;
        i != backup_infos.size(); i++) {
      rocksdb::BackupInfo backup_info = backup_infos[i];
      jobject obj = rocksdb::BackupInfoJni::construct0(env,
          backup_info.backup_id,
          backup_info.timestamp,
          backup_info.size,
          backup_info.number_files);
      env->CallBooleanMethod(jbackup_info_handle_list,
          rocksdb::ListJni::getListAddMethodId(env), obj);
    }
    return jbackup_info_handle_list;
  }
};

class WBWIRocksIteratorJni {
 public:
    // Get the java class id of org.rocksdb.WBWIRocksIterator.
    static jclass getJClass(JNIEnv* env) {
      static jclass jclazz = env->FindClass("org/rocksdb/WBWIRocksIterator");
      assert(jclazz != nullptr);
      return jclazz;
    }

    static jfieldID getWriteEntryField(JNIEnv* env) {
      static jfieldID fid =
          env->GetFieldID(getJClass(env), "entry",
          "Lorg/rocksdb/WBWIRocksIterator$WriteEntry;");
      assert(fid != nullptr);
      return fid;
    }

    static jobject getWriteEntry(JNIEnv* env, jobject jwbwi_rocks_iterator) {
      jobject jwe =
          env->GetObjectField(jwbwi_rocks_iterator, getWriteEntryField(env));
      assert(jwe != nullptr);
      return jwe;
    }
};

class WriteTypeJni {
 public:
    // Get the PUT enum field of org.rocksdb.WBWIRocksIterator.WriteType
    static jobject PUT(JNIEnv* env) {
      return getEnum(env, "PUT");
    }

    // Get the MERGE enum field of org.rocksdb.WBWIRocksIterator.WriteType
    static jobject MERGE(JNIEnv* env) {
      return getEnum(env, "MERGE");
    }

    // Get the DELETE enum field of org.rocksdb.WBWIRocksIterator.WriteType
    static jobject DELETE(JNIEnv* env) {
      return getEnum(env, "DELETE");
    }

    // Get the LOG enum field of org.rocksdb.WBWIRocksIterator.WriteType
    static jobject LOG(JNIEnv* env) {
      return getEnum(env, "LOG");
    }

 private:
    // Get the java class id of org.rocksdb.WBWIRocksIterator.WriteType.
    static jclass getJClass(JNIEnv* env) {
      jclass jclazz = env->FindClass("org/rocksdb/WBWIRocksIterator$WriteType");
      assert(jclazz != nullptr);
      return jclazz;
    }

    // Get an enum field of org.rocksdb.WBWIRocksIterator.WriteType
    static jobject getEnum(JNIEnv* env, const char name[]) {
      jclass jclazz = getJClass(env);
      jfieldID jfid =
          env->GetStaticFieldID(jclazz, name,
          "Lorg/rocksdb/WBWIRocksIterator$WriteType;");
      assert(jfid != nullptr);
      return env->GetStaticObjectField(jclazz, jfid);
    }
};

class WriteEntryJni {
 public:
    // Get the java class id of org.rocksdb.WBWIRocksIterator.WriteEntry.
    static jclass getJClass(JNIEnv* env) {
      static jclass jclazz =
          env->FindClass("org/rocksdb/WBWIRocksIterator$WriteEntry");
      assert(jclazz != nullptr);
      return jclazz;
    }

    static void setWriteType(JNIEnv* env, jobject jwrite_entry,
        WriteType write_type) {
      jobject jwrite_type;
      switch (write_type) {
        case kPutRecord:
          jwrite_type = WriteTypeJni::PUT(env);
          break;

        case kMergeRecord:
          jwrite_type = WriteTypeJni::MERGE(env);
          break;

        case kDeleteRecord:
          jwrite_type = WriteTypeJni::DELETE(env);
          break;

        case kLogDataRecord:
          jwrite_type = WriteTypeJni::LOG(env);
          break;

        default:
          jwrite_type = nullptr;
      }
      assert(jwrite_type != nullptr);
      env->SetObjectField(jwrite_entry, getWriteTypeField(env), jwrite_type);
    }

    static void setKey(JNIEnv* env, jobject jwrite_entry,
        const rocksdb::Slice* slice) {
      jobject jkey = env->GetObjectField(jwrite_entry, getKeyField(env));
      AbstractSliceJni::setHandle(env, jkey, slice);
    }

    static void setValue(JNIEnv* env, jobject jwrite_entry,
        const rocksdb::Slice* slice) {
      jobject jvalue = env->GetObjectField(jwrite_entry, getValueField(env));
      AbstractSliceJni::setHandle(env, jvalue, slice);
    }

 private:
    static jfieldID getWriteTypeField(JNIEnv* env) {
      static jfieldID fid = env->GetFieldID(
          getJClass(env), "type", "Lorg/rocksdb/WBWIRocksIterator$WriteType;");
        assert(fid != nullptr);
        return fid;
    }

    static jfieldID getKeyField(JNIEnv* env) {
      static jfieldID fid = env->GetFieldID(
          getJClass(env), "key", "Lorg/rocksdb/DirectSlice;");
      assert(fid != nullptr);
      return fid;
    }

    static jfieldID getValueField(JNIEnv* env) {
      static jfieldID fid = env->GetFieldID(
          getJClass(env), "value", "Lorg/rocksdb/DirectSlice;");
      assert(fid != nullptr);
      return fid;
    }
};

class InfoLogLevelJni {
 public:
    // Get the DEBUG_LEVEL enum field of org.rocksdb.InfoLogLevel
    static jobject DEBUG_LEVEL(JNIEnv* env) {
      return getEnum(env, "DEBUG_LEVEL");
    }

    // Get the INFO_LEVEL enum field of org.rocksdb.InfoLogLevel
    static jobject INFO_LEVEL(JNIEnv* env) {
      return getEnum(env, "INFO_LEVEL");
    }

    // Get the WARN_LEVEL enum field of org.rocksdb.InfoLogLevel
    static jobject WARN_LEVEL(JNIEnv* env) {
      return getEnum(env, "WARN_LEVEL");
    }

    // Get the ERROR_LEVEL enum field of org.rocksdb.InfoLogLevel
    static jobject ERROR_LEVEL(JNIEnv* env) {
      return getEnum(env, "ERROR_LEVEL");
    }

    // Get the FATAL_LEVEL enum field of org.rocksdb.InfoLogLevel
    static jobject FATAL_LEVEL(JNIEnv* env) {
      return getEnum(env, "FATAL_LEVEL");
    }

 private:
    // Get the java class id of org.rocksdb.WBWIRocksIterator.WriteType.
    static jclass getJClass(JNIEnv* env) {
      jclass jclazz = env->FindClass("org/rocksdb/InfoLogLevel");
      assert(jclazz != nullptr);
      return jclazz;
    }

    // Get an enum field of org.rocksdb.WBWIRocksIterator.WriteType
    static jobject getEnum(JNIEnv* env, const char name[]) {
      jclass jclazz = getJClass(env);
      jfieldID jfid =
          env->GetStaticFieldID(jclazz, name,
          "Lorg/rocksdb/InfoLogLevel;");
      assert(jfid != nullptr);
      return env->GetStaticObjectField(jclazz, jfid);
    }
};

// The portal class for org.rocksdb.Logger
class LoggerJni : public RocksDBNativeClass<
    std::shared_ptr<rocksdb::LoggerJniCallback>*, LoggerJni> {
 public:
  static jclass getJClass(JNIEnv* env) {
    return RocksDBNativeClass::getJClass(env,
        "org/rocksdb/Logger");
  }

  // Get the java method `name` of org.rocksdb.Logger.
  static jmethodID getLogMethodId(JNIEnv* env) {
    static jmethodID mid = env->GetMethodID(
        getJClass(env), "log",
        "(Lorg/rocksdb/InfoLogLevel;Ljava/lang/String;)V");
    assert(mid != nullptr);
    return mid;
  }
};

class JniUtil {
 public:
    /*
     * Copies a jstring to a std::string
     * and releases the original jstring
     */
    static std::string copyString(JNIEnv* env, jstring js) {
      const char *utf = env->GetStringUTFChars(js, NULL);
      std::string name(utf);
      env->ReleaseStringUTFChars(js, utf);
      return name;
    }

    /*
     * Helper for operations on a key and value
     * for example WriteBatch->Put
     *
     * TODO(AR) could be extended to cover returning rocksdb::Status
     * from `op` and used for RocksDB->Put etc.
     */
    static void kv_op(
        std::function<void(rocksdb::Slice, rocksdb::Slice)> op,
        JNIEnv* env, jobject jobj,
        jbyteArray jkey, jint jkey_len,
        jbyteArray jentry_value, jint jentry_value_len) {
      jbyte* key = env->GetByteArrayElements(jkey, nullptr);
      jbyte* value = env->GetByteArrayElements(jentry_value, nullptr);
      rocksdb::Slice key_slice(reinterpret_cast<char*>(key), jkey_len);
      rocksdb::Slice value_slice(reinterpret_cast<char*>(value),
          jentry_value_len);

      op(key_slice, value_slice);

      env->ReleaseByteArrayElements(jkey, key, JNI_ABORT);
      env->ReleaseByteArrayElements(jentry_value, value, JNI_ABORT);
    }

    /*
     * Helper for operations on a key
     * for example WriteBatch->Delete
     *
     * TODO(AR) could be extended to cover returning rocksdb::Status
     * from `op` and used for RocksDB->Delete etc.
     */
    static void k_op(
        std::function<void(rocksdb::Slice)> op,
        JNIEnv* env, jobject jobj,
        jbyteArray jkey, jint jkey_len) {
      jbyte* key = env->GetByteArrayElements(jkey, nullptr);
      rocksdb::Slice key_slice(reinterpret_cast<char*>(key), jkey_len);

      op(key_slice);

      env->ReleaseByteArrayElements(jkey, key, JNI_ABORT);
    }
};

}  // namespace rocksdb
#endif  // JAVA_ROCKSJNI_PORTAL_H_
