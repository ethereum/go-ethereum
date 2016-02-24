// Copyright 2013 Facebook
/**
 * RedisListIterator:
 * An abstraction over the "list" concept (e.g.: for redis lists).
 * Provides functionality to read, traverse, edit, and write these lists.
 *
 * Upon construction, the RedisListIterator is given a block of list data.
 * Internally, it stores a pointer to the data and a pointer to current item.
 * It also stores a "result" list that will be mutated over time.
 *
 * Traversal and mutation are done by "forward iteration".
 * The Push() and Skip() methods will advance the iterator to the next item.
 * However, Push() will also "write the current item to the result".
 * Skip() will simply move to next item, causing current item to be dropped.
 *
 * Upon completion, the result (accessible by WriteResult()) will be saved.
 * All "skipped" items will be gone; all "pushed" items will remain.
 *
 * @throws Any of the operations may throw a RedisListException if an invalid
 *          operation is performed or if the data is found to be corrupt.
 *
 * @notes By default, if WriteResult() is called part-way through iteration,
 *        it will automatically advance the iterator to the end, and Keep()
 *        all items that haven't been traversed yet. This may be subject
 *        to review.
 *
 * @notes Can access the "current" item via GetCurrent(), and other
 *        list-specific information such as Length().
 *
 * @notes The internal representation is due to change at any time. Presently,
 *        the list is represented as follows:
 *          - 32-bit integer header: the number of items in the list
 *          - For each item:
 *              - 32-bit int (n): the number of bytes representing this item
 *              - n bytes of data: the actual data.
 *
 * @author Deon Nicholas (dnicholas@fb.com)
 */

#ifndef ROCKSDB_LITE
#pragma once

#include <string>

#include "redis_list_exception.h"
#include "rocksdb/slice.h"
#include "util/coding.h"

namespace rocksdb {

/// An abstraction over the "list" concept.
/// All operations may throw a RedisListException
class RedisListIterator {
 public:
  /// Construct a redis-list-iterator based on data.
  /// If the data is non-empty, it must formatted according to @notes above.
  ///
  /// If the data is valid, we can assume the following invariant(s):
  ///  a) length_, num_bytes_ are set correctly.
  ///  b) cur_byte_ always refers to the start of the current element,
  ///       just before the bytes that specify element length.
  ///  c) cur_elem_ is always the index of the current element.
  ///  d) cur_elem_length_ is always the number of bytes in current element,
  ///       excluding the 4-byte header itself.
  ///  e) result_ will always contain data_[0..cur_byte_) and a header
  ///  f) Whenever corrupt data is encountered or an invalid operation is
  ///      attempted, a RedisListException will immediately be thrown.
  RedisListIterator(const std::string& list_data)
      : data_(list_data.data()),
        num_bytes_(static_cast<uint32_t>(list_data.size())),
        cur_byte_(0),
        cur_elem_(0),
        cur_elem_length_(0),
        length_(0),
        result_() {

    // Initialize the result_ (reserve enough space for header)
    InitializeResult();

    // Parse the data only if it is not empty.
    if (num_bytes_ == 0) {
      return;
    }

    // If non-empty, but less than 4 bytes, data must be corrupt
    if (num_bytes_ < sizeof(length_)) {
      ThrowError("Corrupt header.");    // Will break control flow
    }

    // Good. The first bytes specify the number of elements
    length_ = DecodeFixed32(data_);
    cur_byte_ = sizeof(length_);

    // If we have at least one element, point to that element.
    // Also, read the first integer of the element (specifying the size),
    //   if possible.
    if (length_ > 0) {
      if (cur_byte_ + sizeof(cur_elem_length_) <= num_bytes_) {
        cur_elem_length_ = DecodeFixed32(data_+cur_byte_);
      } else {
        ThrowError("Corrupt data for first element.");
      }
    }

    // At this point, we are fully set-up.
    // The invariants described in the header should now be true.
  }

  /// Reserve some space for the result_.
  /// Equivalent to result_.reserve(bytes).
  void Reserve(int bytes) {
    result_.reserve(bytes);
  }

  /// Go to next element in data file.
  /// Also writes the current element to result_.
  RedisListIterator& Push() {
    WriteCurrentElement();
    MoveNext();
    return *this;
  }

  /// Go to next element in data file.
  /// Drops/skips the current element. It will not be written to result_.
  RedisListIterator& Skip() {
    MoveNext();
    --length_;          // One less item
    --cur_elem_;        // We moved one forward, but index did not change
    return *this;
  }

  /// Insert elem into the result_ (just BEFORE the current element / byte)
  /// Note: if Done() (i.e.: iterator points to end), this will append elem.
  void InsertElement(const Slice& elem) {
    // Ensure we are in a valid state
    CheckErrors();

    const int kOrigSize = static_cast<int>(result_.size());
    result_.resize(kOrigSize + SizeOf(elem));
    EncodeFixed32(result_.data() + kOrigSize,
                  static_cast<uint32_t>(elem.size()));
    memcpy(result_.data() + kOrigSize + sizeof(uint32_t), elem.data(),
           elem.size());
    ++length_;
    ++cur_elem_;
  }

