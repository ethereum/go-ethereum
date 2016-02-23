//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include "rocksdb/utilities/document_db.h"

#include "rocksdb/cache.h"
#include "rocksdb/table.h"
#include "rocksdb/filter_policy.h"
#include "rocksdb/comparator.h"
#include "rocksdb/db.h"
#include "rocksdb/slice.h"
#include "rocksdb/utilities/json_document.h"
#include "util/coding.h"
#include "util/mutexlock.h"
#include "port/port.h"

namespace rocksdb {

// IMPORTANT NOTE: Secondary index column families should be very small and
// generally fit in memory. Assume that accessing secondary index column
// families is much faster than accessing primary index (data heap) column
// family. Accessing a key (i.e. checking for existance) from a column family in
// RocksDB is not much faster than accessing both key and value since they are
// kept together and loaded from storage together.

namespace {
// < 0   <=>  lhs < rhs
// == 0  <=>  lhs == rhs
// > 0   <=>  lhs == rhs
// TODO(icanadi) move this to JSONDocument?
int DocumentCompare(const JSONDocument& lhs, const JSONDocument& rhs) {
  assert(lhs.IsObject() == false && rhs.IsObject() == false &&
         lhs.type() == rhs.type());

  switch (lhs.type()) {
    case JSONDocument::kNull:
      return 0;
    case JSONDocument::kBool:
      return static_cast<int>(lhs.GetBool()) - static_cast<int>(rhs.GetBool());
    case JSONDocument::kDouble: {
      double res = lhs.GetDouble() - rhs.GetDouble();
      return res == 0.0 ? 0 : (res < 0.0 ? -1 : 1);
    }
    case JSONDocument::kInt64: {
      int64_t res = lhs.GetInt64() - rhs.GetInt64();
      return res == 0 ? 0 : (res < 0 ? -1 : 1);
    }
    case JSONDocument::kString:
      return Slice(lhs.GetString()).compare(Slice(rhs.GetString()));
    default:
      assert(false);
  }
  return 0;
}
}  // namespace

class Filter {
 public:
  // returns nullptr on parse failure
  static Filter* ParseFilter(const JSONDocument& filter);

  struct Interval {
    JSONDocument upper_bound;
    JSONDocument lower_bound;
    bool upper_inclusive;
    bool lower_inclusive;
    Interval()
        : upper_bound(),
          lower_bound(),
          upper_inclusive(false),
          lower_inclusive(false) {}
    Interval(const JSONDocument& ub, const JSONDocument& lb, bool ui, bool li)
        : upper_bound(ub),
          lower_bound(lb),
          upper_inclusive(ui),
          lower_inclusive(li) {
    }

    void UpdateUpperBound(const JSONDocument& ub, bool inclusive);
    void UpdateLowerBound(const JSONDocument& lb, bool inclusive);
  };

  bool SatisfiesFilter(const JSONDocument& document) const;
  const Interval* GetInterval(const std::string& field) const;

 private:
  explicit Filter(const JSONDocument& filter) : filter_(filter.Copy()) {
    assert(filter_.IsOwner());
  }

