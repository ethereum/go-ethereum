#ifndef SRC_UTIL_INL_H_
#define SRC_UTIL_INL_H_

#include "util.h"
#include "v8.h"

namespace node {

inline v8::Local<v8::String> OneByteString(v8::Isolate* isolate,
                                           const char* data,
                                           int length) {
  return v8::String::NewFromOneByte(isolate,
                                    reinterpret_cast<const uint8_t*>(data),
                                    v8::NewStringType::kNormal,
                                    length).ToLocalChecked();
}

inline v8::Local<v8::String> OneByteString(v8::Isolate* isolate,
                                           const signed char* data,
                                           int length) {
  return v8::String::NewFromOneByte(isolate,
                                    reinterpret_cast<const uint8_t*>(data),
                                    v8::NewStringType::kNormal,
                                    length).ToLocalChecked();
}

inline v8::Local<v8::String> OneByteString(v8::Isolate* isolate,
                                           const unsigned char* data,
                                           int length) {
  return v8::String::NewFromOneByte(isolate,
                                    reinterpret_cast<const uint8_t*>(data),
                                    v8::NewStringType::kNormal,
                                    length).ToLocalChecked();
}

}  // namespace node

#endif  // SRC_UTIL_INL_H_
