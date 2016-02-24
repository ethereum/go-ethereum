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
 * This file defines FbsonJsonParserT (template) and FbsonJsonParser.
 *
 * FbsonJsonParserT is a template class which implements a JSON parser.
 * FbsonJsonParserT parses JSON text, and serialize it to FBSON binary format
 * by using FbsonWriterT object. By default, FbsonJsonParserT creates a new
 * FbsonWriterT object with an output stream object.  However, you can also
 * pass in your FbsonWriterT or any stream object that implements some basic
 * interface of std::ostream (see FbsonStream.h).
 *
 * FbsonJsonParser specializes FbsonJsonParserT with FbsonOutStream type (see
 * FbsonStream.h). So unless you want to provide own a different output stream
 * type, use FbsonJsonParser object.
 *
 * ** Parsing JSON **
 * FbsonJsonParserT parses JSON string, and directly serializes into FBSON
 * packed bytes. There are three ways to parse a JSON string: (1) using
 * c-string, (2) using string with len, (3) using std::istream object. You can
 * use custome streambuf to redirect output. FbsonOutBuffer is a streambuf used
 * internally if the input is raw character buffer.
 *
 * You can reuse an FbsonJsonParserT object to parse/serialize multiple JSON
 * strings, and the previous FBSON will be overwritten.
 *
 * If parsing fails (returned false), the error code will be set to one of
 * FbsonErrType, and can be retrieved by calling getErrorCode().
 *
 * ** External dictionary **
 * During parsing a JSON string, you can pass a call-back function to map a key
 * string to an id, and store the dictionary id in FBSON to save space. The
 * purpose of using an external dictionary is more towards a collection of
 * documents (which has common keys) rather than a single document, so that
 * space saving will be siginificant.
 *
 * ** Endianness **
 * Note: FBSON serialization doesn't assume endianness of the server. However
 * you will need to ensure that the endianness at the reader side is the same
 * as that at the writer side (if they are on different machines). Otherwise,
 * proper conversion is needed when a number value is returned to the
 * caller/writer.
 *
 * @author Tian Xia <tianx@fb.com>
 */

#ifndef FBSON_FBSONPARSER_H
#define FBSON_FBSONPARSER_H

#include <cmath>
#include <limits>
#include "FbsonDocument.h"
#include "FbsonWriter.h"

namespace fbson {

const char* const kJsonDelim = " ,]}\t\r\n";
const char* const kWhiteSpace = " \t\n\r";

/*
 * Error codes
 */
enum class FbsonErrType {
  E_NONE = 0,
  E_INVALID_VER,
  E_EMPTY_STR,
  E_OUTPUT_FAIL,
  E_INVALID_DOCU,
  E_INVALID_VALUE,
  E_INVALID_KEY,
  E_INVALID_STR,
  E_INVALID_OBJ,
  E_INVALID_ARR,
  E_INVALID_HEX,
  E_INVALID_OCTAL,
  E_INVALID_DECIMAL,
  E_INVALID_EXPONENT,
  E_HEX_OVERFLOW,
  E_OCTAL_OVERFLOW,
  E_DECIMAL_OVERFLOW,
  E_DOUBLE_OVERFLOW,
  E_EXPONENT_OVERFLOW,
};

/*
 * Template FbsonJsonParserT
 */
template <class OS_TYPE>
class FbsonJsonParserT {
 public:
  FbsonJsonParserT() : err_(FbsonErrType::E_NONE) {}

  explicit FbsonJsonParserT(OS_TYPE& os)
      : writer_(os), err_(FbsonErrType::E_NONE) {}

  // parse a UTF-8 JSON string
  bool parse(const std::string& str, hDictInsert handler = nullptr) {
    return parse(str.c_str(), (unsigned int)str.size(), handler);
  }

  // parse a UTF-8 JSON c-style string (NULL terminated)
  bool parse(const char* c_str, hDictInsert handler = nullptr) {
    return parse(c_str, (unsigned int)strlen(c_str), handler);
  }