  // copied from the parameter
  const JSONDocument filter_;
  // constant after construction
  std::unordered_map<std::string, Interval> intervals_;
};

void Filter::Interval::UpdateUpperBound(const JSONDocument& ub,
                                        bool inclusive) {
  bool update = upper_bound.IsNull();
  if (!update) {
    int cmp = DocumentCompare(upper_bound, ub);
    update = (cmp > 0) || (cmp == 0 && !inclusive);
  }
  if (update) {
    upper_bound = ub;
    upper_inclusive = inclusive;
  }
}

void Filter::Interval::UpdateLowerBound(const JSONDocument& lb,
                                        bool inclusive) {
  bool update = lower_bound.IsNull();
  if (!update) {
    int cmp = DocumentCompare(lower_bound, lb);
    update = (cmp < 0) || (cmp == 0 && !inclusive);
  }
  if (update) {
    lower_bound = lb;
    lower_inclusive = inclusive;
  }
}

Filter* Filter::ParseFilter(const JSONDocument& filter) {
  if (filter.IsObject() == false) {
    return nullptr;
  }

  std::unique_ptr<Filter> f(new Filter(filter));

  for (const auto& items : f->filter_.Items()) {
    if (items.first.size() && items.first[0] == '$') {
      // fields starting with '$' are commands
      continue;
    }
    assert(f->intervals_.find(items.first) == f->intervals_.end());
    if (items.second.IsObject()) {
      if (items.second.Count() == 0) {
        // uhm...?
        return nullptr;
      }
      Interval interval;
      for (const auto& condition : items.second.Items()) {
        if (condition.second.IsObject() || condition.second.IsArray()) {
          // comparison operators not defined on objects. invalid array
          return nullptr;
        }
        // comparison operators:
        if (condition.first == "$gt") {
          interval.UpdateLowerBound(condition.second, false);
        } else if (condition.first == "$gte") {
          interval.UpdateLowerBound(condition.second, true);
        } else if (condition.first == "$lt") {
          interval.UpdateUpperBound(condition.second, false);
        } else if (condition.first == "$lte") {
          interval.UpdateUpperBound(condition.second, true);
        } else {
          // TODO(icanadi) more logical operators
          return nullptr;
        }
      }
      f->intervals_.insert({items.first, interval});
    } else {
      // equality
      f->intervals_.insert(
          {items.first, Interval(items.second,
                                 items.second, true, true)});
    }
  }

  return f.release();
}

const Filter::Interval* Filter::GetInterval(const std::string& field) const {
  auto itr = intervals_.find(field);
  if (itr == intervals_.end()) {
    return nullptr;
  }
  // we can do that since intervals_ is constant after construction
  return &itr->second;
}

bool Filter::SatisfiesFilter(const JSONDocument& document) const {
  for (const auto& interval : intervals_) {
    if (!document.Contains(interval.first)) {
      // doesn't have the value, doesn't satisfy the filter
      // (we don't support null queries yet)
      return false;
    }
    auto value = document[interval.first];
    if (!interval.second.upper_bound.IsNull()) {
      if (value.type() != interval.second.upper_bound.type()) {
        // no cross-type queries yet
        // TODO(icanadi) do this at least for numbers!
        return false;
      }
      int cmp = DocumentCompare(interval.second.upper_bound, value);
      if (cmp < 0 || (cmp == 0 && interval.second.upper_inclusive == false)) {
        // bigger (or equal) than upper bound
        return false;
      }
    }
    if (!interval.second.lower_bound.IsNull()) {
      if (value.type() != interval.second.lower_bound.type()) {
        // no cross-type queries yet
        return false;
      }
      int cmp = DocumentCompare(interval.second.lower_bound, value);
      if (cmp > 0 || (cmp == 0 && interval.second.lower_inclusive == false)) {
        // smaller (or equal) than the lower bound
        return false;
      }
    }
  }
  return true;
}

class Index {
 public:
  Index() = default;
  virtual ~Index() {}

  virtual const char* Name() const = 0;

  // Functions that are executed during write time
  // ---------------------------------------------
  // GetIndexKey() generates a key that will be used to index document and
  // returns the key though the second std::string* parameter
  virtual void GetIndexKey(const JSONDocument& document,
                           std::string* key) const = 0;
  // Keys generated with GetIndexKey() will be compared using this comparator.
  // It should be assumed that there will be a suffix added to the index key
  // according to IndexKey implementation
  virtual const Comparator* GetComparator() const = 0;

  // Functions that are executed during query time
  // ---------------------------------------------
  enum Direction {
    kForwards,
    kBackwards,
  };
  // Returns true if this index can provide some optimization for satisfying
  // filter. False otherwise
  virtual bool UsefulIndex(const Filter& filter) const = 0;
  // For every filter (assuming UsefulIndex()) there is a continuous interval of
  // keys in the index that satisfy the index conditions. That interval can be
  // three things:
  // * [A, B]
  // * [A, infinity>
  // * <-infinity, B]
  //
  // Query engine that uses this Index for optimization will access the interval
  // by first calling Position() and then iterating in the Direction (returned
  // by Position()) while ShouldContinueLooking() is true.
  // * For [A, B] interval Position() will Seek() to A and return kForwards.
  // ShouldContinueLooking() will be true until the iterator value gets beyond B
  // -- then it will return false
  // * For [A, infinity> Position() will Seek() to A and return kForwards.
  // ShouldContinueLooking() will always return true
  // * For <-infinity, B] Position() will Seek() to B and return kBackwards.
  // ShouldContinueLooking() will always return true (given that iterator is
  // advanced by calling Prev())
  virtual Direction Position(const Filter& filter,
                             Iterator* iterator) const = 0;
  virtual bool ShouldContinueLooking(const Filter& filter,
                                     const Slice& secondary_key,
                                     Direction direction) const = 0;

  // Static function that is executed when Index is created
  // ---------------------------------------------
  // Create Index from user-supplied description. Return nullptr on parse
  // failure.
  static Index* CreateIndexFromDescription(const JSONDocument& description,
                                           const std::string& name);

