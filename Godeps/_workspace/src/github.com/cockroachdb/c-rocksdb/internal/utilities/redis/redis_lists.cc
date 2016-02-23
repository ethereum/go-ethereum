// Copyright 2013 Facebook
/**
 * A (persistent) Redis API built using the rocksdb backend.
 * Implements Redis Lists as described on: http://redis.io/commands#list
 *
 * @throws All functions may throw a RedisListException on error/corruption.
 *
 * @notes Internally, the set of lists is stored in a rocksdb database,
 *        mapping keys to values. Each "value" is the list itself, storing
 *        some kind of internal representation of the data. All the
 *        representation details are handled by the RedisListIterator class.
 *        The present file should be oblivious to the representation details,
 *        handling only the client (Redis) API, and the calls to rocksdb.
 *
 * @TODO  Presently, all operations take at least O(NV) time where
 *        N is the number of elements in the list, and V is the average
 *        number of bytes per value in the list. So maybe, with merge operator
 *        we can improve this to an optimal O(V) amortized time, since we
 *        wouldn't have to read and re-write the entire list.
 *
 * @author Deon Nicholas (dnicholas@fb.com)
 */

#ifndef ROCKSDB_LITE
#include "redis_lists.h"

#include <iostream>
#include <memory>
#include <cmath>

#include "rocksdb/slice.h"
#include "util/coding.h"

