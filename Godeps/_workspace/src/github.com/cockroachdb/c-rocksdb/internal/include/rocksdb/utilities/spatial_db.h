//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#pragma once
#ifndef ROCKSDB_LITE

#include <string>
#include <vector>

#include "rocksdb/db.h"
#include "rocksdb/slice.h"
#include "rocksdb/utilities/stackable_db.h"

namespace rocksdb {
namespace spatial {

// NOTE: SpatialDB is experimental and we might change its API without warning.
// Please talk to us before developing against SpatialDB API.
//
// SpatialDB is a support for spatial indexes built on top of RocksDB.
// When creating a new SpatialDB, clients specifies a list of spatial indexes to
// build on their data. Each spatial index is defined by the area and
// granularity. If you're storing map data, different spatial index
// granularities can be used for different zoom levels.
//
// Each element inserted into SpatialDB has:
// * a bounding box, which determines how will the element be indexed
// * string blob, which will usually be WKB representation of the polygon
// (http://en.wikipedia.org/wiki/Well-known_text)
// * feature set, which is a map of key-value pairs, where value can be null,
// int, double, bool, string
// * a list of indexes to insert the element in
//
// Each query is executed on a single spatial index. Query guarantees that it
// will return all elements intersecting the specified bounding box, but it
// might also return some extra non-intersecting elements.

// Variant is a class that can be many things: null, bool, int, double or string
// It is used to store different value types in FeatureSet (see below)
struct Variant {
  // Don't change the values here, they are persisted on disk
  enum Type {
    kNull = 0x0,
    kBool = 0x1,
    kInt = 0x2,
    kDouble = 0x3,
    kString = 0x4,
  };

  Variant() : type_(kNull) {}
  /* implicit */ Variant(bool b) : type_(kBool) { data_.b = b; }
  /* implicit */ Variant(uint64_t i) : type_(kInt) { data_.i = i; }
  /* implicit */ Variant(double d) : type_(kDouble) { data_.d = d; }
  /* implicit */ Variant(const std::string& s) : type_(kString) {
    new (&data_.s) std::string(s);
  }

  Variant(const Variant& v) : type_(v.type_) { Init(v, data_); }

  Variant& operator=(const Variant& v);

  Variant(Variant&& rhs) : type_(kNull) { *this = std::move(rhs); }

  Variant& operator=(Variant&& v);

  ~Variant() { Destroy(type_, data_); }

  Type type() const { return type_; }
  bool get_bool() const { return data_.b; }
  uint64_t get_int() const { return data_.i; }
  double get_double() const { return data_.d; }
  const std::string& get_string() const { return *GetStringPtr(data_); }

  bool operator==(const Variant& other) const;
  bool operator!=(const Variant& other) const { return !(*this == other); }

 private:
  Type type_;

  union Data {
    bool b;
    uint64_t i;
    double d;
    // Current version of MS compiler not C++11 compliant so can not put
    // std::string
    // however, even then we still need the rest of the maintenance.
    char s[sizeof(std::string)];
  } data_;

  // Avoid type_punned aliasing problem
  static std::string* GetStringPtr(Data& d) {
    void* p = d.s;
    return reinterpret_cast<std::string*>(p);
  }

  static const std::string* GetStringPtr(const Data& d) {
    const void* p = d.s;
    return reinterpret_cast<const std::string*>(p);
  }

  static void Init(const Variant&, Data&);

  static void Destroy(Type t, Data& d) {
    if (t == kString) {
      using std::string;
      GetStringPtr(d)->~string();
    }
  }
};

// FeatureSet is a map of key-value pairs. One feature set is associated with
// each element in SpatialDB. It can be used to add rich data about the element.
class FeatureSet {
 private:
  typedef std::unordered_map<std::string, Variant> map;

 public:
  class iterator {
   public:
    /* implicit */ iterator(const map::const_iterator itr) : itr_(itr) {}
    iterator& operator++() {
      ++itr_;
      return *this;
    }
    bool operator!=(const iterator& other) { return itr_ != other.itr_; }
    bool operator==(const iterator& other) { return itr_ == other.itr_; }
    map::value_type operator*() { return *itr_; }

   private:
    map::const_iterator itr_;
  };
  FeatureSet() = default;

  FeatureSet* Set(const std::string& key, const Variant& value);
  bool Contains(const std::string& key) const;
  // REQUIRES: Contains(key)
  const Variant& Get(const std::string& key) const;
  iterator Find(const std::string& key) const;

