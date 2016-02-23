//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include "rocksdb/utilities/spatial_db.h"

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <algorithm>
#include <condition_variable>
#include <inttypes.h>
#include <string>
#include <vector>
#include <mutex>
#include <thread>
#include <set>
#include <unordered_set>

#include "rocksdb/cache.h"
#include "rocksdb/options.h"
#include "rocksdb/memtablerep.h"
#include "rocksdb/slice_transform.h"
#include "rocksdb/statistics.h"
#include "rocksdb/table.h"
#include "rocksdb/db.h"
#include "rocksdb/utilities/stackable_db.h"
#include "util/coding.h"
#include "utilities/spatialdb/utils.h"

namespace rocksdb {
namespace spatial {

// Column families are used to store element's data and spatial indexes. We use
// [default] column family to store the element data. This is the format of
// [default] column family:
// * id (fixed 64 big endian) -> blob (length prefixed slice) feature_set
// (serialized)
// We have one additional column family for each spatial index. The name of the
// column family is [spatial$<spatial_index_name>]. The format is:
// * quad_key (fixed 64 bit big endian) id (fixed 64 bit big endian) -> ""
// We store information about indexes in [metadata] column family. Format is:
// * spatial$<spatial_index_name> -> bbox (4 double encodings) tile_bits
// (varint32)

namespace {
const std::string kMetadataColumnFamilyName("metadata");
inline std::string GetSpatialIndexColumnFamilyName(
    const std::string& spatial_index_name) {
  return "spatial$" + spatial_index_name;
}
inline bool GetSpatialIndexName(const std::string& column_family_name,
                                Slice* dst) {
  *dst = Slice(column_family_name);
  if (dst->starts_with("spatial$")) {
    dst->remove_prefix(8);  // strlen("spatial$")
    return true;
  }
  return false;
}

}  // namespace

void Variant::Init(const Variant& v, Data& d) {
  switch (v.type_) {
    case kNull:
      break;
    case kBool:
      d.b = v.data_.b;
      break;
    case kInt:
      d.i = v.data_.i;
      break;
    case kDouble:
      d.d = v.data_.d;
      break;
    case kString:
      new (d.s) std::string(*GetStringPtr(v.data_));
      break;
    default:
      assert(false);
  }
}

Variant& Variant::operator=(const Variant& v) {
  // Construct first a temp so exception from a string ctor
  // does not change this object
  Data tmp;
  Init(v, tmp);

  Type thisType = type_;
  // Boils down to copying bits so safe
  std::swap(tmp, data_);
  type_ = v.type_;

  Destroy(thisType, tmp);

  return *this;
}

Variant& Variant::operator=(Variant&& rhs) {
  Destroy(type_, data_);
  if (rhs.type_ == kString) {
    new (data_.s) std::string(std::move(*GetStringPtr(rhs.data_)));
  } else {
    data_ = rhs.data_;
  }
  type_ = rhs.type_;
  rhs.type_ = kNull;
  return *this;
}

bool Variant::operator==(const Variant& rhs) const {
  if (type_ != rhs.type_) {
    return false;
  }

  switch (type_) {
    case kNull:
      return true;
    case kBool:
      return data_.b == rhs.data_.b;
    case kInt:
      return data_.i == rhs.data_.i;
    case kDouble:
      return data_.d == rhs.data_.d;
    case kString:
      return *GetStringPtr(data_) == *GetStringPtr(rhs.data_);
    default:
      assert(false);
  }
  // it will never reach here, but otherwise the compiler complains
  return false;
}

FeatureSet* FeatureSet::Set(const std::string& key, const Variant& value) {
  map_.insert({key, value});
  return this;
}

bool FeatureSet::Contains(const std::string& key) const {
  return map_.find(key) != map_.end();
}

const Variant& FeatureSet::Get(const std::string& key) const {
  auto itr = map_.find(key);
  assert(itr != map_.end());
  return itr->second;
}

FeatureSet::iterator FeatureSet::Find(const std::string& key) const {
  return iterator(map_.find(key));
}

void FeatureSet::Clear() { map_.clear(); }

void FeatureSet::Serialize(std::string* output) const {
  for (const auto& iter : map_) {
    PutLengthPrefixedSlice(output, iter.first);
    output->push_back(static_cast<char>(iter.second.type()));
    switch (iter.second.type()) {
      case Variant::kNull:
        break;
      case Variant::kBool:
        output->push_back(static_cast<char>(iter.second.get_bool()));
        break;
      case Variant::kInt:
        PutVarint64(output, iter.second.get_int());
        break;
      case Variant::kDouble: {
        PutDouble(output, iter.second.get_double());
        break;
      }
      case Variant::kString:
        PutLengthPrefixedSlice(output, iter.second.get_string());
        break;
      default:
        assert(false);
    }
  }
}

bool FeatureSet::Deserialize(const Slice& input) {
  assert(map_.empty());
  Slice s(input);
  while (s.size()) {
    Slice key;
    if (!GetLengthPrefixedSlice(&s, &key) || s.size() == 0) {
      return false;
    }
    char type = s[0];
    s.remove_prefix(1);
    switch (type) {
      case Variant::kNull: {
        map_.insert({key.ToString(), Variant()});
        break;
      }
      case Variant::kBool: {
        if (s.size() == 0) {
          return false;
        }
        map_.insert({key.ToString(), Variant(static_cast<bool>(s[0]))});
        s.remove_prefix(1);
        break;
      }
      case Variant::kInt: {
        uint64_t v;
        if (!GetVarint64(&s, &v)) {
          return false;
        }
        map_.insert({key.ToString(), Variant(v)});
        break;
      }
      case Variant::kDouble: {
        double d;
        if (!GetDouble(&s, &d)) {
          return false;
        }
        map_.insert({key.ToString(), Variant(d)});
        break;
      }
      case Variant::kString: {
        Slice str;
        if (!GetLengthPrefixedSlice(&s, &str)) {
          return false;
        }
        map_.insert({key.ToString(), str.ToString()});
        break;
      }
      default:
        return false;
    }
  }
  return true;
}

std::string FeatureSet::DebugString() const {
  std::string out = "{";
  bool comma = false;
  for (const auto& iter : map_) {
    if (comma) {
      out.append(", ");
    } else {
      comma = true;
    }
    out.append("\"" + iter.first + "\": ");
    switch (iter.second.type()) {
      case Variant::kNull:
        out.append("null");
        break;
      case Variant::kBool:
        if (iter.second.get_bool()) {
          out.append("true");
        } else {
          out.append("false");
        }
        break;
      case Variant::kInt: {
        char buf[32];
        snprintf(buf, sizeof(buf), "%" PRIu64, iter.second.get_int());
        out.append(buf);
        break;
      }
      case Variant::kDouble: {
        char buf[32];
        snprintf(buf, sizeof(buf), "%lf", iter.second.get_double());
        out.append(buf);
        break;
      }
      case Variant::kString:
        out.append("\"" + iter.second.get_string() + "\"");
        break;
      default:
        assert(false);
    }
  }
  return out + "}";
}

class ValueGetter {
 public:
  ValueGetter() {}
  virtual ~ValueGetter() {}