  // parse a UTF-8 JSON string with length
  bool parse(const char* pch, unsigned int len, hDictInsert handler = nullptr) {
    if (!pch || len == 0) {
      err_ = FbsonErrType::E_EMPTY_STR;
      return false;
    }

    FbsonInBuffer sb(pch, len);
    std::istream in(&sb);
    return parse(in, handler);
  }

  // parse UTF-8 JSON text from an input stream
  bool parse(std::istream& in, hDictInsert handler = nullptr) {
    bool res = false;

    // reset output stream
    writer_.reset();

    trim(in);

    if (in.peek() == '{') {
      in.ignore();
      res = parseObject(in, handler);
    } else if (in.peek() == '[') {
      in.ignore();
      res = parseArray(in, handler);
    } else {
      err_ = FbsonErrType::E_INVALID_DOCU;
    }

    trim(in);
    if (res && !in.eof()) {
      err_ = FbsonErrType::E_INVALID_DOCU;
      return false;
    }

    return res;
  }

  FbsonWriterT<OS_TYPE>& getWriter() { return writer_; }

  FbsonErrType getErrorCode() { return err_; }

  // clear error code
  void clearErr() { err_ = FbsonErrType::E_NONE; }

 private:
  // parse a JSON object (comma-separated list of key-value pairs)
  bool parseObject(std::istream& in, hDictInsert handler) {
    if (!writer_.writeStartObject()) {
      err_ = FbsonErrType::E_OUTPUT_FAIL;
      return false;
    }

    trim(in);

    if (in.peek() == '}') {
      in.ignore();
      // empty object
      if (!writer_.writeEndObject()) {
        err_ = FbsonErrType::E_OUTPUT_FAIL;
        return false;
      }
      return true;
    }

    while (in.good()) {
      if (in.get() != '"') {
        err_ = FbsonErrType::E_INVALID_KEY;
        return false;
      }

      if (!parseKVPair(in, handler)) {
        return false;
      }

      trim(in);

      char ch = in.get();
      if (ch == '}') {
        // end of the object
        if (!writer_.writeEndObject()) {
          err_ = FbsonErrType::E_OUTPUT_FAIL;
          return false;
        }
        return true;
      } else if (ch != ',') {
        err_ = FbsonErrType::E_INVALID_OBJ;
        return false;
      }

      trim(in);
    }

    err_ = FbsonErrType::E_INVALID_OBJ;
    return false;
  }

  // parse a JSON array (comma-separated list of values)
  bool parseArray(std::istream& in, hDictInsert handler) {
    if (!writer_.writeStartArray()) {
      err_ = FbsonErrType::E_OUTPUT_FAIL;
      return false;
    }

    trim(in);

    if (in.peek() == ']') {
      in.ignore();
      // empty array
      if (!writer_.writeEndArray()) {
        err_ = FbsonErrType::E_OUTPUT_FAIL;
        return false;
      }
      return true;
    }

    while (in.good()) {
      if (!parseValue(in, handler)) {
        return false;
      }

      trim(in);

      char ch = in.get();
      if (ch == ']') {
        // end of the array
        if (!writer_.writeEndArray()) {
          err_ = FbsonErrType::E_OUTPUT_FAIL;
          return false;
        }
        return true;
      } else if (ch != ',') {
        err_ = FbsonErrType::E_INVALID_ARR;
        return false;
      }

      trim(in);
    }

    err_ = FbsonErrType::E_INVALID_ARR;
    return false;
  }

  // parse a key-value pair, separated by ":"
  bool parseKVPair(std::istream& in, hDictInsert handler) {
    if (parseKey(in, handler) && parseValue(in, handler)) {
      return true;
    }

    return false;
  }