  iterator begin() const { return map_.begin(); }
  iterator end() const { return map_.end(); }

  void Clear();
  size_t Size() const { return map_.size(); }

  void Serialize(std::string* output) const;
  // REQUIRED: empty FeatureSet
  bool Deserialize(const Slice& input);

  std::string DebugString() const;

 private:
  map map_;
};

// BoundingBox is a helper structure for defining rectangles representing
// bounding boxes of spatial elements.
template <typename T>
struct BoundingBox {
  T min_x, min_y, max_x, max_y;
  BoundingBox() = default;
  BoundingBox(T _min_x, T _min_y, T _max_x, T _max_y)
      : min_x(_min_x), min_y(_min_y), max_x(_max_x), max_y(_max_y) {}

  bool Intersects(const BoundingBox<T>& a) const {
    return !(min_x > a.max_x || min_y > a.max_y || a.min_x > max_x ||
             a.min_y > max_y);
  }
};

struct SpatialDBOptions {
  uint64_t cache_size = 1 * 1024 * 1024 * 1024LL;  // 1GB
  int num_threads = 16;
  bool bulk_load = true;
};

// Cursor is used to return data from the query to the client. To get all the
// data from the query, just call Next() while Valid() is true
class Cursor {
 public:
  Cursor() = default;
  virtual ~Cursor() {}

  virtual bool Valid() const = 0;
  // REQUIRES: Valid()
  virtual void Next() = 0;

  // Lifetime of the underlying storage until the next call to Next()
  // REQUIRES: Valid()
  virtual const Slice blob() = 0;
  // Lifetime of the underlying storage until the next call to Next()
  // REQUIRES: Valid()
  virtual const FeatureSet& feature_set() = 0;

  virtual Status status() const = 0;

 private:
  // No copying allowed
  Cursor(const Cursor&);
  void operator=(const Cursor&);
};

// SpatialIndexOptions defines a spatial index that will be built on the data
struct SpatialIndexOptions {
  // Spatial indexes are referenced by names
  std::string name;
  // An area that is indexed. If the element is not intersecting with spatial
  // index's bbox, it will not be inserted into the index
  BoundingBox<double> bbox;
  // tile_bits control the granularity of the spatial index. Each dimension of
  // the bbox will be split into (1 << tile_bits) tiles, so there will be a
  // total of (1 << tile_bits)^2 tiles. It is recommended to configure a size of
  // each  tile to be approximately the size of the query on that spatial index
  uint32_t tile_bits;
  SpatialIndexOptions() {}
  SpatialIndexOptions(const std::string& _name,
                      const BoundingBox<double>& _bbox, uint32_t _tile_bits)
      : name(_name), bbox(_bbox), tile_bits(_tile_bits) {}
};

class SpatialDB : public StackableDB {
 public:
  // Creates the SpatialDB with specified list of indexes.
  // REQUIRED: db doesn't exist
  static Status Create(const SpatialDBOptions& options, const std::string& name,
                       const std::vector<SpatialIndexOptions>& spatial_indexes);

  // Open the existing SpatialDB.  The resulting db object will be returned
  // through db parameter.
  // REQUIRED: db was created using SpatialDB::Create
  static Status Open(const SpatialDBOptions& options, const std::string& name,
                     SpatialDB** db, bool read_only = false);

  explicit SpatialDB(DB* db) : StackableDB(db) {}

  // Insert the element into the DB. Element will be inserted into specified
  // spatial_indexes, based on specified bbox.
  // REQUIRES: spatial_indexes.size() > 0
  virtual Status Insert(const WriteOptions& write_options,
                        const BoundingBox<double>& bbox, const Slice& blob,
                        const FeatureSet& feature_set,
                        const std::vector<std::string>& spatial_indexes) = 0;

  // Calling Compact() after inserting a bunch of elements should speed up
  // reading. This is especially useful if you use SpatialDBOptions::bulk_load
  // Num threads determines how many threads we'll use for compactions. Setting
  // this to bigger number will use more IO and CPU, but finish faster
  virtual Status Compact(int num_threads = 1) = 0;

  // Query the specified spatial_index. Query will return all elements that
  // intersect bbox, but it may also return some extra elements.
  virtual Cursor* Query(const ReadOptions& read_options,
                        const BoundingBox<double>& bbox,
                        const std::string& spatial_index) = 0;
};

}  // namespace spatial
}  // namespace rocksdb
#endif  // ROCKSDB_LITE