  virtual bool Get(uint64_t id) = 0;
  virtual const Slice value() const = 0;

  virtual Status status() const = 0;
};

class ValueGetterFromDB : public ValueGetter {
 public:
  ValueGetterFromDB(DB* db, ColumnFamilyHandle* cf) : db_(db), cf_(cf) {}

  virtual bool Get(uint64_t id) override {
    std::string encoded_id;
    PutFixed64BigEndian(&encoded_id, id);
    status_ = db_->Get(ReadOptions(), cf_, encoded_id, &value_);
    if (status_.IsNotFound()) {
      status_ = Status::Corruption("Index inconsistency");
      return false;
    }

    return true;
  }

  virtual const Slice value() const override { return value_; }

  virtual Status status() const override { return status_; }

 private:
  std::string value_;
  DB* db_;
  ColumnFamilyHandle* cf_;
  Status status_;
};

class ValueGetterFromIterator : public ValueGetter {
 public:
  explicit ValueGetterFromIterator(Iterator* iterator) : iterator_(iterator) {}

  virtual bool Get(uint64_t id) override {
    std::string encoded_id;
    PutFixed64BigEndian(&encoded_id, id);
    iterator_->Seek(encoded_id);

    if (!iterator_->Valid() || iterator_->key() != Slice(encoded_id)) {
      status_ = Status::Corruption("Index inconsistency");
      return false;
    }

    return true;
  }

