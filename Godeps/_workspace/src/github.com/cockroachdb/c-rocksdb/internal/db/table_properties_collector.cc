//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include "db/table_properties_collector.h"

#include "db/dbformat.h"
#include "util/coding.h"
#include "util/string_util.h"

namespace rocksdb {

Status InternalKeyPropertiesCollector::InternalAdd(const Slice& key,
                                                   const Slice& value,
                                                   uint64_t file_size) {
  ParsedInternalKey ikey;
  if (!ParseInternalKey(key, &ikey)) {
    return Status::InvalidArgument("Invalid internal key");
  }

  if (ikey.type == ValueType::kTypeDeletion) {
    ++deleted_keys_;
  }

  return Status::OK();
}

Status InternalKeyPropertiesCollector::Finish(
    UserCollectedProperties* properties) {
  assert(properties);
  assert(properties->find(
        InternalKeyTablePropertiesNames::kDeletedKeys) == properties->end());
  std::string val;

  PutVarint64(&val, deleted_keys_);
  properties->insert({ InternalKeyTablePropertiesNames::kDeletedKeys, val });

  return Status::OK();
}

UserCollectedProperties
InternalKeyPropertiesCollector::GetReadableProperties() const {
  return {
    { "kDeletedKeys", ToString(deleted_keys_) }
  };
}

namespace {
EntryType GetEntryType(ValueType value_type) {
  switch (value_type) {
    case kTypeValue:
      return kEntryPut;
    case kTypeDeletion:
      return kEntryDelete;
    case kTypeMerge:
      return kEntryMerge;
    default:
      return kEntryOther;
  }
}
}  // namespace

Status UserKeyTablePropertiesCollector::InternalAdd(const Slice& key,
                                                    const Slice& value,
                                                    uint64_t file_size) {
  ParsedInternalKey ikey;
  if (!ParseInternalKey(key, &ikey)) {
    return Status::InvalidArgument("Invalid internal key");
  }

  return collector_->AddUserKey(ikey.user_key, value, GetEntryType(ikey.type),
                                ikey.sequence, file_size);
}

Status UserKeyTablePropertiesCollector::Finish(
    UserCollectedProperties* properties) {
  return collector_->Finish(properties);
}

UserCollectedProperties
UserKeyTablePropertiesCollector::GetReadableProperties() const {
  return collector_->GetReadableProperties();
}


const std::string InternalKeyTablePropertiesNames::kDeletedKeys
  = "rocksdb.deleted.keys";

uint64_t GetDeletedKeys(
    const UserCollectedProperties& props) {
  auto pos = props.find(InternalKeyTablePropertiesNames::kDeletedKeys);
  if (pos == props.end()) {
    return 0;
  }
  Slice raw = pos->second;
  uint64_t val = 0;
  return GetVarint64(&raw, &val) ? val : 0;
}

}  // namespace rocksdb
