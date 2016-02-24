//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE
#include "rocksdb/utilities/json_document.h"
#include "third-party/fbson/FbsonWriter.h"

namespace rocksdb {
JSONDocumentBuilder::JSONDocumentBuilder()
: writer_(new fbson::FbsonWriter()) {
}

JSONDocumentBuilder::JSONDocumentBuilder(fbson::FbsonOutStream* out)
: writer_(new fbson::FbsonWriter(*out)) {
}

void JSONDocumentBuilder::Reset() {
  writer_->reset();
}

bool JSONDocumentBuilder::WriteStartArray() {
  return writer_->writeStartArray();
}

bool JSONDocumentBuilder::WriteEndArray() {
  return writer_->writeEndArray();
}

bool JSONDocumentBuilder::WriteStartObject() {
  return writer_->writeStartObject();
}

bool JSONDocumentBuilder::WriteEndObject() {
  return writer_->writeEndObject();
}

bool JSONDocumentBuilder::WriteKeyValue(const std::string& key,
                                        const JSONDocument& value) {
  size_t bytesWritten = writer_->writeKey(key.c_str(), key.size());
  if (bytesWritten == 0) {
    return false;
  }
  return WriteJSONDocument(value);
}

bool JSONDocumentBuilder::WriteJSONDocument(const JSONDocument& value) {
  switch (value.type()) {
    case JSONDocument::kNull:
      return writer_->writeNull() != 0;
    case JSONDocument::kInt64:
      return writer_->writeInt64(value.GetInt64());
    case JSONDocument::kDouble:
      return writer_->writeDouble(value.GetDouble());
    case JSONDocument::kBool:
      return writer_->writeBool(value.GetBool());
    case JSONDocument::kString:
    {
      bool res = writer_->writeStartString();
      if (!res) {
        return false;
      }
      const std::string& str = value.GetString();
      res = writer_->writeString(str.c_str(),
                  static_cast<uint32_t>(str.size()));
      if (!res) {
        return false;
      }
      return writer_->writeEndString();
    }
    case JSONDocument::kArray:
    {
      bool res = WriteStartArray();
      if (!res) {
        return false;
      }
      for (size_t i = 0; i < value.Count(); ++i) {
        res = WriteJSONDocument(value[i]);
        if (!res) {
          return false;
        }
      }
      return WriteEndArray();
    }
    case JSONDocument::kObject:
    {
      bool res = WriteStartObject();
      if (!res) {
        return false;
      }
      for (auto keyValue : value.Items()) {
        WriteKeyValue(keyValue.first, keyValue.second);
      }
      return WriteEndObject();
    }
    default:
      assert(false);
  }
  return false;
}

JSONDocument JSONDocumentBuilder::GetJSONDocument() {
  fbson::FbsonValue* value =
      fbson::FbsonDocument::createValue(writer_->getOutput()->getBuffer(),
                       static_cast<uint32_t>(writer_->getOutput()->getSize()));
  return JSONDocument(value, true);
}

JSONDocumentBuilder::~JSONDocumentBuilder() {
}

}  // namespace rocksdb

#endif  // ROCKSDB_LITE