  virtual const Slice value() const override { return iterator_->value(); }

  virtual Status status() const override { return status_; }

 private:
  std::unique_ptr<Iterator> iterator_;
  Status status_;
};

class SpatialIndexCursor : public Cursor {
 public:
  // tile_box is inclusive
  SpatialIndexCursor(Iterator* spatial_iterator, ValueGetter* value_getter,
                     const BoundingBox<uint64_t>& tile_bbox, uint32_t tile_bits)
      : value_getter_(value_getter), valid_(true) {
    // calculate quad keys we'll need to query
    std::vector<uint64_t> quad_keys;
    quad_keys.reserve((tile_bbox.max_x - tile_bbox.min_x + 1) *
                      (tile_bbox.max_y - tile_bbox.min_y + 1));
    for (uint64_t x = tile_bbox.min_x; x <= tile_bbox.max_x; ++x) {
      for (uint64_t y = tile_bbox.min_y; y <= tile_bbox.max_y; ++y) {
        quad_keys.push_back(GetQuadKeyFromTile(x, y, tile_bits));
      }
    }
    std::sort(quad_keys.begin(), quad_keys.end());

    // load primary key ids for all quad keys
    for (auto quad_key : quad_keys) {
      std::string encoded_quad_key;
      PutFixed64BigEndian(&encoded_quad_key, quad_key);
      Slice slice_quad_key(encoded_quad_key);

      // If CheckQuadKey is true, there is no need to reseek, since
      // spatial_iterator is already pointing at the correct quad key. This is
      // an optimization.
      if (!CheckQuadKey(spatial_iterator, slice_quad_key)) {
        spatial_iterator->Seek(slice_quad_key);
      }

      while (CheckQuadKey(spatial_iterator, slice_quad_key)) {
        // extract ID from spatial_iterator
        uint64_t id;
        bool ok = GetFixed64BigEndian(
            Slice(spatial_iterator->key().data() + sizeof(uint64_t),
                  sizeof(uint64_t)),
            &id);
        if (!ok) {
          valid_ = false;
          status_ = Status::Corruption("Spatial index corruption");
          break;
        }
        primary_key_ids_.insert(id);
        spatial_iterator->Next();
      }
    }

    if (!spatial_iterator->status().ok()) {
      status_ = spatial_iterator->status();
      valid_ = false;
    }
    delete spatial_iterator;

    valid_ = valid_ && !primary_key_ids_.empty();

    if (valid_) {
      primary_keys_iterator_ = primary_key_ids_.begin();
      ExtractData();
    }
  }

  virtual bool Valid() const override { return valid_; }

  virtual void Next() override {
    assert(valid_);

    ++primary_keys_iterator_;
    if (primary_keys_iterator_ == primary_key_ids_.end()) {
      valid_ = false;
      return;
    }

    ExtractData();
  }

  virtual const Slice blob() override { return current_blob_; }
  virtual const FeatureSet& feature_set() override {
    return current_feature_set_;
  }

  virtual Status status() const override {
    if (!status_.ok()) {
      return status_;
    }
    return value_getter_->status();
  }

 private:
  // * returns true if spatial iterator is on the current quad key and all is
  // well
  // * returns false if spatial iterator is not on current, or iterator is
  // invalid or corruption
  bool CheckQuadKey(Iterator* spatial_iterator, const Slice& quad_key) {
    if (!spatial_iterator->Valid()) {
      return false;
    }
    if (spatial_iterator->key().size() != 2 * sizeof(uint64_t)) {
      status_ = Status::Corruption("Invalid spatial index key");
      valid_ = false;
      return false;
    }
    Slice spatial_iterator_quad_key(spatial_iterator->key().data(),
                                    sizeof(uint64_t));
    if (spatial_iterator_quad_key != quad_key) {
      // caller needs to reseek
      return false;
    }
    // if we come to here, we have found the quad key
    return true;
  }

