#ifndef SRC_UTIL_H_
#define SRC_UTIL_H_

#define FIXED_ONE_BYTE_STRING(isolate, string)                                \
  (node::OneByteString((isolate), (string), sizeof(string) - 1))

#endif  // SRC_UTIL_H_
