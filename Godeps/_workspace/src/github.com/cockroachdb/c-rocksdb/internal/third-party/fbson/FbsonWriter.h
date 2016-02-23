/*
 *  Copyright (c) 2014, Facebook, Inc.
 *  All rights reserved.
 *
 *  This source code is licensed under the BSD-style license found in the
 *  LICENSE file in the root directory of this source tree. An additional grant
 *  of patent rights can be found in the PATENTS file in the same directory.
 *
 */

/*
 * This file defines FbsonWriterT (template) and FbsonWriter.
 *
 * FbsonWriterT is a template class which implements an FBSON serializer.
 * Users call various write functions of FbsonWriterT object to write values
 * directly to FBSON packed bytes. All write functions of value or key return
 * the number of bytes written to FBSON, or 0 if there is an error. To write an
 * object, an array, or a string, you must call writeStart[..] before writing
 * values or key, and call writeEnd[..] after finishing at the end.
 *
 * By default, an FbsonWriterT object creates an output stream buffer.
 * Alternatively, you can also pass any output stream object to a writer, as
 * long as the stream object implements some basic functions of std::ostream
 * (such as FbsonOutStream, see FbsonStream.h).
 *
 * FbsonWriter specializes FbsonWriterT with FbsonOutStream type (see
 * FbsonStream.h). So unless you want to provide own a different output stream
 * type, use FbsonParser object.
 *
 * @author Tian Xia <tianx@fb.com>
 */

#ifndef FBSON_FBSONWRITER_H
#define FBSON_FBSONWRITER_H

#include <stack>
#include "FbsonDocument.h"
#include "FbsonStream.h"

namespace fbson {

template <class OS_TYPE>
class FbsonWriterT {
 public:
  FbsonWriterT()
      : alloc_(true), hasHdr_(false), kvState_(WS_Value), str_pos_(0) {
    os_ = new OS_TYPE();
  }

  explicit FbsonWriterT(OS_TYPE& os)
      : os_(&os),
        alloc_(false),
        hasHdr_(false),
        kvState_(WS_Value),
        str_pos_(0) {}

  ~FbsonWriterT() {
    if (alloc_) {
      delete os_;
    }
  }

  void reset() {
    os_->clear();
    os_->seekp(0);
    hasHdr_ = false;
    kvState_ = WS_Value;
    for (; !stack_.empty(); stack_.pop())
      ;
  }

  // write a key string (or key id if an external dict is provided)
  uint32_t writeKey(const char* key,
                    uint8_t len,
                    hDictInsert handler = nullptr) {
    if (len && !stack_.empty() && verifyKeyState()) {
      int key_id = -1;
      if (handler) {
        key_id = handler(key, len);
      }

      uint32_t size = sizeof(uint8_t);
      if (key_id < 0) {
        os_->put(len);
        os_->write(key, len);
        size += len;
      } else if (key_id <= FbsonKeyValue::sMaxKeyId) {
        FbsonKeyValue::keyid_type idx = key_id;
        os_->put(0);
        os_->write((char*)&idx, sizeof(FbsonKeyValue::keyid_type));
        size += sizeof(FbsonKeyValue::keyid_type);
      } else { // key id overflow
        assert(0);
        return 0;
      }

      kvState_ = WS_Key;
      return size;
    }

    return 0;
  }

  // write a key id
  uint32_t writeKey(FbsonKeyValue::keyid_type idx) {
    if (!stack_.empty() && verifyKeyState()) {
      os_->put(0);
      os_->write((char*)&idx, sizeof(FbsonKeyValue::keyid_type));
      kvState_ = WS_Key;
      return sizeof(uint8_t) + sizeof(FbsonKeyValue::keyid_type);
    }

    return 0;
  }

  uint32_t writeNull() {
    if (!stack_.empty() && verifyValueState()) {
      os_->put((FbsonTypeUnder)FbsonType::T_Null);
      kvState_ = WS_Value;
      return sizeof(FbsonValue);
    }

    return 0;
  }