 private:
  // No copying allowed
  Index(const Index&);
  void operator=(const Index&);
};

// Encoding helper function
namespace {
std::string InternalSecondaryIndexName(const std::string& user_name) {
  return "index_" + user_name;
}

// Don't change these, they are persisted in secondary indexes
enum JSONPrimitivesEncoding : char {
  kNull = 0x1,
  kBool = 0x2,
  kDouble = 0x3,
  kInt64 = 0x4,
  kString = 0x5,
};

// encodes simple JSON members (meaning string, integer, etc)
// the end result of this will be lexicographically compared to each other
bool EncodeJSONPrimitive(const JSONDocument& json, std::string* dst) {
  // TODO(icanadi) revise this at some point, have a custom comparator
  switch (json.type()) {
    case JSONDocument::kNull:
      dst->push_back(kNull);
      break;
    case JSONDocument::kBool:
      dst->push_back(kBool);
      dst->push_back(static_cast<char>(json.GetBool()));
      break;
    case JSONDocument::kDouble:
      dst->push_back(kDouble);
      PutFixed64(dst, static_cast<uint64_t>(json.GetDouble()));
      break;
    case JSONDocument::kInt64:
      dst->push_back(kInt64);
      {
        auto val = json.GetInt64();
        dst->push_back((val < 0) ? '0' : '1');
        PutFixed64(dst, static_cast<uint64_t>(val));
      }
      break;
    case JSONDocument::kString:
      dst->push_back(kString);
      dst->append(json.GetString());
      break;
    default:
      return false;
  }
  return true;
}

}  // namespace

// format of the secondary key is:
// <secondary_key><primary_key><offset_of_primary_key uint32_t>
class IndexKey {
 public:
  IndexKey() : ok_(false) {}
  explicit IndexKey(const Slice& slice) {
    if (slice.size() < sizeof(uint32_t)) {
      ok_ = false;
      return;
    }
    uint32_t primary_key_offset =
        DecodeFixed32(slice.data() + slice.size() - sizeof(uint32_t));
    if (primary_key_offset >= slice.size() - sizeof(uint32_t)) {
      ok_ = false;
      return;
    }
    parts_[0] = Slice(slice.data(), primary_key_offset);
    parts_[1] = Slice(slice.data() + primary_key_offset,
                      slice.size() - primary_key_offset - sizeof(uint32_t));
    ok_ = true;
  }
  IndexKey(const Slice& secondary_key, const Slice& primary_key) : ok_(true) {
    parts_[0] = secondary_key;
    parts_[1] = primary_key;
  }

  SliceParts GetSliceParts() {
    uint32_t primary_key_offset = static_cast<uint32_t>(parts_[0].size());
    EncodeFixed32(primary_key_offset_buf_, primary_key_offset);
    parts_[2] = Slice(primary_key_offset_buf_, sizeof(uint32_t));
    return SliceParts(parts_, 3);
  }

  const Slice& GetPrimaryKey() const { return parts_[1]; }
  const Slice& GetSecondaryKey() const { return parts_[0]; }

  bool ok() const { return ok_; }

 private:
  bool ok_;
  // 0 -- secondary key
  // 1 -- primary key
  // 2 -- primary key offset
  Slice parts_[3];
  char primary_key_offset_buf_[sizeof(uint32_t)];
};

class SimpleSortedIndex : public Index {
 public:
  SimpleSortedIndex(const std::string& field, const std::string& name)
      : field_(field), name_(name) {}

  virtual const char* Name() const override { return name_.c_str(); }

  virtual void GetIndexKey(const JSONDocument& document, std::string* key) const
      override {
    if (!document.Contains(field_)) {
      if (!EncodeJSONPrimitive(JSONDocument(JSONDocument::kNull), key)) {
        assert(false);
      }
    } else {
      if (!EncodeJSONPrimitive(document[field_], key)) {
        assert(false);
      }
    }
  }
  virtual const Comparator* GetComparator() const override {
    return BytewiseComparator();
  }

  virtual bool UsefulIndex(const Filter& filter) const override {
    return filter.GetInterval(field_) != nullptr;
  }
  // REQUIRES: UsefulIndex(filter) == true
  virtual Direction Position(const Filter& filter,
                             Iterator* iterator) const override {
    auto interval = filter.GetInterval(field_);
    assert(interval != nullptr);  // because index is useful
    Direction direction;

    const JSONDocument* limit;
    if (!interval->lower_bound.IsNull()) {
      limit = &(interval->lower_bound);
      direction = kForwards;
    } else {
      limit = &(interval->upper_bound);
      direction = kBackwards;
    }

    std::string encoded_limit;
    if (!EncodeJSONPrimitive(*limit, &encoded_limit)) {
      assert(false);
    }
    iterator->Seek(Slice(encoded_limit));

    return direction;
  }
  // REQUIRES: UsefulIndex(filter) == true
  virtual bool ShouldContinueLooking(
      const Filter& filter, const Slice& secondary_key,
      Index::Direction direction) const override {
    auto interval = filter.GetInterval(field_);
    assert(interval != nullptr);  // because index is useful
    if (direction == kForwards) {
      if (interval->upper_bound.IsNull()) {
        // continue looking, no upper bound
        return true;
      }
      std::string encoded_upper_bound;
      if (!EncodeJSONPrimitive(interval->upper_bound, &encoded_upper_bound)) {
        // uhm...?
        // TODO(icanadi) store encoded upper and lower bounds in Filter*?
        assert(false);
      }
      // TODO(icanadi) we need to somehow decode this and use DocumentCompare()
      int compare = secondary_key.compare(Slice(encoded_upper_bound));
      // if (current key is bigger than upper bound) OR (current key is equal to
      // upper bound, but inclusive is false) THEN stop looking. otherwise,
      // continue
      return (compare > 0 ||
              (compare == 0 && interval->upper_inclusive == false))
                 ? false
                 : true;
    } else {
      assert(direction == kBackwards);
      if (interval->lower_bound.IsNull()) {
        // continue looking, no lower bound
        return true;
      }
      std::string encoded_lower_bound;
      if (!EncodeJSONPrimitive(interval->lower_bound, &encoded_lower_bound)) {
        // uhm...?
        // TODO(icanadi) store encoded upper and lower bounds in Filter*?
        assert(false);
      }
      // TODO(icanadi) we need to somehow decode this and use DocumentCompare()
      int compare = secondary_key.compare(Slice(encoded_lower_bound));
      // if (current key is smaller than lower bound) OR (current key is equal
      // to lower bound, but inclusive is false) THEN stop looking. otherwise,
      // continue
      return (compare < 0 ||
              (compare == 0 && interval->lower_inclusive == false))
                 ? false
                 : true;
    }

    assert(false);
    // this is here just so compiler doesn't complain
    return false;
  }