  void ExtractData() {
    assert(valid_);
    valid_ = value_getter_->Get(*primary_keys_iterator_);

    if (valid_) {
      Slice data = value_getter_->value();
      current_feature_set_.Clear();
      if (!GetLengthPrefixedSlice(&data, &current_blob_) ||
          !current_feature_set_.Deserialize(data)) {
        status_ = Status::Corruption("Primary key column family corruption");
        valid_ = false;
      }
    }

  }

  unique_ptr<ValueGetter> value_getter_;
  bool valid_;
  Status status_;

  FeatureSet current_feature_set_;
  Slice current_blob_;

  // This is loaded from spatial iterator.
  std::unordered_set<uint64_t> primary_key_ids_;
  std::unordered_set<uint64_t>::iterator primary_keys_iterator_;
};

class ErrorCursor : public Cursor {
 public:
  explicit ErrorCursor(Status s) : s_(s) { assert(!s.ok()); }
  virtual Status status() const override { return s_; }
  virtual bool Valid() const override { return false; }
  virtual void Next() override { assert(false); }

  virtual const Slice blob() override {
    assert(false);
    return Slice();
  }
  virtual const FeatureSet& feature_set() override {
    assert(false);
    // compiler complains otherwise
    return trash_;
  }

 private:
  Status s_;
  FeatureSet trash_;
};

class SpatialDBImpl : public SpatialDB {
 public:
  // * db -- base DB that needs to be forwarded to StackableDB
  // * data_column_family -- column family used to store the data
  // * spatial_indexes -- a list of spatial indexes together with column
  // families that correspond to those spatial indexes
  // * next_id -- next ID in auto-incrementing ID. This is usually
  // `max_id_currenty_in_db + 1`
  SpatialDBImpl(
      DB* db, ColumnFamilyHandle* data_column_family,
      const std::vector<std::pair<SpatialIndexOptions, ColumnFamilyHandle*>>&
          spatial_indexes,
      uint64_t next_id, bool read_only)
      : SpatialDB(db),
        data_column_family_(data_column_family),
        next_id_(next_id),
        read_only_(read_only) {
    for (const auto& index : spatial_indexes) {
      name_to_index_.insert(
          {index.first.name, IndexColumnFamily(index.first, index.second)});
    }
  }

  ~SpatialDBImpl() {
    for (auto& iter : name_to_index_) {
      delete iter.second.column_family;
    }
    delete data_column_family_;
  }

  virtual Status Insert(
      const WriteOptions& write_options, const BoundingBox<double>& bbox,
      const Slice& blob, const FeatureSet& feature_set,
      const std::vector<std::string>& spatial_indexes) override {
    WriteBatch batch;

    if (spatial_indexes.size() == 0) {
      return Status::InvalidArgument("Spatial indexes can't be empty");
    }

    const size_t kWriteOutEveryBytes = 1024 * 1024;  // 1MB
    uint64_t id = next_id_.fetch_add(1);

    for (const auto& si : spatial_indexes) {
      auto itr = name_to_index_.find(si);
      if (itr == name_to_index_.end()) {
        return Status::InvalidArgument("Can't find index " + si);
      }
      const auto& spatial_index = itr->second.index;
      if (!spatial_index.bbox.Intersects(bbox)) {
        continue;
      }
      BoundingBox<uint64_t> tile_bbox = GetTileBoundingBox(spatial_index, bbox);

      for (uint64_t x = tile_bbox.min_x; x <= tile_bbox.max_x; ++x) {
        for (uint64_t y = tile_bbox.min_y; y <= tile_bbox.max_y; ++y) {
          // see above for format
          std::string key;
          PutFixed64BigEndian(
              &key, GetQuadKeyFromTile(x, y, spatial_index.tile_bits));
          PutFixed64BigEndian(&key, id);
          batch.Put(itr->second.column_family, key, Slice());
          if (batch.GetDataSize() >= kWriteOutEveryBytes) {
            Status s = Write(write_options, &batch);
            batch.Clear();
            if (!s.ok()) {
              return s;
            }
          }
        }
      }
    }

    // see above for format
    std::string data_key;
    PutFixed64BigEndian(&data_key, id);
    std::string data_value;
    PutLengthPrefixedSlice(&data_value, blob);
    feature_set.Serialize(&data_value);
    batch.Put(data_column_family_, data_key, data_value);

    return Write(write_options, &batch);
  }

