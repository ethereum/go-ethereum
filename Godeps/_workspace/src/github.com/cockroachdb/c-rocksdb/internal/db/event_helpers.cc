//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include "db/event_helpers.h"

namespace rocksdb {

namespace {
inline double SafeDivide(double a, double b) { return b == 0.0 ? 0 : a / b; }
}  // namespace

void EventHelpers::AppendCurrentTime(JSONWriter* jwriter) {
  *jwriter << "time_micros"
           << std::chrono::duration_cast<std::chrono::microseconds>(
                  std::chrono::system_clock::now().time_since_epoch()).count();
}

void EventHelpers::LogAndNotifyTableFileCreation(
    EventLogger* event_logger,
    const std::vector<std::shared_ptr<EventListener>>& listeners,
    const FileDescriptor& fd, const TableFileCreationInfo& info) {
  assert(event_logger);
  JSONWriter jwriter;
  AppendCurrentTime(&jwriter);
  jwriter << "cf_name" << info.cf_name
          << "job" << info.job_id
          << "event" << "table_file_creation"
          << "file_number" << fd.GetNumber()
          << "file_size" << fd.GetFileSize();

  // table_properties
  {
    jwriter << "table_properties";
    jwriter.StartObject();

    // basic properties:
    jwriter << "data_size" << info.table_properties.data_size
            << "index_size" << info.table_properties.index_size
            << "filter_size" << info.table_properties.filter_size
            << "raw_key_size" << info.table_properties.raw_key_size
            << "raw_average_key_size" << SafeDivide(
                info.table_properties.raw_key_size,
                info.table_properties.num_entries)
            << "raw_value_size" << info.table_properties.raw_value_size
            << "raw_average_value_size" << SafeDivide(
               info.table_properties.raw_value_size,
               info.table_properties.num_entries)
            << "num_data_blocks" << info.table_properties.num_data_blocks
            << "num_entries" << info.table_properties.num_entries
            << "filter_policy_name" <<
                info.table_properties.filter_policy_name;

    // user collected properties
    for (const auto& prop : info.table_properties.user_collected_properties) {
      jwriter << prop.first << prop.second;
    }
    jwriter.EndObject();
  }
  jwriter.EndObject();

  event_logger->Log(jwriter);

#ifndef ROCKSDB_LITE
  if (listeners.size() == 0) {
    return;
  }

  for (auto listener : listeners) {
    listener->OnTableFileCreated(info);
  }
#endif  // !ROCKSDB_LITE
}

void EventHelpers::LogAndNotifyTableFileDeletion(
    EventLogger* event_logger, int job_id,
    uint64_t file_number, const std::string& file_path,
    const Status& status, const std::string& dbname,
    const std::vector<std::shared_ptr<EventListener>>& listeners) {

  JSONWriter jwriter;
  AppendCurrentTime(&jwriter);

  jwriter << "job" << job_id
          << "event" << "table_file_deletion"
          << "file_number" << file_number;
  if (!status.ok()) {
    jwriter << "status" << status.ToString();
  }

  jwriter.EndObject();

  event_logger->Log(jwriter);

#ifndef ROCKSDB_LITE
  TableFileDeletionInfo info;
  info.db_name = dbname;
  info.job_id = job_id;
  info.file_path = file_path;
  info.status = status;
  for (auto listener : listeners) {
    listener->OnTableFileDeleted(info);
  }
#endif  // !ROCKSDB_LITE
}

}  // namespace rocksdb