 private:
  std::string field_;
  std::string name_;
};

Index* Index::CreateIndexFromDescription(const JSONDocument& description,
                                         const std::string& name) {
  if (!description.IsObject() || description.Count() != 1) {
    // not supported yet
    return nullptr;
  }
  const auto& field = *description.Items().begin();
  if (field.second.IsInt64() == false || field.second.GetInt64() != 1) {
    // not supported yet
    return nullptr;
  }
  return new SimpleSortedIndex(field.first, name);
}

class CursorWithFilterIndexed : public Cursor {
 public:
  CursorWithFilterIndexed(Iterator* primary_index_iter,
                          Iterator* secondary_index_iter, const Index* index,
                          const Filter* filter)
      : primary_index_iter_(primary_index_iter),
        secondary_index_iter_(secondary_index_iter),
        index_(index),
        filter_(filter),
        valid_(true),
        current_json_document_(nullptr) {
    assert(filter_.get() != nullptr);
    direction_ = index->Position(*filter_.get(), secondary_index_iter_.get());
    UpdateIndexKey();
    AdvanceUntilSatisfies();
  }

  virtual bool Valid() const override {
    return valid_ && secondary_index_iter_->Valid();
  }
  virtual void Next() override {
    assert(Valid());
    Advance();
    AdvanceUntilSatisfies();
  }
  // temporary object. copy it if you want to use it
  virtual const JSONDocument& document() const override {
    assert(Valid());
    return *current_json_document_;
  }
  virtual Status status() const override {
    if (!status_.ok()) {
      return status_;
    }
    if (!primary_index_iter_->status().ok()) {
      return primary_index_iter_->status();
    }
    return secondary_index_iter_->status();
  }

 private:
  void Advance() {
    if (direction_ == Index::kForwards) {
      secondary_index_iter_->Next();
    } else {
      secondary_index_iter_->Prev();
    }
    UpdateIndexKey();
  }
  void AdvanceUntilSatisfies() {
    bool found = false;
    while (secondary_index_iter_->Valid() &&
           index_->ShouldContinueLooking(
               *filter_.get(), index_key_.GetSecondaryKey(), direction_)) {
      if (!UpdateJSONDocument()) {
        // corruption happened
        return;
      }
      if (filter_->SatisfiesFilter(*current_json_document_)) {
        // we found satisfied!
        found = true;
        break;
      } else {
        // doesn't satisfy :(
        Advance();
      }
    }
    if (!found) {
      valid_ = false;
    }
  }

  bool UpdateJSONDocument() {
    assert(secondary_index_iter_->Valid());
    primary_index_iter_->Seek(index_key_.GetPrimaryKey());
    if (!primary_index_iter_->Valid()) {
      status_ = Status::Corruption(
          "Inconsistency between primary and secondary index");
      valid_ = false;
      return false;
    }
    current_json_document_.reset(
        JSONDocument::Deserialize(primary_index_iter_->value()));
    assert(current_json_document_->IsOwner());
    if (current_json_document_.get() == nullptr) {
      status_ = Status::Corruption("JSON deserialization failed");
      valid_ = false;
      return false;
    }
    return true;
  }
  void UpdateIndexKey() {
    if (secondary_index_iter_->Valid()) {
      index_key_ = IndexKey(secondary_index_iter_->key());
      if (!index_key_.ok()) {
        status_ = Status::Corruption("Invalid index key");
        valid_ = false;
      }
    }
  }
  std::unique_ptr<Iterator> primary_index_iter_;
  std::unique_ptr<Iterator> secondary_index_iter_;
  // we don't own index_
  const Index* index_;
  Index::Direction direction_;
  std::unique_ptr<const Filter> filter_;
  bool valid_;
  IndexKey index_key_;
  std::unique_ptr<JSONDocument> current_json_document_;
  Status status_;
};

class CursorFromIterator : public Cursor {
 public:
  explicit CursorFromIterator(Iterator* iter)
      : iter_(iter), current_json_document_(nullptr) {
    iter_->SeekToFirst();
    UpdateCurrentJSON();
  }

