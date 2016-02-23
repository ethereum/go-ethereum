// Copyright (c) 2015, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// This file implements the callback "bridge" between Java and C++ for
// rocksdb::Logger

#ifndef JAVA_ROCKSJNI_LOGGERJNICALLBACK_H_
#define JAVA_ROCKSJNI_LOGGERJNICALLBACK_H_

#include <jni.h>
#include <string>
#include "port/port.h"
#include "rocksdb/env.h"

namespace rocksdb {

  class LoggerJniCallback : public Logger {
   public:
     LoggerJniCallback(JNIEnv* env, jobject jLogger);
     virtual ~LoggerJniCallback();

     using Logger::SetInfoLogLevel;
     using Logger::GetInfoLogLevel;
     // Write an entry to the log file with the specified format.
     virtual void Logv(const char* format, va_list ap);
     // Write an entry to the log file with the specified log level
     // and format.  Any log with level under the internal log level
     // of *this (see @SetInfoLogLevel and @GetInfoLogLevel) will not be
     // printed.
     virtual void Logv(const InfoLogLevel log_level,
         const char* format, va_list ap);

   protected:
     JNIEnv* getJniEnv() const;
   private:
     JavaVM* m_jvm;
     jobject m_jLogger;
     jmethodID m_jLogMethodId;
  };
}  // namespace rocksdb

#endif  // JAVA_ROCKSJNI_LOGGERJNICALLBACK_H_