  uint32_t writeBool(bool b) {
    if (!stack_.empty() && verifyValueState()) {
      if (b) {
        os_->put((FbsonTypeUnder)FbsonType::T_True);
      } else {
        os_->put((FbsonTypeUnder)FbsonType::T_False);
      }

      kvState_ = WS_Value;
      return sizeof(FbsonValue);
    }

    return 0;
  }

  uint32_t writeInt8(int8_t v) {
    if (!stack_.empty() && verifyValueState()) {
      os_->put((FbsonTypeUnder)FbsonType::T_Int8);
      os_->put(v);
      kvState_ = WS_Value;
      return sizeof(Int8Val);
    }

    return 0;
  }

  uint32_t writeInt16(int16_t v) {
    if (!stack_.empty() && verifyValueState()) {
      os_->put((FbsonTypeUnder)FbsonType::T_Int16);
      os_->write((char*)&v, sizeof(int16_t));
      kvState_ = WS_Value;
      return sizeof(Int16Val);
    }

    return 0;
  }

  uint32_t writeInt32(int32_t v) {
    if (!stack_.empty() && verifyValueState()) {
      os_->put((FbsonTypeUnder)FbsonType::T_Int32);
      os_->write((char*)&v, sizeof(int32_t));
      kvState_ = WS_Value;
      return sizeof(Int32Val);
    }

    return 0;
  }

  uint32_t writeInt64(int64_t v) {
    if (!stack_.empty() && verifyValueState()) {
      os_->put((FbsonTypeUnder)FbsonType::T_Int64);
      os_->write((char*)&v, sizeof(int64_t));
      kvState_ = WS_Value;
      return sizeof(Int64Val);
    }

    return 0;
  }

  uint32_t writeDouble(double v) {
    if (!stack_.empty() && verifyValueState()) {
      os_->put((FbsonTypeUnder)FbsonType::T_Double);
      os_->write((char*)&v, sizeof(double));
      kvState_ = WS_Value;
      return sizeof(DoubleVal);
    }

    return 0;
  }

  // must call writeStartString before writing a string val
  bool writeStartString() {
    if (!stack_.empty() && verifyValueState()) {
      os_->put((FbsonTypeUnder)FbsonType::T_String);
      str_pos_ = os_->tellp();

      // fill the size bytes with 0 for now
      uint32_t size = 0;
      os_->write((char*)&size, sizeof(uint32_t));

      kvState_ = WS_String;
      return true;
    }

    return false;
  }

  // finish writing a string val
  bool writeEndString() {
    if (kvState_ == WS_String) {
      std::streampos cur_pos = os_->tellp();
      int32_t size = (int32_t)(cur_pos - str_pos_ - sizeof(uint32_t));
      assert(size >= 0);

      os_->seekp(str_pos_);
      os_->write((char*)&size, sizeof(uint32_t));
      os_->seekp(cur_pos);

      kvState_ = WS_Value;
      return true;
    }

    return false;
  }

  uint32_t writeString(const char* str, uint32_t len) {
    if (kvState_ == WS_String) {
      os_->write(str, len);
      return len;
    }

    return 0;
  }

  uint32_t writeString(char ch) {
    if (kvState_ == WS_String) {
      os_->put(ch);
      return 1;
    }

    return 0;
  }

  // must call writeStartBinary before writing a binary val
  bool writeStartBinary() {
    if (!stack_.empty() && verifyValueState()) {
      os_->put((FbsonTypeUnder)FbsonType::T_Binary);
      str_pos_ = os_->tellp();

      // fill the size bytes with 0 for now
      uint32_t size = 0;
      os_->write((char*)&size, sizeof(uint32_t));

      kvState_ = WS_Binary;
      return true;
    }

    return false;
  }

  // finish writing a binary val
  bool writeEndBinary() {
    if (kvState_ == WS_Binary) {
      std::streampos cur_pos = os_->tellp();
      int32_t size = (int32_t)(cur_pos - str_pos_ - sizeof(uint32_t));
      assert(size >= 0);

      os_->seekp(str_pos_);
      os_->write((char*)&size, sizeof(uint32_t));
      os_->seekp(cur_pos);

      kvState_ = WS_Value;
      return true;
    }

    return false;
  }