  virtual bool Valid() const override { return status_.ok() && iter_->Valid(); }
  virtual void Next() override {
    iter_->Next();
    UpdateCurrentJSON();
  }
  virtual const JSONDocument& document() const override {
    assert(Valid());
    return *current_json_document_;
  };
  virtual Status status() const override {
    if (!status_.ok()) {
      return status_;
    }
    return iter_->status();
  }

  // not part of public Cursor interface
  Slice key() const { return iter_->key(); }

 private:
  void UpdateCurrentJSON() {
    if (Valid()) {
      current_json_document_.reset(JSONDocument::Deserialize(iter_->value()));
      if (current_json_document_.get() == nullptr) {
        status_ = Status::Corruption("JSON deserialization failed");
      }
    }
  }

  Status status_;
  std::unique_ptr<Iterator> iter_;
  std::unique_ptr<JSONDocument> current_json_document_;
};

class CursorWithFilter : public Cursor {
 public:
  CursorWithFilter(Cursor* base_cursor, const Filter* filter)
      : base_cursor_(base_cursor), filter_(filter) {
    assert(filter_.get() != nullptr);
    SeekToNextSatisfies();
  }
  virtual bool Valid() const override { return base_cursor_->Valid(); }
  virtual void Next() override {
    assert(Valid());
    base_cursor_->Next();
    SeekToNextSatisfies();
  }
  virtual const JSONDocument& document() const override {
    assert(Valid());
    return base_cursor_->document();
  }
  virtual Status status() const override { return base_cursor_->status(); }

 private:
  void SeekToNextSatisfies() {
    for (; base_cursor_->Valid(); base_cursor_->Next()) {
      if (filter_->SatisfiesFilter(base_cursor_->document())) {
        break;
      }
    }
  }
  std::unique_ptr<Cursor> base_cursor_;
  std::unique_ptr<const Filter> filter_;
};

class CursorError : public Cursor {
 public:
  explicit CursorError(Status s) : s_(s) { assert(!s.ok()); }
  virtual Status status() const override { return s_; }
  virtual bool Valid() const override { return false; }
  virtual void Next() override {}
  virtual const JSONDocument& document() const override {
    assert(false);
    // compiler complains otherwise
    return trash_;
  }

 private:
  Status s_;
  JSONDocument trash_;
};

class DocumentDBImpl : public DocumentDB {
 public:
  DocumentDBImpl(
      DB* db, ColumnFamilyHandle* primary_key_column_family,
      const std::vector<std::pair<Index*, ColumnFamilyHandle*>>& indexes,
      const Options& rocksdb_options)
      : DocumentDB(db),
        primary_key_column_family_(primary_key_column_family),
        rocksdb_options_(rocksdb_options) {
    for (const auto& index : indexes) {
      name_to_index_.insert(
          {index.first->Name(), IndexColumnFamily(index.first, index.second)});
    }
  }

  ~DocumentDBImpl() {
    for (auto& iter : name_to_index_) {
      delete iter.second.index;
      delete iter.second.column_family;
    }
    delete primary_key_column_family_;
  }

  virtual Status CreateIndex(const WriteOptions& write_options,
                             const IndexDescriptor& index) override {
    auto index_obj =
        Index::CreateIndexFromDescription(*index.description, index.name);
    if (index_obj == nullptr) {
      return Status::InvalidArgument("Failed parsing index description");
    }

    ColumnFamilyHandle* cf_handle;
    Status s =
        CreateColumnFamily(ColumnFamilyOptions(rocksdb_options_),
                           InternalSecondaryIndexName(index.name), &cf_handle);
    if (!s.ok()) {
      delete index_obj;
      return s;
    }

    MutexLock l(&write_mutex_);

    std::unique_ptr<CursorFromIterator> cursor(new CursorFromIterator(
        DocumentDB::NewIterator(ReadOptions(), primary_key_column_family_)));

    WriteBatch batch;
    for (; cursor->Valid(); cursor->Next()) {
      std::string secondary_index_key;
      index_obj->GetIndexKey(cursor->document(), &secondary_index_key);
      IndexKey index_key(Slice(secondary_index_key), cursor->key());
      batch.Put(cf_handle, index_key.GetSliceParts(), SliceParts());
    }

    if (!cursor->status().ok()) {
      delete index_obj;
      return cursor->status();
    }

    {
      MutexLock l_nti(&name_to_index_mutex_);
      name_to_index_.insert(
          {index.name, IndexColumnFamily(index_obj, cf_handle)});
    }

    return DocumentDB::Write(write_options, &batch);
  }