  // parse a key (must be string)
  bool parseKey(std::istream& in, hDictInsert handler) {
    char key[FbsonKeyValue::sMaxKeyLen];
    int i = 0;
    while (in.good() && in.peek() != '"' && i < FbsonKeyValue::sMaxKeyLen) {
      key[i++] = in.get();
    }

    if (!in.good() || in.peek() != '"' || i == 0) {
      err_ = FbsonErrType::E_INVALID_KEY;
      return false;
    }

    in.ignore(); // discard '"'

    int key_id = -1;
    if (handler) {
      key_id = handler(key, i);
    }

    if (key_id < 0) {
      writer_.writeKey(key, i);
    } else {
      writer_.writeKey(key_id);
    }

    trim(in);

    if (in.get() != ':') {
      err_ = FbsonErrType::E_INVALID_OBJ;
      return false;
    }

    return true;
  }

  // parse a value
  bool parseValue(std::istream& in, hDictInsert handler) {
    bool res = false;

    trim(in);

    switch (in.peek()) {
    case 'N':
    case 'n': {
      in.ignore();
      res = parseNull(in);
      break;
    }
    case 'T':
    case 't': {
      in.ignore();
      res = parseTrue(in);
      break;
    }
    case 'F':
    case 'f': {
      in.ignore();
      res = parseFalse(in);
      break;
    }
    case '"': {
      in.ignore();
      res = parseString(in);
      break;
    }
    case '{': {
      in.ignore();
      res = parseObject(in, handler);
      break;
    }
    case '[': {
      in.ignore();
      res = parseArray(in, handler);
      break;
    }
    default: {
      res = parseNumber(in);
      break;
    }
    }

    return res;
  }

  // parse NULL value
  bool parseNull(std::istream& in) {
    if (tolower(in.get()) == 'u' && tolower(in.get()) == 'l' &&
        tolower(in.get()) == 'l') {
      writer_.writeNull();
      return true;
    }

    err_ = FbsonErrType::E_INVALID_VALUE;
    return false;
  }

  // parse TRUE value
  bool parseTrue(std::istream& in) {
    if (tolower(in.get()) == 'r' && tolower(in.get()) == 'u' &&
        tolower(in.get()) == 'e') {
      writer_.writeBool(true);
      return true;
    }

    err_ = FbsonErrType::E_INVALID_VALUE;
    return false;
  }

  // parse FALSE value
  bool parseFalse(std::istream& in) {
    if (tolower(in.get()) == 'a' && tolower(in.get()) == 'l' &&
        tolower(in.get()) == 's' && tolower(in.get()) == 'e') {
      writer_.writeBool(false);
      return true;
    }

    err_ = FbsonErrType::E_INVALID_VALUE;
    return false;
  }

  // parse a string
  bool parseString(std::istream& in) {
    if (!writer_.writeStartString()) {
      err_ = FbsonErrType::E_OUTPUT_FAIL;
      return false;
    }

    bool escaped = false;
    char buffer[4096]; // write 4KB at a time
    int nread = 0;
    while (in.good()) {
      char ch = in.get();
      if (ch != '"' || escaped) {
        buffer[nread++] = ch;
        if (nread == 4096) {
          // flush buffer
          if (!writer_.writeString(buffer, nread)) {
            err_ = FbsonErrType::E_OUTPUT_FAIL;
            return false;
          }
          nread = 0;
        }
        // set/reset escape
        if (ch == '\\' || escaped) {
          escaped = !escaped;
        }
      } else {
        // write all remaining bytes in the buffer
        if (nread > 0) {
          if (!writer_.writeString(buffer, nread)) {
            err_ = FbsonErrType::E_OUTPUT_FAIL;
            return false;
          }
        }
        // end writing string
        if (!writer_.writeEndString()) {
          err_ = FbsonErrType::E_OUTPUT_FAIL;
          return false;
        }
        return true;
      }
    }

    err_ = FbsonErrType::E_INVALID_STR;
    return false;
  }

