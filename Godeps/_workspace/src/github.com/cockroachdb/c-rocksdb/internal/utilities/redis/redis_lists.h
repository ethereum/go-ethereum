/**
 * A (persistent) Redis API built using the rocksdb backend.
 * Implements Redis Lists as described on: http://redis.io/commands#list
 *
 * @throws All functions may throw a RedisListException
 *
 * @author Deon Nicholas (dnicholas@fb.com)
 * Copyright 2013 Facebook
 */

#ifndef ROCKSDB_LITE
#pragma once

#include <string>
#include "rocksdb/db.h"
#include "redis_list_iterator.h"
#include "redis_list_exception.h"

namespace rocksdb {

/// The Redis functionality (see http://redis.io/commands#list)
/// All functions may THROW a RedisListException
class RedisLists {
 public: // Constructors / Destructors
  /// Construct a new RedisLists database, with name/path of db.
  /// Will clear the database on open iff destructive is true (default false).
  /// Otherwise, it will restore saved changes.
  /// May throw RedisListException
  RedisLists(const std::string& db_path,
             Options options, bool destructive = false);

 public:  // Accessors
  /// The number of items in (list: key)
  int Length(const std::string& key);

  /// Search the list for the (index)'th item (0-based) in (list:key)
  /// A negative index indicates: "from end-of-list"
  /// If index is within range: return true, and return the value in *result.
  /// If (index < -length OR index>=length), then index is out of range:
  ///   return false (and *result is left unchanged)
  /// May throw RedisListException
  bool Index(const std::string& key, int32_t index,
             std::string* result);

  /// Return (list: key)[first..last] (inclusive)
  /// May throw RedisListException
  std::vector<std::string> Range(const std::string& key,
                                 int32_t first, int32_t last);

  /// Prints the entire (list: key), for debugging.
  void Print(const std::string& key);

 public: // Insert/Update
  /// Insert value before/after pivot in (list: key). Return the length.
  /// May throw RedisListException
  int InsertBefore(const std::string& key, const std::string& pivot,
                   const std::string& value);
  int InsertAfter(const std::string& key, const std::string& pivot,
                  const std::string& value);

  /// Push / Insert value at beginning/end of the list. Return the length.
  /// May throw RedisListException
  int PushLeft(const std::string& key, const std::string& value);
  int PushRight(const std::string& key, const std::string& value);

  /// Set (list: key)[idx] = val. Return true on success, false on fail
  /// May throw RedisListException
  bool Set(const std::string& key, int32_t index, const std::string& value);

 public: // Delete / Remove / Pop / Trim
  /// Trim (list: key) so that it will only contain the indices from start..stop
  /// Returns true on success
  /// May throw RedisListException
  bool Trim(const std::string& key, int32_t start, int32_t stop);

  /// If list is empty, return false and leave *result unchanged.
  /// Else, remove the first/last elem, store it in *result, and return true
  bool PopLeft(const std::string& key, std::string* result);  // First
  bool PopRight(const std::string& key, std::string* result); // Last

  /// Remove the first (or last) num occurrences of value from the list (key)
  /// Return the number of elements removed.
  /// May throw RedisListException
  int Remove(const std::string& key, int32_t num,
             const std::string& value);
  int RemoveFirst(const std::string& key, int32_t num,
                  const std::string& value);
  int RemoveLast(const std::string& key, int32_t num,
                 const std::string& value);

 private: // Private Functions
  /// Calls InsertBefore or InsertAfter
  int Insert(const std::string& key, const std::string& pivot,
             const std::string& value, bool insert_after);
 private:
  std::string db_name_;       // The actual database name/path
  WriteOptions put_option_;
  ReadOptions get_option_;

  /// The backend rocksdb database.
  /// Map : key --> list
  ///       where a list is a sequence of elements
  ///       and an element is a 4-byte integer (n), followed by n bytes of data
  std::unique_ptr<DB> db_;
};

} // namespace rocksdb
#endif  // ROCKSDB_LITE
