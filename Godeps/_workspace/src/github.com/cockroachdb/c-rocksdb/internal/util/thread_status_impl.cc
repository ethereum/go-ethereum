// Copyright (c) 2013, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//

#include <sstream>

#include "rocksdb/env.h"
#include "rocksdb/thread_status.h"
#include "util/logging.h"
#include "util/thread_operation.h"

namespace rocksdb {

#if ROCKSDB_USING_THREAD_STATUS
const std::string& ThreadStatus::GetThreadTypeName(
    ThreadStatus::ThreadType thread_type) {
  static std::string thread_type_names[NUM_THREAD_TYPES + 1] = {
      "High Pri", "Low Pri", "User", "Unknown"};
  if (thread_type < 0 || thread_type >= NUM_THREAD_TYPES) {
    return thread_type_names[NUM_THREAD_TYPES];  // "Unknown"
  }
  return thread_type_names[thread_type];
}

const std::string& ThreadStatus::GetOperationName(
    ThreadStatus::OperationType op_type) {
  if (op_type < 0 || op_type >= NUM_OP_TYPES) {
    return global_operation_table[OP_UNKNOWN].name;
  }
  return global_operation_table[op_type].name;
}

const std::string& ThreadStatus::GetOperationStageName(
    ThreadStatus::OperationStage stage) {
  if (stage < 0 || stage >= NUM_OP_STAGES) {
    return global_op_stage_table[STAGE_UNKNOWN].name;
  }
  return global_op_stage_table[stage].name;
}

const std::string& ThreadStatus::GetStateName(
    ThreadStatus::StateType state_type) {
  if (state_type < 0 || state_type >= NUM_STATE_TYPES) {
    return global_state_table[STATE_UNKNOWN].name;
  }
  return global_state_table[state_type].name;
}

const std::string ThreadStatus::MicrosToString(uint64_t micros) {
  if (micros == 0) {
    return "";
  }
  const int kBufferLen = 100;
  char buffer[kBufferLen];
  AppendHumanMicros(micros, buffer, kBufferLen, false);
  return std::string(buffer);
}

const std::string& ThreadStatus::GetOperationPropertyName(
    ThreadStatus::OperationType op_type, int i) {
  static const std::string empty_str = "";
  switch (op_type) {
    case ThreadStatus::OP_COMPACTION:
      if (i >= NUM_COMPACTION_PROPERTIES) {
        return empty_str;
      }
      return compaction_operation_properties[i].name;
    case ThreadStatus::OP_FLUSH:
      if (i >= NUM_FLUSH_PROPERTIES) {
        return empty_str;
      }
      return flush_operation_properties[i].name;
    default:
      return empty_str;
  }
}

std::map<std::string, uint64_t>
    ThreadStatus::InterpretOperationProperties(
    ThreadStatus::OperationType op_type,
    const uint64_t* op_properties) {
  int num_properties;
  switch (op_type) {
    case OP_COMPACTION:
      num_properties = NUM_COMPACTION_PROPERTIES;
      break;
    case OP_FLUSH:
      num_properties = NUM_FLUSH_PROPERTIES;
      break;
    default:
      num_properties = 0;
  }

  std::map<std::string, uint64_t> property_map;
  for (int i = 0; i < num_properties; ++i) {
    if (op_type == OP_COMPACTION &&
        i == COMPACTION_INPUT_OUTPUT_LEVEL) {
      property_map.insert(
          {"BaseInputLevel", op_properties[i] >> 32});
      property_map.insert(
          {"OutputLevel", op_properties[i] % (uint64_t(1) << 32U)});
    } else if (op_type == OP_COMPACTION &&
               i == COMPACTION_PROP_FLAGS) {
      property_map.insert(
          {"IsManual", ((op_properties[i] & 2) >> 1)});
      property_map.insert(
          {"IsDeletion", ((op_properties[i] & 4) >> 2)});
      property_map.insert(
          {"IsTrivialMove", ((op_properties[i] & 8) >> 3)});
    } else {
      property_map.insert(
          {GetOperationPropertyName(op_type, i), op_properties[i]});
    }
  }
  return property_map;
}


#else

const std::string& ThreadStatus::GetThreadTypeName(
    ThreadStatus::ThreadType thread_type) {
  static std::string dummy_str = "";
  return dummy_str;
}

const std::string& ThreadStatus::GetOperationName(
    ThreadStatus::OperationType op_type) {
  static std::string dummy_str = "";
  return dummy_str;
}

const std::string& ThreadStatus::GetOperationStageName(
    ThreadStatus::OperationStage stage) {
  static std::string dummy_str = "";
  return dummy_str;
}

const std::string& ThreadStatus::GetStateName(
    ThreadStatus::StateType state_type) {
  static std::string dummy_str = "";
  return dummy_str;
}

const std::string ThreadStatus::MicrosToString(
    uint64_t op_elapsed_time) {
  static std::string dummy_str = "";
  return dummy_str;
}

const std::string& ThreadStatus::GetOperationPropertyName(
    ThreadStatus::OperationType op_type, int i) {
  static std::string dummy_str = "";
  return dummy_str;
}

std::map<std::string, uint64_t>
    ThreadStatus::InterpretOperationProperties(
    ThreadStatus::OperationType op_type,
    const uint64_t* op_properties) {
  return std::map<std::string, uint64_t>();
}

#endif  // ROCKSDB_USING_THREAD_STATUS
}  // namespace rocksdb