  // parse a number
  // Number format can be hex, octal, or decimal (including float).
  // Only decimal can have (+/-) sign prefix.
  bool parseNumber(std::istream& in) {
    bool ret = false;
    switch (in.peek()) {
    case '0': {
      in.ignore();

      if (in.peek() == 'x' || in.peek() == 'X') {
        in.ignore();
        ret = parseHex(in);
      } else if (in.peek() == '.') {
        in.ignore();
        ret = parseDouble(in, 0, 0, 1);
      } else {
        ret = parseOctal(in);
      }

      break;
    }
    case '-': {
      in.ignore();
      ret = parseDecimal(in, -1);
      break;
    }
    case '+':
      in.ignore();
    // fall through
    default:
      ret = parseDecimal(in, 1);
      break;
    }

    return ret;
  }

  // parse a number in hex format
  bool parseHex(std::istream& in) {
    uint64_t val = 0;
    int num_digits = 0;
    char ch = tolower(in.peek());
    while (in.good() && !strchr(kJsonDelim, ch) && (++num_digits) <= 16) {
      if (ch >= '0' && ch <= '9') {
        val = (val << 4) + (ch - '0');
      } else if (ch >= 'a' && ch <= 'f') {
        val = (val << 4) + (ch - 'a' + 10);
      } else { // unrecognized hex digit
        err_ = FbsonErrType::E_INVALID_HEX;
        return false;
      }

      in.ignore();
      ch = tolower(in.peek());
    }

    int size = 0;
    if (num_digits <= 2) {
      size = writer_.writeInt8((int8_t)val);
    } else if (num_digits <= 4) {
      size = writer_.writeInt16((int16_t)val);
    } else if (num_digits <= 8) {
      size = writer_.writeInt32((int32_t)val);
    } else if (num_digits <= 16) {
      size = writer_.writeInt64(val);
    } else {
      err_ = FbsonErrType::E_HEX_OVERFLOW;
      return false;
    }

    if (size == 0) {
      err_ = FbsonErrType::E_OUTPUT_FAIL;
      return false;
    }

    return true;
  }

  // parse a number in octal format
  bool parseOctal(std::istream& in) {
    int64_t val = 0;
    char ch = in.peek();
    while (in.good() && !strchr(kJsonDelim, ch)) {
      if (ch >= '0' && ch <= '7') {
        val = val * 8 + (ch - '0');
      } else {
        err_ = FbsonErrType::E_INVALID_OCTAL;
        return false;
      }

      // check if the number overflows
      if (val < 0) {
        err_ = FbsonErrType::E_OCTAL_OVERFLOW;
        return false;
      }

      in.ignore();
      ch = in.peek();
    }

    int size = 0;
    if (val <= std::numeric_limits<int8_t>::max()) {
      size = writer_.writeInt8((int8_t)val);
    } else if (val <= std::numeric_limits<int16_t>::max()) {
      size = writer_.writeInt16((int16_t)val);
    } else if (val <= std::numeric_limits<int32_t>::max()) {
      size = writer_.writeInt32((int32_t)val);
    } else { // val <= INT64_MAX
      size = writer_.writeInt64(val);
    }

    if (size == 0) {
      err_ = FbsonErrType::E_OUTPUT_FAIL;
      return false;
    }

    return true;
  }

  // parse a number in decimal (including float)
  bool parseDecimal(std::istream& in, int sign) {
    int64_t val = 0;
    int precision = 0;

    char ch = 0;
    while (in.good() && (ch = in.peek()) == '0')
      in.ignore();

    while (in.good() && !strchr(kJsonDelim, ch)) {
      if (ch >= '0' && ch <= '9') {
        val = val * 10 + (ch - '0');
        ++precision;
      } else if (ch == '.') {
        // note we don't pop out '.'
        return parseDouble(in, val, precision, sign);
      } else {
        err_ = FbsonErrType::E_INVALID_DECIMAL;
        return false;
      }

      in.ignore();

      // if the number overflows int64_t, first parse it as double iff we see a
      // decimal point later. Otherwise, will treat it as overflow
      if (val < 0 && val > std::numeric_limits<int64_t>::min()) {
        return parseDouble(in, (uint64_t)val, precision, sign);
      }

      ch = in.peek();
    }

    if (sign < 0) {
      val = -val;
    }

    int size = 0;
    if (val >= std::numeric_limits<int8_t>::min() &&
        val <= std::numeric_limits<int8_t>::max()) {
      size = writer_.writeInt8((int8_t)val);
    } else if (val >= std::numeric_limits<int16_t>::min() &&
               val <= std::numeric_limits<int16_t>::max()) {
      size = writer_.writeInt16((int16_t)val);
    } else if (val >= std::numeric_limits<int32_t>::min() &&
               val <= std::numeric_limits<int32_t>::max()) {
      size = writer_.writeInt32((int32_t)val);
    } else { // val <= INT64_MAX
      size = writer_.writeInt64(val);
    }

    if (size == 0) {
      err_ = FbsonErrType::E_OUTPUT_FAIL;
      return false;
    }

    return true;
  }

