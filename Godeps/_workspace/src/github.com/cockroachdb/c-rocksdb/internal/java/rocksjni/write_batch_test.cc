// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// This file implements the "bridge" between Java and C++ and enables
// calling c++ rocksdb::WriteBatch methods testing from Java side.
#include <memory>

#include "db/memtable.h"
#include "db/write_batch_internal.h"
#include "db/writebuffer.h"
#include "include/org_rocksdb_WriteBatch.h"
#include "include/org_rocksdb_WriteBatch_Handler.h"
#include "include/org_rocksdb_WriteBatchTest.h"
#include "include/org_rocksdb_WriteBatchTestInternalHelper.h"
#include "rocksdb/db.h"
#include "rocksdb/env.h"
#include "rocksdb/immutable_options.h"
#include "rocksdb/memtablerep.h"
#include "rocksdb/status.h"
#include "rocksdb/write_batch.h"
#include "rocksjni/portal.h"
#include "util/logging.h"
#include "util/scoped_arena_iterator.h"
#include "util/testharness.h"

/*
 * Class:     org_rocksdb_WriteBatchTest
 * Method:    getContents
 * Signature: (Lorg/rocksdb/WriteBatch;)[B
 */
jbyteArray Java_org_rocksdb_WriteBatchTest_getContents(
    JNIEnv* env, jclass jclazz, jobject jobj) {
  rocksdb::WriteBatch* b = rocksdb::WriteBatchJni::getHandle(env, jobj);
  assert(b != nullptr);

  // todo: Currently the following code is directly copied from
  // db/write_bench_test.cc.  It could be implemented in java once
  // all the necessary components can be accessed via jni api.

  rocksdb::InternalKeyComparator cmp(rocksdb::BytewiseComparator());
  auto factory = std::make_shared<rocksdb::SkipListFactory>();
  rocksdb::Options options;
  rocksdb::WriteBuffer wb(options.db_write_buffer_size);
  options.memtable_factory = factory;
  rocksdb::MemTable* mem = new rocksdb::MemTable(
      cmp, rocksdb::ImmutableCFOptions(options),
      rocksdb::MutableCFOptions(options, rocksdb::ImmutableCFOptions(options)),
      &wb, rocksdb::kMaxSequenceNumber);
  mem->Ref();
  std::string state;
  rocksdb::ColumnFamilyMemTablesDefault cf_mems_default(mem);
  rocksdb::Status s =
      rocksdb::WriteBatchInternal::InsertInto(b, &cf_mems_default);
  int count = 0;
  rocksdb::Arena arena;
  rocksdb::ScopedArenaIterator iter(mem->NewIterator(
      rocksdb::ReadOptions(), &arena));
  for (iter->SeekToFirst(); iter->Valid(); iter->Next()) {
    rocksdb::ParsedInternalKey ikey;
    memset(reinterpret_cast<void*>(&ikey), 0, sizeof(ikey));
    assert(rocksdb::ParseInternalKey(iter->key(), &ikey));
    switch (ikey.type) {
      case rocksdb::kTypeValue:
        state.append("Put(");
        state.append(ikey.user_key.ToString());
        state.append(", ");
        state.append(iter->value().ToString());
        state.append(")");
        count++;
        break;
      case rocksdb::kTypeMerge:
        state.append("Merge(");
        state.append(ikey.user_key.ToString());
        state.append(", ");
        state.append(iter->value().ToString());
        state.append(")");
        count++;
        break;
      case rocksdb::kTypeDeletion:
        state.append("Delete(");
        state.append(ikey.user_key.ToString());
        state.append(")");
        count++;
        break;
      default:
        assert(false);
        break;
    }
    state.append("@");
    state.append(rocksdb::NumberToString(ikey.sequence));
  }
  if (!s.ok()) {
    state.append(s.ToString());
  } else if (count != rocksdb::WriteBatchInternal::Count(b)) {
    state.append("CountMismatch()");
  }
  delete mem->Unref();

  jbyteArray jstate = env->NewByteArray(static_cast<jsize>(state.size()));
  env->SetByteArrayRegion(jstate, 0, static_cast<jsize>(state.size()),
                          reinterpret_cast<const jbyte*>(state.c_str()));

  return jstate;
}

/*
 * Class:     org_rocksdb_WriteBatchTestInternalHelper
 * Method:    setSequence
 * Signature: (Lorg/rocksdb/WriteBatch;J)V
 */
void Java_org_rocksdb_WriteBatchTestInternalHelper_setSequence(
    JNIEnv* env, jclass jclazz, jobject jobj, jlong jsn) {
  rocksdb::WriteBatch* wb = rocksdb::WriteBatchJni::getHandle(env, jobj);
  assert(wb != nullptr);

  rocksdb::WriteBatchInternal::SetSequence(
      wb, static_cast<rocksdb::SequenceNumber>(jsn));
}

/*
 * Class:     org_rocksdb_WriteBatchTestInternalHelper
 * Method:    sequence
 * Signature: (Lorg/rocksdb/WriteBatch;)J
 */
jlong Java_org_rocksdb_WriteBatchTestInternalHelper_sequence(
    JNIEnv* env, jclass jclazz, jobject jobj) {
  rocksdb::WriteBatch* wb = rocksdb::WriteBatchJni::getHandle(env, jobj);
  assert(wb != nullptr);

  return static_cast<jlong>(rocksdb::WriteBatchInternal::Sequence(wb));
}

/*
 * Class:     org_rocksdb_WriteBatchTestInternalHelper
 * Method:    append
 * Signature: (Lorg/rocksdb/WriteBatch;Lorg/rocksdb/WriteBatch;)V
 */
void Java_org_rocksdb_WriteBatchTestInternalHelper_append(
    JNIEnv* env, jclass jclazz, jobject jwb1, jobject jwb2) {
  rocksdb::WriteBatch* wb1 = rocksdb::WriteBatchJni::getHandle(env, jwb1);
  assert(wb1 != nullptr);
  rocksdb::WriteBatch* wb2 = rocksdb::WriteBatchJni::getHandle(env, jwb2);
  assert(wb2 != nullptr);

  rocksdb::WriteBatchInternal::Append(wb1, wb2);
}