namespace rocksdb
{

/// Constructors

RedisLists::RedisLists(const std::string& db_path,
                       Options options, bool destructive)
    : put_option_(),
      get_option_() {

  // Store the name of the database
  db_name_ = db_path;

  // If destructive, destroy the DB before re-opening it.
  if (destructive) {
    DestroyDB(db_name_, Options());
  }

  // Now open and deal with the db
  DB* db;
  Status s = DB::Open(options, db_name_, &db);
  if (!s.ok()) {
    std::cerr << "ERROR " << s.ToString() << std::endl;
    assert(false);
  }

  db_ = std::unique_ptr<DB>(db);
}


/// Accessors

// Number of elements in the list associated with key
//   : throws RedisListException
int RedisLists::Length(const std::string& key) {
  // Extract the string data representing the list.
  std::string data;
  db_->Get(get_option_, key, &data);

  // Return the length
  RedisListIterator it(data);
  return it.Length();
}

// Get the element at the specified index in the (list: key)
// Returns <empty> ("") on out-of-bounds
//   : throws RedisListException
bool RedisLists::Index(const std::string& key, int32_t index,
                       std::string* result) {
  // Extract the string data representing the list.
  std::string data;
  db_->Get(get_option_, key, &data);

  // Handle REDIS negative indices (from the end); fast iff Length() takes O(1)
  if (index < 0) {
    index = Length(key) - (-index);  //replace (-i) with (N-i).
  }

  // Iterate through the list until the desired index is found.
  int curIndex = 0;
  RedisListIterator it(data);
  while(curIndex < index && !it.Done()) {
    ++curIndex;
    it.Skip();
  }

  // If we actually found the index
  if (curIndex == index && !it.Done()) {
    Slice elem;
    it.GetCurrent(&elem);
    if (result != NULL) {
      *result = elem.ToString();
    }

    return true;
  } else {
    return false;
  }
}

// Return a truncated version of the list.
// First, negative values for first/last are interpreted as "end of list".
// So, if first == -1, then it is re-set to index: (Length(key) - 1)
// Then, return exactly those indices i such that first <= i <= last.
//   : throws RedisListException
std::vector<std::string> RedisLists::Range(const std::string& key,
                                           int32_t first, int32_t last) {
  // Extract the string data representing the list.
  std::string data;
  db_->Get(get_option_, key, &data);

  // Handle negative bounds (-1 means last element, etc.)
  int listLen = Length(key);
  if (first < 0) {
    first = listLen - (-first);           // Replace (-x) with (N-x)
  }
  if (last < 0) {
    last = listLen - (-last);
  }

  // Verify bounds (and truncate the range so that it is valid)
  first = std::max(first, 0);
  last = std::min(last, listLen-1);
  int len = std::max(last-first+1, 0);

  // Initialize the resulting list
  std::vector<std::string> result(len);

  // Traverse the list and update the vector
  int curIdx = 0;
  Slice elem;
  for (RedisListIterator it(data); !it.Done() && curIdx<=last; it.Skip()) {
    if (first <= curIdx && curIdx <= last) {
      it.GetCurrent(&elem);
      result[curIdx-first].assign(elem.data(),elem.size());
    }

    ++curIdx;
  }

  // Return the result. Might be empty
  return result;
}

// Print the (list: key) out to stdout. For debugging mostly. Public for now.
void RedisLists::Print(const std::string& key) {
  // Extract the string data representing the list.
  std::string data;
  db_->Get(get_option_, key, &data);

  // Iterate through the list and print the items
  Slice elem;
  for (RedisListIterator it(data); !it.Done(); it.Skip()) {
    it.GetCurrent(&elem);
    std::cout << "ITEM " << elem.ToString() << std::endl;
  }

  //Now print the byte data
  RedisListIterator it(data);
  std::cout << "==Printing data==" << std::endl;
  std::cout << data.size() << std::endl;
  std::cout << it.Size() << " " << it.Length() << std::endl;
  Slice result = it.WriteResult();
  std::cout << result.data() << std::endl;
  if (true) {
    std::cout << "size: " << result.size() << std::endl;
    const char* val = result.data();
    for(int i=0; i<(int)result.size(); ++i) {
      std::cout << (int)val[i] << " " << (val[i]>=32?val[i]:' ') << std::endl;
    }
    std::cout << std::endl;
  }
}

/// Insert/Update Functions
/// Note: The "real" insert function is private. See below.

// InsertBefore and InsertAfter are simply wrappers around the Insert function.
int RedisLists::InsertBefore(const std::string& key, const std::string& pivot,
                             const std::string& value) {
  return Insert(key, pivot, value, false);
}

int RedisLists::InsertAfter(const std::string& key, const std::string& pivot,
                            const std::string& value) {
  return Insert(key, pivot, value, true);
}

// Prepend value onto beginning of (list: key)
//   : throws RedisListException
int RedisLists::PushLeft(const std::string& key, const std::string& value) {
  // Get the original list data
  std::string data;
  db_->Get(get_option_, key, &data);

  // Construct the result
  RedisListIterator it(data);
  it.Reserve(it.Size() + it.SizeOf(value));
  it.InsertElement(value);

  // Push the data back to the db and return the length
  db_->Put(put_option_, key, it.WriteResult());
  return it.Length();
}

// Append value onto end of (list: key)
// TODO: Make this O(1) time. Might require MergeOperator.
//   : throws RedisListException
int RedisLists::PushRight(const std::string& key, const std::string& value) {
  // Get the original list data
  std::string data;
  db_->Get(get_option_, key, &data);

  // Create an iterator to the data and seek to the end.
  RedisListIterator it(data);
  it.Reserve(it.Size() + it.SizeOf(value));
  while (!it.Done()) {
    it.Push();    // Write each element as we go
  }

  // Insert the new element at the current position (the end)
  it.InsertElement(value);

  // Push it back to the db, and return length
  db_->Put(put_option_, key, it.WriteResult());
  return it.Length();
}

// Set (list: key)[idx] = val. Return true on success, false on fail.
//   : throws RedisListException
bool RedisLists::Set(const std::string& key, int32_t index,
                     const std::string& value) {
  // Get the original list data
  std::string data;
  db_->Get(get_option_, key, &data);

  // Handle negative index for REDIS (meaning -index from end of list)
  if (index < 0) {
    index = Length(key) - (-index);
  }

  // Iterate through the list until we find the element we want
  int curIndex = 0;
  RedisListIterator it(data);
  it.Reserve(it.Size() + it.SizeOf(value));  // Over-estimate is fine
  while(curIndex < index && !it.Done()) {
    it.Push();
    ++curIndex;
  }

  // If not found, return false (this occurs when index was invalid)
  if (it.Done() || curIndex != index) {
    return false;
  }

  // Write the new element value, and drop the previous element value
  it.InsertElement(value);
  it.Skip();

  // Write the data to the database
  // Check status, since it needs to return true/false guarantee
  Status s = db_->Put(put_option_, key, it.WriteResult());

  // Success
  return s.ok();
}

/// Delete / Remove / Pop functions

// Trim (list: key) so that it will only contain the indices from start..stop
//  Invalid indices will not generate an error, just empty,
//  or the portion of the list that fits in this interval
//   : throws RedisListException
bool RedisLists::Trim(const std::string& key, int32_t start, int32_t stop) {
  // Get the original list data
  std::string data;
  db_->Get(get_option_, key, &data);

  // Handle negative indices in REDIS
  int listLen = Length(key);
  if (start < 0) {
    start = listLen - (-start);
  }
  if (stop < 0) {
    stop = listLen - (-stop);
  }

  // Truncate bounds to only fit in the list
  start = std::max(start, 0);
  stop = std::min(stop, listLen-1);

  // Construct an iterator for the list. Drop all undesired elements.
  int curIndex = 0;
  RedisListIterator it(data);
  it.Reserve(it.Size());          // Over-estimate
  while(!it.Done()) {
    // If not within the range, just skip the item (drop it).
    // Otherwise, continue as usual.
    if (start <= curIndex && curIndex <= stop) {
      it.Push();
    } else {
      it.Skip();
    }

    // Increment the current index
    ++curIndex;
  }

  // Write the (possibly empty) result to the database
  Status s = db_->Put(put_option_, key, it.WriteResult());

  // Return true as long as the write succeeded
  return s.ok();
}

// Return and remove the first element in the list (or "" if empty)
//   : throws RedisListException
bool RedisLists::PopLeft(const std::string& key, std::string* result) {
  // Get the original list data
  std::string data;
  db_->Get(get_option_, key, &data);

  // Point to first element in the list (if it exists), and get its value/size
  RedisListIterator it(data);
  if (it.Length() > 0) {            // Proceed only if list is non-empty
    Slice elem;
    it.GetCurrent(&elem);           // Store the value of the first element
    it.Reserve(it.Size() - it.SizeOf(elem));
    it.Skip();                      // DROP the first item and move to next

    // Update the db
    db_->Put(put_option_, key, it.WriteResult());

    // Return the value
    if (result != NULL) {
      *result = elem.ToString();
    }
    return true;
  } else {
    return false;
  }
}

// Remove and return the last element in the list (or "" if empty)
// TODO: Make this O(1). Might require MergeOperator.
//   : throws RedisListException
bool RedisLists::PopRight(const std::string& key, std::string* result) {
  // Extract the original list data
  std::string data;
  db_->Get(get_option_, key, &data);

  // Construct an iterator to the data and move to last element
  RedisListIterator it(data);
  it.Reserve(it.Size());
  int len = it.Length();
  int curIndex = 0;
  while(curIndex < (len-1) && !it.Done()) {
    it.Push();
    ++curIndex;
  }

  // Extract and drop/skip the last element
  if (curIndex == len-1) {
    assert(!it.Done());         // Sanity check. Should not have ended here.

    // Extract and pop the element
    Slice elem;
    it.GetCurrent(&elem);       // Save value of element.
    it.Skip();                  // Skip the element

    // Write the result to the database
    db_->Put(put_option_, key, it.WriteResult());

    // Return the value
    if (result != NULL) {
      *result = elem.ToString();
    }
    return true;
  } else {
    // Must have been an empty list
    assert(it.Done() && len==0 && curIndex == 0);
    return false;
  }
}

// Remove the (first or last) "num" occurrences of value in (list: key)
//   : throws RedisListException
int RedisLists::Remove(const std::string& key, int32_t num,
                       const std::string& value) {
  // Negative num ==> RemoveLast; Positive num ==> Remove First
  if (num < 0) {
    return RemoveLast(key, -num, value);
  } else if (num > 0) {
    return RemoveFirst(key, num, value);
  } else {
    return RemoveFirst(key, Length(key), value);
  }
}

// Remove the first "num" occurrences of value in (list: key).
//   : throws RedisListException
int RedisLists::RemoveFirst(const std::string& key, int32_t num,
                            const std::string& value) {
  // Ensure that the number is positive
  assert(num >= 0);

  // Extract the original list data
  std::string data;
  db_->Get(get_option_, key, &data);

  // Traverse the list, appending all but the desired occurrences of value
  int numSkipped = 0;         // Keep track of the number of times value is seen
  Slice elem;
  RedisListIterator it(data);
  it.Reserve(it.Size());
  while (!it.Done()) {
    it.GetCurrent(&elem);

    if (elem == value && numSkipped < num) {
      // Drop this item if desired
      it.Skip();
      ++numSkipped;
    } else {
      // Otherwise keep the item and proceed as normal
      it.Push();
    }
  }

  // Put the result back to the database
  db_->Put(put_option_, key, it.WriteResult());

  // Return the number of elements removed
  return numSkipped;
}


// Remove the last "num" occurrences of value in (list: key).
// TODO: I traverse the list 2x. Make faster. Might require MergeOperator.
//   : throws RedisListException
int RedisLists::RemoveLast(const std::string& key, int32_t num,
                           const std::string& value) {
  // Ensure that the number is positive
  assert(num >= 0);

  // Extract the original list data
  std::string data;
  db_->Get(get_option_, key, &data);

  // Temporary variable to hold the "current element" in the blocks below
  Slice elem;

  // Count the total number of occurrences of value
  int totalOccs = 0;
  for (RedisListIterator it(data); !it.Done(); it.Skip()) {
    it.GetCurrent(&elem);
    if (elem == value) {
      ++totalOccs;
    }
  }

  // Construct an iterator to the data. Reserve enough space for the result.
  RedisListIterator it(data);
  int bytesRemoved = std::min(num,totalOccs)*it.SizeOf(value);
  it.Reserve(it.Size() - bytesRemoved);

  // Traverse the list, appending all but the desired occurrences of value.
  // Note: "Drop the last k occurrences" is equivalent to
  //  "keep only the first n-k occurrences", where n is total occurrences.
  int numKept = 0;          // Keep track of the number of times value is kept
  while(!it.Done()) {
    it.GetCurrent(&elem);

    // If we are within the deletion range and equal to value, drop it.
    // Otherwise, append/keep/push it.
    if (elem == value) {
      if (numKept < totalOccs - num) {
        it.Push();
        ++numKept;
      } else {
        it.Skip();
      }
    } else {
      // Always append the others
      it.Push();
    }
  }

  // Put the result back to the database
  db_->Put(put_option_, key, it.WriteResult());

  // Return the number of elements removed
  return totalOccs - numKept;
}

/// Private functions

// Insert element value into (list: key), right before/after
//  the first occurrence of pivot
//   : throws RedisListException
int RedisLists::Insert(const std::string& key, const std::string& pivot,
                       const std::string& value, bool insert_after) {
  // Get the original list data
  std::string data;
  db_->Get(get_option_, key, &data);

  // Construct an iterator to the data and reserve enough space for result.
  RedisListIterator it(data);
  it.Reserve(it.Size() + it.SizeOf(value));

  // Iterate through the list until we find the element we want
  Slice elem;
  bool found = false;
  while(!it.Done() && !found) {
    it.GetCurrent(&elem);

    // When we find the element, insert the element and mark found
    if (elem == pivot) {                // Found it!
      found = true;
      if (insert_after == true) {       // Skip one more, if inserting after it
        it.Push();
      }
      it.InsertElement(value);
    } else {
      it.Push();
    }

  }

  // Put the data (string) into the database
  if (found) {
    db_->Put(put_option_, key, it.WriteResult());
  }

  // Returns the new (possibly unchanged) length of the list
  return it.Length();
}

}  // namespace rocksdb
#endif  // ROCKSDB_LITE