  virtual Status DropIndex(const std::string& name) override {
    MutexLock l(&write_mutex_);

    auto index_iter = name_to_index_.find(name);
    if (index_iter == name_to_index_.end()) {
      return Status::InvalidArgument("No such index");
    }

    Status s = DropColumnFamily(index_iter->second.column_family);
    if (!s.ok()) {
      return s;
    }

    delete index_iter->second.index;
    delete index_iter->second.column_family;

    // remove from name_to_index_
    {
      MutexLock l_nti(&name_to_index_mutex_);
      name_to_index_.erase(index_iter);
    }

    return Status::OK();
  }

  virtual Status Insert(const WriteOptions& options,
                        const JSONDocument& document) override {
    WriteBatch batch;

    if (!document.IsObject()) {
      return Status::InvalidArgument("Document not an object");
    }
    if (!document.Contains(kPrimaryKey)) {
      return Status::InvalidArgument("No primary key");
    }
    auto primary_key = document[kPrimaryKey];
    if (primary_key.IsNull() ||
        (!primary_key.IsString() && !primary_key.IsInt64())) {
      return Status::InvalidArgument(
          "Primary key format error");
    }
    std::string encoded_document;
    document.Serialize(&encoded_document);
    std::string primary_key_encoded;
    if (!EncodeJSONPrimitive(primary_key, &primary_key_encoded)) {
      // previous call should be guaranteed to pass because of all primary_key
      // conditions checked before
      assert(false);
    }
    Slice primary_key_slice(primary_key_encoded);

    // Lock now, since we're starting DB operations
    MutexLock l(&write_mutex_);
    // check if there is already a document with the same primary key
    std::string value;
    Status s = DocumentDB::Get(ReadOptions(), primary_key_column_family_,
                               primary_key_slice, &value);
    if (!s.IsNotFound()) {
      return s.ok() ? Status::InvalidArgument("Duplicate primary key!") : s;
    }

    batch.Put(primary_key_column_family_, primary_key_slice, encoded_document);

    for (const auto& iter : name_to_index_) {
      std::string secondary_index_key;
      iter.second.index->GetIndexKey(document, &secondary_index_key);
      IndexKey index_key(Slice(secondary_index_key), primary_key_slice);
      batch.Put(iter.second.column_family, index_key.GetSliceParts(),
                SliceParts());
    }

    return DocumentDB::Write(options, &batch);
  }

  virtual Status Remove(const ReadOptions& read_options,
                        const WriteOptions& write_options,
                        const JSONDocument& query) override {
    MutexLock l(&write_mutex_);
    std::unique_ptr<Cursor> cursor(
        ConstructFilterCursor(read_options, nullptr, query));

    WriteBatch batch;
    for (; cursor->status().ok() && cursor->Valid(); cursor->Next()) {
      const auto& document = cursor->document();
      if (!document.IsObject()) {
        return Status::Corruption("Document corruption");
      }
      if (!document.Contains(kPrimaryKey)) {
        return Status::Corruption("Document corruption");
      }
      auto primary_key = document[kPrimaryKey];
      if (primary_key.IsNull() ||
          (!primary_key.IsString() && !primary_key.IsInt64())) {
        return Status::Corruption("Document corruption");
      }

      // TODO(icanadi) Instead of doing this, just get primary key encoding from
      // cursor, as it already has this information
      std::string primary_key_encoded;
      if (!EncodeJSONPrimitive(primary_key, &primary_key_encoded)) {
        // previous call should be guaranteed to pass because of all primary_key
        // conditions checked before
        assert(false);
      }
      Slice primary_key_slice(primary_key_encoded);
      batch.Delete(primary_key_column_family_, primary_key_slice);

      for (const auto& iter : name_to_index_) {
        std::string secondary_index_key;
        iter.second.index->GetIndexKey(document, &secondary_index_key);
        IndexKey index_key(Slice(secondary_index_key), primary_key_slice);
        batch.Delete(iter.second.column_family, index_key.GetSliceParts());
      }
    }

    if (!cursor->status().ok()) {
      return cursor->status();
    }

    return DocumentDB::Write(write_options, &batch);
  }

