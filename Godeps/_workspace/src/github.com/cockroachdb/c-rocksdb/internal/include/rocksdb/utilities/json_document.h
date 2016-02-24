//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
#pragma once
#ifndef ROCKSDB_LITE

#include <deque>
#include <map>
#include <memory>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "rocksdb/slice.h"

// We use JSONDocument for DocumentDB API
// Implementation inspired by folly::dynamic, rapidjson and fbson

namespace fbson {
  class FbsonValue;
  class ObjectVal;
  template <typename T>
  class FbsonWriterT;
  class FbsonOutStream;
  typedef FbsonWriterT<FbsonOutStream> FbsonWriter;
}  // namespace fbson

namespace rocksdb {

// NOTE: none of this is thread-safe
class JSONDocument {
 public:
  // return nullptr on parse failure
  static JSONDocument* ParseJSON(const char* json);

  enum Type {
    kNull,
    kArray,
    kBool,
    kDouble,
    kInt64,
    kObject,
    kString,
  };

  /* implicit */ JSONDocument();  // null
  /* implicit */ JSONDocument(bool b);
  /* implicit */ JSONDocument(double d);
  /* implicit */ JSONDocument(int8_t i);
  /* implicit */ JSONDocument(int16_t i);
  /* implicit */ JSONDocument(int32_t i);
  /* implicit */ JSONDocument(int64_t i);
  /* implicit */ JSONDocument(const std::string& s);
  /* implicit */ JSONDocument(const char* s);
  // constructs JSONDocument of specific type with default value
  explicit JSONDocument(Type _type);

  JSONDocument(const JSONDocument& json_document);

  JSONDocument(JSONDocument&& json_document);

  Type type() const;

  // REQUIRES: IsObject()
  bool Contains(const std::string& key) const;
  // REQUIRES: IsObject()
  // Returns non-owner object
  JSONDocument operator[](const std::string& key) const;

  // REQUIRES: IsArray() == true || IsObject() == true
  size_t Count() const;

  // REQUIRES: IsArray()
  // Returns non-owner object
  JSONDocument operator[](size_t i) const;

  JSONDocument& operator=(JSONDocument jsonDocument);

  bool IsNull() const;
  bool IsArray() const;
  bool IsBool() const;
  bool IsDouble() const;
  bool IsInt64() const;
  bool IsObject() const;
  bool IsString() const;

  // REQUIRES: IsBool() == true
  bool GetBool() const;
  // REQUIRES: IsDouble() == true
  double GetDouble() const;
  // REQUIRES: IsInt64() == true
  int64_t GetInt64() const;
  // REQUIRES: IsString() == true
  std::string GetString() const;

  bool operator==(const JSONDocument& rhs) const;

  bool operator!=(const JSONDocument& rhs) const;

  JSONDocument Copy() const;

  bool IsOwner() const;

  std::string DebugString() const;

 private:
  class ItemsIteratorGenerator;

 public:
  // REQUIRES: IsObject()
  ItemsIteratorGenerator Items() const;

  // appends serialized object to dst
  void Serialize(std::string* dst) const;
  // returns nullptr if Slice doesn't represent valid serialized JSONDocument
  static JSONDocument* Deserialize(const Slice& src);

 private:
  friend class JSONDocumentBuilder;

  JSONDocument(fbson::FbsonValue* val, bool makeCopy);

  void InitFromValue(const fbson::FbsonValue* val);

  // iteration on objects
  class const_item_iterator {
   private:
    class Impl;
   public:
    typedef std::pair<std::string, JSONDocument> value_type;
    explicit const_item_iterator(Impl* impl);
    const_item_iterator(const_item_iterator&&);
    const_item_iterator& operator++();
    bool operator!=(const const_item_iterator& other);
    value_type operator*();
    ~const_item_iterator();
   private:
    friend class ItemsIteratorGenerator;
    std::unique_ptr<Impl> it_;
  };

  class ItemsIteratorGenerator {
   public:
    explicit ItemsIteratorGenerator(const fbson::ObjectVal& object);
    const_item_iterator begin() const;

    const_item_iterator end() const;

   private:
    const fbson::ObjectVal& object_;
  };

  std::unique_ptr<char[]> data_;
  mutable fbson::FbsonValue* value_;

  // Our serialization format's first byte specifies the encoding version. That
  // way, we can easily change our format while providing backwards
  // compatibility. This constant specifies the current version of the
  // serialization format
  static const char kSerializationFormatVersion;
};

class JSONDocumentBuilder {
 public:
  JSONDocumentBuilder();

  explicit JSONDocumentBuilder(fbson::FbsonOutStream* out);

  void Reset();

  bool WriteStartArray();

  bool WriteEndArray();

  bool WriteStartObject();

  bool WriteEndObject();

  bool WriteKeyValue(const std::string& key, const JSONDocument& value);

  bool WriteJSONDocument(const JSONDocument& value);

  JSONDocument GetJSONDocument();

  ~JSONDocumentBuilder();

 private:
  std::unique_ptr<fbson::FbsonWriter> writer_;
};

}  // namespace rocksdb

#endif  // ROCKSDB_LITE