  virtual Status Compact(int num_threads) override {
    std::vector<ColumnFamilyHandle*> column_families;
    column_families.push_back(data_column_family_);

    for (auto& iter : name_to_index_) {
      column_families.push_back(iter.second.column_family);
    }

    std::mutex state_mutex;
    std::condition_variable cv;
    Status s;
    int threads_running = 0;

    std::vector<std::thread> threads;

    for (auto cfh : column_families) {
      threads.emplace_back([&, cfh] {
          {
            std::unique_lock<std::mutex> lk(state_mutex);
            cv.wait(lk, [&] { return threads_running < num_threads; });
            threads_running++;
          }

          Status t = Flush(FlushOptions(), cfh);
          if (t.ok()) {
            t = CompactRange(CompactRangeOptions(), cfh, nullptr, nullptr);
          }

          {
            std::unique_lock<std::mutex> lk(state_mutex);
            threads_running--;
            if (s.ok() && !t.ok()) {
              s = t;
            }
            cv.notify_one();
          }
      });
    }

    for (auto& t : threads) {
      t.join();
    }

    return s;
  }

  virtual Cursor* Query(const ReadOptions& read_options,
                        const BoundingBox<double>& bbox,
                        const std::string& spatial_index) override {
    auto itr = name_to_index_.find(spatial_index);
    if (itr == name_to_index_.end()) {
      return new ErrorCursor(Status::InvalidArgument(
          "Spatial index " + spatial_index + " not found"));
    }
    const auto& si = itr->second.index;
    Iterator* spatial_iterator;
    ValueGetter* value_getter;

    if (read_only_) {
      spatial_iterator = NewIterator(read_options, itr->second.column_family);
      value_getter = new ValueGetterFromDB(this, data_column_family_);
    } else {
      std::vector<Iterator*> iterators;
      Status s = NewIterators(read_options,
                              {data_column_family_, itr->second.column_family},
                              &iterators);
      if (!s.ok()) {
        return new ErrorCursor(s);
      }

      spatial_iterator = iterators[1];
      value_getter = new ValueGetterFromIterator(iterators[0]);
    }
    return new SpatialIndexCursor(spatial_iterator, value_getter,
                                  GetTileBoundingBox(si, bbox), si.tile_bits);
  }

 private:
  ColumnFamilyHandle* data_column_family_;
  struct IndexColumnFamily {
    SpatialIndexOptions index;
    ColumnFamilyHandle* column_family;
    IndexColumnFamily(const SpatialIndexOptions& _index,
                      ColumnFamilyHandle* _cf)
        : index(_index), column_family(_cf) {}
  };
  // constant after construction!
  std::unordered_map<std::string, IndexColumnFamily> name_to_index_;

