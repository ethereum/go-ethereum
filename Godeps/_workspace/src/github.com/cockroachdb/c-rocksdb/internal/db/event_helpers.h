//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
#pragma once

#include <memory>
#include <string>
#include <vector>

#include "db/column_family.h"
#include "db/version_edit.h"
#include "rocksdb/listener.h"
#include "rocksdb/table_properties.h"
#include "util/event_logger.h"

namespace rocksdb {

class EventHelpers {
 public:
  static void AppendCurrentTime(JSONWriter* json_writer);
  static void LogAndNotifyTableFileCreation(
      EventLogger* event_logger,
      const std::vector<std::shared_ptr<EventListener>>& listeners,
      const FileDescriptor& fd, const TableFileCreationInfo& info);
  static void LogAndNotifyTableFileDeletion(
      EventLogger* event_logger, int job_id,
      uint64_t file_number, const std::string& file_path,
      const Status& status, const std::string& db_name,
      const std::vector<std::shared_ptr<EventListener>>& listeners);
};

}  // namespace rocksdb
