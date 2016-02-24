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
 * This header file defines miscellaneous utility classes.
 *
 * @author Tian Xia <tianx@fb.com>
 */

#ifndef FBSON_FBSONUTIL_H
#define FBSON_FBSONUTIL_H

#include <sstream>
#include "FbsonDocument.h"

namespace fbson {

#define OUT_BUF_SIZE 1024

/*
 * FbsonToJson converts an FbsonValue object to a JSON string.
 */
class FbsonToJson {
 public:
  FbsonToJson() : os_(buffer_, OUT_BUF_SIZE) {}

  // get json string
  const char* json(const FbsonValue* pval) {
    os_.clear();
    os_.seekp(0);

    if (pval) {
      intern_json(pval);
    }

    os_.put(0);
    return os_.getBuffer();
  }

 private:
  // recursively convert FbsonValue
  void intern_json(const FbsonValue* val) {
    switch (val->type()) {
    case FbsonType::T_Null: {
      os_.write("null", 4);
      break;
    }
    case FbsonType::T_True: {
      os_.write("true", 4);
      break;
    }
    case FbsonType::T_False: {
      os_.write("false", 5);
      break;
    }
    case FbsonType::T_Int8: {
      os_.write(((Int8Val*)val)->val());
      break;
    }
    case FbsonType::T_Int16: {
      os_.write(((Int16Val*)val)->val());
      break;
    }
    case FbsonType::T_Int32: {
      os_.write(((Int32Val*)val)->val());
      break;
    }
    case FbsonType::T_Int64: {
      os_.write(((Int64Val*)val)->val());
      break;
    }
    case FbsonType::T_Double: {
      os_.write(((DoubleVal*)val)->val());
      break;
    }
    case FbsonType::T_String: {
      os_.put('"');
      os_.write(((StringVal*)val)->getBlob(), ((StringVal*)val)->getBlobLen());
      os_.put('"');
      break;
    }
    case FbsonType::T_Binary: {
      os_.write("\"<BINARY>", 9);
      os_.write(((BinaryVal*)val)->getBlob(), ((BinaryVal*)val)->getBlobLen());
      os_.write("<BINARY>\"", 9);
      break;
    }
    case FbsonType::T_Object: {
      object_to_json((ObjectVal*)val);
      break;
    }
    case FbsonType::T_Array: {
      array_to_json((ArrayVal*)val);
      break;
    }
    default:
      break;
    }
  }

  // convert object
  void object_to_json(const ObjectVal* val) {
    os_.put('{');

    auto iter = val->begin();
    auto iter_fence = val->end();

    while (iter < iter_fence) {
      // write key
      if (iter->klen()) {
        os_.put('"');
        os_.write(iter->getKeyStr(), iter->klen());
        os_.put('"');
      } else {
        os_.write(iter->getKeyId());
      }
      os_.put(':');

      // convert value
      intern_json(iter->value());

      ++iter;
      if (iter != iter_fence) {
        os_.put(',');
      }
    }

    assert(iter == iter_fence);

    os_.put('}');
  }

  // convert array to json
  void array_to_json(const ArrayVal* val) {
    os_.put('[');

    auto iter = val->begin();
    auto iter_fence = val->end();

    while (iter != iter_fence) {
      // convert value
      intern_json((const FbsonValue*)iter);
      ++iter;
      if (iter != iter_fence) {
        os_.put(',');
      }
    }

    assert(iter == iter_fence);

    os_.put(']');
  }

 private:
  FbsonOutStream os_;
  char buffer_[OUT_BUF_SIZE];
};

} // namespace fbson

#endif // FBSON_FBSONUTIL_H