  virtual Status Update(const ReadOptions& read_options,
                        const WriteOptions& write_options,
                        const JSONDocument& filter,
                        const JSONDocument& updates) override {
    MutexLock l(&write_mutex_);
    std::unique_ptr<Cursor> cursor(
        ConstructFilterCursor(read_options, nullptr, filter));

    if (!updates.IsObject()) {
        return Status::Corruption("Bad update document format");
    }
    WriteBatch batch;
    for (; cursor->status().ok() && cursor->Valid(); cursor->Next()) {
      const auto& old_document = cursor->document();
      JSONDocument new_document(old_document);
      if (!new_document.IsObject()) {
        return Status::Corruption("Document corruption");
      }
      // TODO(icanadi) Make this nicer, something like class Filter
      for (const auto& update : updates.Items()) {
        if (update.first == "$set") {
          JSONDocumentBuilder builder;
          bool res __attribute__((unused)) = builder.WriteStartObject();
          assert(res);
          for (const auto& itr : update.second.Items()) {
            if (itr.first == kPrimaryKey) {
              return Status::NotSupported("Please don't change primary key");
            }
            res = builder.WriteKeyValue(itr.first, itr.second);
            assert(res);
          }
          res = builder.WriteEndObject();
          assert(res);
          JSONDocument update_document = builder.GetJSONDocument();
          builder.Reset();
          res = builder.WriteStartObject();
          assert(res);
          for (const auto& itr : new_document.Items()) {
            if (update_document.Contains(itr.first)) {
              res = builder.WriteKeyValue(itr.first,
                                          update_document[itr.first]);
            } else {
              res = builder.WriteKeyValue(itr.first, new_document[itr.first]);
            }
            assert(res);
          }
          res = builder.WriteEndObject();
          assert(res);
          new_document = builder.GetJSONDocument();
          assert(new_document.IsOwner());
        } else {
          // TODO(icanadi) more commands
          return Status::InvalidArgument("Can't understand update command");
        }
      }

      // TODO(icanadi) reuse some of this code
      if (!new_document.Contains(kPrimaryKey)) {
        return Status::Corruption("Corrupted document -- primary key missing");
      }
      auto primary_key = new_document[kPrimaryKey];
      if (primary_key.IsNull() ||
          (!primary_key.IsString() && !primary_key.IsInt64())) {
        // This will happen when document on storage doesn't have primary key,
        // since we don't support any update operations on primary key. That's
        // why this is corruption error
        return Status::Corruption("Corrupted document -- primary key missing");
      }
      std::string encoded_document;
      new_document.Serialize(&encoded_document);
      std::string primary_key_encoded;
      if (!EncodeJSONPrimitive(primary_key, &primary_key_encoded)) {
        // previous call should be guaranteed to pass because of all primary_key
        // conditions checked before
        assert(false);
      }
      Slice primary_key_slice(primary_key_encoded);
      batch.Put(primary_key_column_family_, primary_key_slice,
                encoded_document);

      for (const auto& iter : name_to_index_) {
        std::string old_key, new_key;
        iter.second.index->GetIndexKey(old_document, &old_key);
        iter.second.index->GetIndexKey(new_document, &new_key);
        if (old_key == new_key) {
          // don't need to update this secondary index
          continue;
        }

        IndexKey old_index_key(Slice(old_key), primary_key_slice);
        IndexKey new_index_key(Slice(new_key), primary_key_slice);

        batch.Delete(iter.second.column_family, old_index_key.GetSliceParts());
        batch.Put(iter.second.column_family, new_index_key.GetSliceParts(),
                  SliceParts());
      }
    }

    if (!cursor->status().ok()) {
      return cursor->status();
    }

    return DocumentDB::Write(write_options, &batch);
  }

  virtual Cursor* Query(const ReadOptions& read_options,
                        const JSONDocument& query) override {
    Cursor* cursor = nullptr;

    if (!query.IsArray()) {
      return new CursorError(
          Status::InvalidArgument("Query has to be an array"));
    }

    // TODO(icanadi) support index "_id"
    for (size_t i = 0; i < query.Count(); ++i) {
      const auto& command_doc = query[i];
      if (command_doc.Count() != 1) {
        // there can be only one key-value pair in each of array elements.
        // key is the command and value are the params
        delete cursor;
        return new CursorError(Status::InvalidArgument("Invalid query"));
      }
      const auto& command = *command_doc.Items().begin();

      if (command.first == "$filter") {
        cursor = ConstructFilterCursor(read_options, cursor, command.second);
      } else {
        // only filter is supported for now
        delete cursor;
        return new CursorError(Status::InvalidArgument("Invalid query"));
      }
    }

    if (cursor == nullptr) {
      cursor = new CursorFromIterator(
          DocumentDB::NewIterator(read_options, primary_key_column_family_));
    }

    return cursor;
  }

  // RocksDB functions
  virtual Status Get(const ReadOptions& options,
                     ColumnFamilyHandle* column_family, const Slice& key,
                     std::string* value) override {
    return Status::NotSupported("");
  }
  virtual Status Get(const ReadOptions& options, const Slice& key,
                     std::string* value) override {
    return Status::NotSupported("");
  }
  virtual Status Write(const WriteOptions& options,
                       WriteBatch* updates) override {
    return Status::NotSupported("");
  }
  virtual Iterator* NewIterator(const ReadOptions& options,
                                ColumnFamilyHandle* column_family) override {
    return nullptr;
  }
  virtual Iterator* NewIterator(const ReadOptions& options) override {
    return nullptr;
  }