  // parse IEEE745 double precision:
  // Significand precision length - 15
  // Maximum exponent value - 308
  //
  // "If a decimal string with at most 15 significant digits is converted to
  // IEEE 754 double precision representation and then converted back to a
  // string with the same number of significant digits, then the final string
  // should match the original"
  bool parseDouble(std::istream& in, double val, int precision, int sign) {
    int integ = precision;
    int frac = 0;
    bool is_frac = false;

    char ch = in.peek();
    if (ch == '.') {
      is_frac = true;
      in.ignore();
      ch = in.peek();
    }

    int exp = 0;
    while (in.good() && !strchr(kJsonDelim, ch)) {
      if (ch >= '0' && ch <= '9') {
        if (precision < 15) {
          val = val * 10 + (ch - '0');
          if (is_frac) {
            ++frac;
          } else {
            ++integ;
          }
          ++precision;
        } else if (!is_frac) {
          ++exp;
        }
      } else if (ch == 'e' || ch == 'E') {
        in.ignore();
        int exp2;
        if (!parseExponent(in, exp2)) {
          return false;
        }

        exp += exp2;
        // check if exponent overflows
        if (exp > 308 || exp < -308) {
          err_ = FbsonErrType::E_EXPONENT_OVERFLOW;
          return false;
        }

        is_frac = true;
        break;
      }

      in.ignore();
      ch = in.peek();
    }

    if (!is_frac) {
      err_ = FbsonErrType::E_DECIMAL_OVERFLOW;
      return false;
    }

    val *= std::pow(10, exp - frac);
    if (std::isnan(val) || std::isinf(val)) {
      err_ = FbsonErrType::E_DOUBLE_OVERFLOW;
      return false;
    }

    if (sign < 0) {
      val = -val;
    }

    if (writer_.writeDouble(val) == 0) {
      err_ = FbsonErrType::E_OUTPUT_FAIL;
      return false;
    }

    return true;
  }

  // parse the exponent part of a double number
  bool parseExponent(std::istream& in, int& exp) {
    bool neg = false;

    char ch = in.peek();
    if (ch == '+') {
      in.ignore();
      ch = in.peek();
    } else if (ch == '-') {
      neg = true;
      in.ignore();
      ch = in.peek();
    }

    exp = 0;
    while (in.good() && !strchr(kJsonDelim, ch)) {
      if (ch >= '0' && ch <= '9') {
        exp = exp * 10 + (ch - '0');
      } else {
        err_ = FbsonErrType::E_INVALID_EXPONENT;
        return false;
      }

      if (exp > 308) {
        err_ = FbsonErrType::E_EXPONENT_OVERFLOW;
        return false;
      }

      in.ignore();
      ch = in.peek();
    }

    if (neg) {
      exp = -exp;
    }

    return true;
  }

  void trim(std::istream& in) {
    while (in.good() && strchr(kWhiteSpace, in.peek())) {
      in.ignore();
    }
  }

 private:
  FbsonWriterT<OS_TYPE> writer_;
  FbsonErrType err_;
};

typedef FbsonJsonParserT<FbsonOutStream> FbsonJsonParser;

} // namespace fbson

#endif // FBSON_FBSONPARSER_H