  std::atomic<uint64_t> next_id_;
  bool read_only_;
};

namespace {
DBOptions GetDBOptionsFromSpatialDBOptions(const SpatialDBOptions& options) {
  DBOptions db_options;
  db_options.max_open_files = 50000;
  db_options.max_background_compactions = 3 * options.num_threads / 4;
  db_options.max_background_flushes =
      options.num_threads - db_options.max_background_compactions;
  db_options.env->SetBackgroundThreads(db_options.max_background_compactions,
                                       Env::LOW);
  db_options.env->SetBackgroundThreads(db_options.max_background_flushes,
                                       Env::HIGH);
  db_options.statistics = CreateDBStatistics();
  if (options.bulk_load) {
    db_options.stats_dump_period_sec = 600;
    db_options.disableDataSync = true;
  } else {
    db_options.stats_dump_period_sec = 1800;  // 30min
  }
  return db_options;
}

ColumnFamilyOptions GetColumnFamilyOptions(const SpatialDBOptions& options,
                                           std::shared_ptr<Cache> block_cache) {
  ColumnFamilyOptions column_family_options;
  column_family_options.write_buffer_size = 128 * 1024 * 1024;  // 128MB
  column_family_options.max_write_buffer_number = 4;
  column_family_options.max_bytes_for_level_base = 256 * 1024 * 1024;  // 256MB
  column_family_options.target_file_size_base = 64 * 1024 * 1024;      // 64MB
  column_family_options.level0_file_num_compaction_trigger = 2;
  column_family_options.level0_slowdown_writes_trigger = 16;
  column_family_options.level0_slowdown_writes_trigger = 32;
  // only compress levels >= 2
  column_family_options.compression_per_level.resize(
      column_family_options.num_levels);
  for (int i = 0; i < column_family_options.num_levels; ++i) {
    if (i < 2) {
      column_family_options.compression_per_level[i] = kNoCompression;
    } else {
      column_family_options.compression_per_level[i] = kLZ4Compression;
    }
  }
  BlockBasedTableOptions table_options;
  table_options.block_cache = block_cache;
  column_family_options.table_factory.reset(
      NewBlockBasedTableFactory(table_options));
  return column_family_options;
}

ColumnFamilyOptions OptimizeOptionsForDataColumnFamily(
    ColumnFamilyOptions options, std::shared_ptr<Cache> block_cache) {
  options.prefix_extractor.reset(NewNoopTransform());
  BlockBasedTableOptions block_based_options;
  block_based_options.index_type = BlockBasedTableOptions::kHashSearch;
  block_based_options.block_cache = block_cache;
  options.table_factory.reset(NewBlockBasedTableFactory(block_based_options));
  return options;
}

}  // namespace

class MetadataStorage {
 public:
  MetadataStorage(DB* db, ColumnFamilyHandle* cf) : db_(db), cf_(cf) {}
  ~MetadataStorage() {}

  // format: <min_x double> <min_y double> <max_x double> <max_y double>
  // <tile_bits varint32>
  Status AddIndex(const SpatialIndexOptions& index) {
    std::string encoded_index;
    PutDouble(&encoded_index, index.bbox.min_x);
    PutDouble(&encoded_index, index.bbox.min_y);
    PutDouble(&encoded_index, index.bbox.max_x);
    PutDouble(&encoded_index, index.bbox.max_y);
    PutVarint32(&encoded_index, index.tile_bits);
    return db_->Put(WriteOptions(), cf_,
                    GetSpatialIndexColumnFamilyName(index.name), encoded_index);
  }

  Status GetIndex(const std::string& name, SpatialIndexOptions* dst) {
    std::string value;
    Status s = db_->Get(ReadOptions(), cf_,
                        GetSpatialIndexColumnFamilyName(name), &value);
    if (!s.ok()) {
      return s;
    }
    dst->name = name;
    Slice encoded_index(value);
    bool ok = GetDouble(&encoded_index, &(dst->bbox.min_x));
    ok = ok && GetDouble(&encoded_index, &(dst->bbox.min_y));
    ok = ok && GetDouble(&encoded_index, &(dst->bbox.max_x));
    ok = ok && GetDouble(&encoded_index, &(dst->bbox.max_y));
    ok = ok && GetVarint32(&encoded_index, &(dst->tile_bits));
    return ok ? Status::OK() : Status::Corruption("Index encoding corrupted");
  }