 private:
  Cursor* ConstructFilterCursor(ReadOptions read_options, Cursor* cursor,
                                const JSONDocument& query) {
    std::unique_ptr<const Filter> filter(Filter::ParseFilter(query));
    if (filter.get() == nullptr) {
      return new CursorError(Status::InvalidArgument("Invalid query"));
    }

    IndexColumnFamily tmp_storage(nullptr, nullptr);

    if (cursor == nullptr) {
      IndexColumnFamily* index_column_family = nullptr;
      if (query.Contains("$index") && query["$index"].IsString()) {
        {
          auto index_name = query["$index"];
          MutexLock l(&name_to_index_mutex_);
          auto index_iter = name_to_index_.find(index_name.GetString());
          if (index_iter != name_to_index_.end()) {
            tmp_storage = index_iter->second;
            index_column_family = &tmp_storage;
          } else {
            return new CursorError(
                Status::InvalidArgument("Index does not exist"));
          }
        }
      }

      if (index_column_family != nullptr &&
          index_column_family->index->UsefulIndex(*filter.get())) {
        std::vector<Iterator*> iterators;
        Status s = DocumentDB::NewIterators(
            read_options,
            {primary_key_column_family_, index_column_family->column_family},
            &iterators);
        if (!s.ok()) {
          delete cursor;
          return new CursorError(s);
        }
        assert(iterators.size() == 2);
        return new CursorWithFilterIndexed(iterators[0], iterators[1],
                                           index_column_family->index,
                                           filter.release());
      } else {
        return new CursorWithFilter(
            new CursorFromIterator(DocumentDB::NewIterator(
                read_options, primary_key_column_family_)),
            filter.release());
      }
    } else {
      return new CursorWithFilter(cursor, filter.release());
    }
    assert(false);
    return nullptr;
  }

  // currently, we lock and serialize all writes to rocksdb. reads are not
  // locked and always get consistent view of the database. we should optimize
  // locking in the future
  port::Mutex write_mutex_;
  port::Mutex name_to_index_mutex_;
  const char* kPrimaryKey = "_id";
  struct IndexColumnFamily {
    IndexColumnFamily(Index* _index, ColumnFamilyHandle* _column_family)
        : index(_index), column_family(_column_family) {}
    Index* index;
    ColumnFamilyHandle* column_family;
  };


  // name_to_index_ protected:
  // 1) when writing -- 1. lock write_mutex_, 2. lock name_to_index_mutex_
  // 2) when reading -- lock name_to_index_mutex_ OR write_mutex_
  std::unordered_map<std::string, IndexColumnFamily> name_to_index_;
  ColumnFamilyHandle* primary_key_column_family_;
  Options rocksdb_options_;
};

namespace {
Options GetRocksDBOptionsFromOptions(const DocumentDBOptions& options) {
  Options rocksdb_options;
  rocksdb_options.max_background_compactions = options.background_threads - 1;
  rocksdb_options.max_background_flushes = 1;
  rocksdb_options.write_buffer_size = options.memtable_size;
  rocksdb_options.max_write_buffer_number = 6;
  BlockBasedTableOptions table_options;
  table_options.block_cache = NewLRUCache(options.cache_size);
  rocksdb_options.table_factory.reset(NewBlockBasedTableFactory(table_options));
  return rocksdb_options;
}
}  // namespace

Status DocumentDB::Open(const DocumentDBOptions& options,
                        const std::string& name,
                        const std::vector<DocumentDB::IndexDescriptor>& indexes,
                        DocumentDB** db, bool read_only) {
  Options rocksdb_options = GetRocksDBOptionsFromOptions(options);
  rocksdb_options.create_if_missing = true;

  std::vector<ColumnFamilyDescriptor> column_families;
  column_families.push_back(ColumnFamilyDescriptor(
      kDefaultColumnFamilyName, ColumnFamilyOptions(rocksdb_options)));
  for (const auto& index : indexes) {
    column_families.emplace_back(InternalSecondaryIndexName(index.name),
                                 ColumnFamilyOptions(rocksdb_options));
  }
  std::vector<ColumnFamilyHandle*> handles;
  DB* base_db;
  Status s;
  if (read_only) {
    s = DB::OpenForReadOnly(DBOptions(rocksdb_options), name, column_families,
                            &handles, &base_db);
  } else {
    s = DB::Open(DBOptions(rocksdb_options), name, column_families, &handles,
                 &base_db);
  }
  if (!s.ok()) {
    return s;
  }

  std::vector<std::pair<Index*, ColumnFamilyHandle*>> index_cf(indexes.size());
  assert(handles.size() == indexes.size() + 1);
  for (size_t i = 0; i < indexes.size(); ++i) {
    auto index = Index::CreateIndexFromDescription(*indexes[i].description,
                                                   indexes[i].name);
    index_cf[i] = {index, handles[i + 1]};
  }
  *db = new DocumentDBImpl(base_db, handles[0], index_cf, rocksdb_options);
  return Status::OK();
}

}  // namespace rocksdb
#endif  // ROCKSDB_LITE