  /// Access the current element, and save the result into *curElem
  void GetCurrent(Slice* curElem) {
    // Ensure we are in a valid state
    CheckErrors();

    // Ensure that we are not past the last element.
    if (Done()) {
      ThrowError("Invalid dereferencing.");
    }

    // Dereference the element
    *curElem = Slice(data_+cur_byte_+sizeof(cur_elem_length_),
                     cur_elem_length_);
  }

  // Number of elements
  int Length() const {
    return length_;
  }

  // Number of bytes in the final representation (i.e: WriteResult().size())
  int Size() const {
    // result_ holds the currently written data
    // data_[cur_byte..num_bytes-1] is the remainder of the data
    return static_cast<int>(result_.size() + (num_bytes_ - cur_byte_));
  }

  // Reached the end?
  bool Done() const {
    return cur_byte_ >= num_bytes_ || cur_elem_ >= length_;
  }

  /// Returns a string representing the final, edited, data.
  /// Assumes that all bytes of data_ in the range [0,cur_byte_) have been read
  ///  and that result_ contains this data.
  /// The rest of the data must still be written.
  /// So, this method ADVANCES THE ITERATOR TO THE END before writing.
  Slice WriteResult() {
    CheckErrors();

    // The header should currently be filled with dummy data (0's)
    // Correctly update the header.
    // Note, this is safe since result_ is a vector (guaranteed contiguous)
    EncodeFixed32(&result_[0],length_);

    // Append the remainder of the data to the result.
    result_.insert(result_.end(),data_+cur_byte_, data_ +num_bytes_);

    // Seek to end of file
    cur_byte_ = num_bytes_;
    cur_elem_ = length_;
    cur_elem_length_ = 0;

    // Return the result
    return Slice(result_.data(),result_.size());
  }

 public: // Static public functions

  /// An upper-bound on the amount of bytes needed to store this element.
  /// This is used to hide representation information from the client.
  /// E.G. This can be used to compute the bytes we want to Reserve().
  static uint32_t SizeOf(const Slice& elem) {
    // [Integer Length . Data]
    return static_cast<uint32_t>(sizeof(uint32_t) + elem.size());
  }

 private: // Private functions

  /// Initializes the result_ string.
  /// It will fill the first few bytes with 0's so that there is
  ///  enough space for header information when we need to write later.
  /// Currently, "header information" means: the length (number of elements)
  /// Assumes that result_ is empty to begin with
  void InitializeResult() {
    assert(result_.empty());            // Should always be true.
    result_.resize(sizeof(uint32_t),0); // Put a block of 0's as the header
  }

  /// Go to the next element (used in Push() and Skip())
  void MoveNext() {
    CheckErrors();

    // Check to make sure we are not already in a finished state
    if (Done()) {
      ThrowError("Attempting to iterate past end of list.");
    }

    // Move forward one element.
    cur_byte_ += sizeof(cur_elem_length_) + cur_elem_length_;
    ++cur_elem_;

    // If we are at the end, finish
    if (Done()) {
      cur_elem_length_ = 0;
      return;
    }

    // Otherwise, we should be able to read the new element's length
    if (cur_byte_ + sizeof(cur_elem_length_) > num_bytes_) {
      ThrowError("Corrupt element data.");
    }

    // Set the new element's length
    cur_elem_length_ = DecodeFixed32(data_+cur_byte_);

    return;
  }

  /// Append the current element (pointed to by cur_byte_) to result_
  /// Assumes result_ has already been reserved appropriately.
  void WriteCurrentElement() {
    // First verify that the iterator is still valid.
    CheckErrors();
    if (Done()) {
      ThrowError("Attempting to write invalid element.");
    }

    // Append the cur element.
    result_.insert(result_.end(),
                   data_+cur_byte_,
                   data_+cur_byte_+ sizeof(uint32_t) + cur_elem_length_);
  }

  /// Will ThrowError() if neccessary.
  /// Checks for common/ubiquitous errors that can arise after most operations.
  /// This method should be called before any reading operation.
  /// If this function succeeds, then we are guaranteed to be in a valid state.
  /// Other member functions should check for errors and ThrowError() also
  ///  if an error occurs that is specific to it even while in a valid state.
  void CheckErrors() {
    // Check if any crazy thing has happened recently
    if ((cur_elem_ > length_) ||                              // Bad index
        (cur_byte_ > num_bytes_) ||                           // No more bytes
        (cur_byte_ + cur_elem_length_ > num_bytes_) ||        // Item too large
        (cur_byte_ == num_bytes_ && cur_elem_ != length_) ||  // Too many items
        (cur_elem_ == length_ && cur_byte_ != num_bytes_)) {  // Too many bytes
      ThrowError("Corrupt data.");
    }
  }

  /// Will throw an exception based on the passed-in message.
  /// This function is guaranteed to STOP THE CONTROL-FLOW.
  /// (i.e.: you do not have to call "return" after calling ThrowError)
  void ThrowError(const char* const msg = NULL) {
    // TODO: For now we ignore the msg parameter. This can be expanded later.
    throw RedisListException();
  }

 private:
  const char* const data_;      // A pointer to the data (the first byte)
  const uint32_t num_bytes_;    // The number of bytes in this list

  uint32_t cur_byte_;           // The current byte being read
  uint32_t cur_elem_;           // The current element being read
  uint32_t cur_elem_length_;    // The number of bytes in current element

  uint32_t length_;             // The number of elements in this list
  std::vector<char> result_;    // The output data
};

} // namespace rocksdb
#endif  // ROCKSDB_LITE