 private:
  DB* db_;
  ColumnFamilyHandle* cf_;
};

Status SpatialDB::Create(
    const SpatialDBOptions& options, const std::string& name,
    const std::vector<SpatialIndexOptions>& spatial_indexes) {
  DBOptions db_options = GetDBOptionsFromSpatialDBOptions(options);
  db_options.create_if_missing = true;
  db_options.create_missing_column_families = true;
  db_options.error_if_exists = true;

  auto block_cache = NewLRUCache(options.cache_size);
  ColumnFamilyOptions column_family_options =
      GetColumnFamilyOptions(options, block_cache);

  std::vector<ColumnFamilyDescriptor> column_families;
  column_families.push_back(ColumnFamilyDescriptor(
      kDefaultColumnFamilyName,
      OptimizeOptionsForDataColumnFamily(column_family_options, block_cache)));
  column_families.push_back(
      ColumnFamilyDescriptor(kMetadataColumnFamilyName, column_family_options));

  for (const auto& index : spatial_indexes) {
    column_families.emplace_back(GetSpatialIndexColumnFamilyName(index.name),
                                 column_family_options);
  }

  std::vector<ColumnFamilyHandle*> handles;
  DB* base_db;
  Status s = DB::Open(db_options, name, column_families, &handles, &base_db);
  if (!s.ok()) {
    return s;
  }
  MetadataStorage metadata(base_db, handles[1]);
  for (const auto& index : spatial_indexes) {
    s = metadata.AddIndex(index);
    if (!s.ok()) {
      break;
    }
  }

  for (auto h : handles) {
    delete h;
  }
  delete base_db;

  return s;
}

Status SpatialDB::Open(const SpatialDBOptions& options, const std::string& name,
                       SpatialDB** db, bool read_only) {
  DBOptions db_options = GetDBOptionsFromSpatialDBOptions(options);
  auto block_cache = NewLRUCache(options.cache_size);
  ColumnFamilyOptions column_family_options =
      GetColumnFamilyOptions(options, block_cache);

  Status s;
  std::vector<std::string> existing_column_families;
  std::vector<std::string> spatial_indexes;
  s = DB::ListColumnFamilies(db_options, name, &existing_column_families);
  if (!s.ok()) {
    return s;
  }
  for (const auto& cf_name : existing_column_families) {
    Slice spatial_index;
    if (GetSpatialIndexName(cf_name, &spatial_index)) {
      spatial_indexes.emplace_back(spatial_index.data(), spatial_index.size());
    }
  }

  std::vector<ColumnFamilyDescriptor> column_families;
  column_families.push_back(ColumnFamilyDescriptor(
      kDefaultColumnFamilyName,
      OptimizeOptionsForDataColumnFamily(column_family_options, block_cache)));
  column_families.push_back(
      ColumnFamilyDescriptor(kMetadataColumnFamilyName, column_family_options));

  for (const auto& index : spatial_indexes) {
    column_families.emplace_back(GetSpatialIndexColumnFamilyName(index),
                                 column_family_options);
  }
  std::vector<ColumnFamilyHandle*> handles;
  DB* base_db;
  if (read_only) {
    s = DB::OpenForReadOnly(db_options, name, column_families, &handles,
                            &base_db);
  } else {
    s = DB::Open(db_options, name, column_families, &handles, &base_db);
  }
  if (!s.ok()) {
    return s;
  }

  MetadataStorage metadata(base_db, handles[1]);

  std::vector<std::pair<SpatialIndexOptions, ColumnFamilyHandle*>> index_cf;
  assert(handles.size() == spatial_indexes.size() + 2);
  for (size_t i = 0; i < spatial_indexes.size(); ++i) {
    SpatialIndexOptions index_options;
    s = metadata.GetIndex(spatial_indexes[i], &index_options);
    if (!s.ok()) {
      break;
    }
    index_cf.emplace_back(index_options, handles[i + 2]);
  }
  uint64_t next_id = 1;
  if (s.ok()) {
    // find next_id
    Iterator* iter = base_db->NewIterator(ReadOptions(), handles[0]);
    iter->SeekToLast();
    if (iter->Valid()) {
      uint64_t last_id = 0;
      if (!GetFixed64BigEndian(iter->key(), &last_id)) {
        s = Status::Corruption("Invalid key in data column family");
      } else {
        next_id = last_id + 1;
      }
    }
    delete iter;
  }
  if (!s.ok()) {
    for (auto h : handles) {
      delete h;
    }
    delete base_db;
    return s;
  }

  // I don't need metadata column family any more, so delete it
  delete handles[1];
  *db = new SpatialDBImpl(base_db, handles[0], index_cf, next_id, read_only);
  return Status::OK();
}

}  // namespace spatial
}  // namespace rocksdb
#endif  // ROCKSDB_LITE