  uint32_t writeBinary(const char* bin, uint32_t len) {
    if (kvState_ == WS_Binary) {
      os_->write(bin, len);
      return len;
    }

    return 0;
  }

  // must call writeStartObject before writing an object val
  bool writeStartObject() {
    if (stack_.empty() || verifyValueState()) {
      if (stack_.empty()) {
        // if this is a new FBSON, write the header
        if (!hasHdr_) {
          writeHeader();
        } else
          return false;
      }

      os_->put((FbsonTypeUnder)FbsonType::T_Object);
      // save the size position
      stack_.push(WriteInfo({WS_Object, os_->tellp()}));

      // fill the size bytes with 0 for now
      uint32_t size = 0;
      os_->write((char*)&size, sizeof(uint32_t));

      kvState_ = WS_Value;
      return true;
    }

    return false;
  }

  // finish writing an object val
  bool writeEndObject() {
    if (!stack_.empty() && stack_.top().state == WS_Object &&
        kvState_ == WS_Value) {
      WriteInfo& ci = stack_.top();
      std::streampos cur_pos = os_->tellp();
      int32_t size = (int32_t)(cur_pos - ci.sz_pos - sizeof(uint32_t));
      assert(size >= 0);

      os_->seekp(ci.sz_pos);
      os_->write((char*)&size, sizeof(uint32_t));
      os_->seekp(cur_pos);
      stack_.pop();

      return true;
    }

    return false;
  }

  // must call writeStartArray before writing an array val
  bool writeStartArray() {
    if (stack_.empty() || verifyValueState()) {
      if (stack_.empty()) {
        // if this is a new FBSON, write the header
        if (!hasHdr_) {
          writeHeader();
        } else
          return false;
      }

      os_->put((FbsonTypeUnder)FbsonType::T_Array);
      // save the size position
      stack_.push(WriteInfo({WS_Array, os_->tellp()}));

      // fill the size bytes with 0 for now
      uint32_t size = 0;
      os_->write((char*)&size, sizeof(uint32_t));

      kvState_ = WS_Value;
      return true;
    }

    return false;
  }

  // finish writing an array val
  bool writeEndArray() {
    if (!stack_.empty() && stack_.top().state == WS_Array &&
        kvState_ == WS_Value) {
      WriteInfo& ci = stack_.top();
      std::streampos cur_pos = os_->tellp();
      int32_t size = (int32_t)(cur_pos - ci.sz_pos - sizeof(uint32_t));
      assert(size >= 0);

      os_->seekp(ci.sz_pos);
      os_->write((char*)&size, sizeof(uint32_t));
      os_->seekp(cur_pos);
      stack_.pop();

      return true;
    }

    return false;
  }

  OS_TYPE* getOutput() { return os_; }

 private:
  // verify we are in the right state before writing a value
  bool verifyValueState() {
    assert(!stack_.empty());
    return (stack_.top().state == WS_Object && kvState_ == WS_Key) ||
           (stack_.top().state == WS_Array && kvState_ == WS_Value);
  }

  // verify we are in the right state before writing a key
  bool verifyKeyState() {
    assert(!stack_.empty());
    return stack_.top().state == WS_Object && kvState_ == WS_Value;
  }

  void writeHeader() {
    os_->put(FBSON_VER);
    hasHdr_ = true;
  }

 private:
  enum WriteState {
    WS_NONE,
    WS_Array,
    WS_Object,
    WS_Key,
    WS_Value,
    WS_String,
    WS_Binary,
  };

  struct WriteInfo {
    WriteState state;
    std::streampos sz_pos;
  };

 private:
  OS_TYPE* os_;
  bool alloc_;
  bool hasHdr_;
  WriteState kvState_; // key or value state
  std::streampos str_pos_;
  std::stack<WriteInfo> stack_;
};

typedef FbsonWriterT<FbsonOutStream> FbsonWriter;

} // namespace fbson

#endif // FBSON_FBSONWRITER_H
