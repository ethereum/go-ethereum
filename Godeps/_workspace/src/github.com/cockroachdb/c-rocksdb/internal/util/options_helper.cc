//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
#include "util/options_helper.h"

#include <cassert>
#include <cctype>
#include <cstdlib>
#include <unordered_set>
#include "rocksdb/cache.h"
#include "rocksdb/convenience.h"
#include "rocksdb/filter_policy.h"
#include "rocksdb/options.h"
#include "rocksdb/rate_limiter.h"
#include "rocksdb/slice_transform.h"
#include "rocksdb/table.h"
#include "table/block_based_table_factory.h"
#include "util/logging.h"
#include "util/string_util.h"

namespace rocksdb {

#ifndef ROCKSDB_LITE

namespace {
CompressionType ParseCompressionType(const std::string& type) {
  if (type == "kNoCompression") {
    return kNoCompression;
  } else if (type == "kSnappyCompression") {
    return kSnappyCompression;
  } else if (type == "kZlibCompression") {
    return kZlibCompression;
  } else if (type == "kBZip2Compression") {
    return kBZip2Compression;
  } else if (type == "kLZ4Compression") {
    return kLZ4Compression;
  } else if (type == "kLZ4HCCompression") {
    return kLZ4HCCompression;
  } else if (type == "kZSTDNotFinalCompression") {
    return kZSTDNotFinalCompression;
  } else {
    throw std::invalid_argument("Unknown compression type: " + type);
  }
  return kNoCompression;
}

BlockBasedTableOptions::IndexType ParseBlockBasedTableIndexType(
    const std::string& type) {
  if (type == "kBinarySearch") {
    return BlockBasedTableOptions::kBinarySearch;
  } else if (type == "kHashSearch") {
    return BlockBasedTableOptions::kHashSearch;
  }
  throw std::invalid_argument("Unknown index type: " + type);
}

ChecksumType ParseBlockBasedTableChecksumType(
    const std::string& type) {
  if (type == "kNoChecksum") {
    return kNoChecksum;
  } else if (type == "kCRC32c") {
    return kCRC32c;
  } else if (type == "kxxHash") {
    return kxxHash;
  }
  throw std::invalid_argument("Unknown checksum type: " + type);
}

bool ParseBoolean(const std::string& type, const std::string& value) {
  if (value == "true" || value == "1") {
    return true;
  } else if (value == "false" || value == "0") {
    return false;
  }
  throw std::invalid_argument(type);
}

uint64_t ParseUint64(const std::string& value) {
  size_t endchar;
#ifndef CYGWIN
  uint64_t num = std::stoull(value.c_str(), &endchar);
#else
  char* endptr;
  uint64_t num = std::strtoul(value.c_str(), &endptr, 0);
  endchar = endptr - value.c_str();
#endif

  if (endchar < value.length()) {
    char c = value[endchar];
    if (c == 'k' || c == 'K')
      num <<= 10LL;
    else if (c == 'm' || c == 'M')
      num <<= 20LL;
    else if (c == 'g' || c == 'G')
      num <<= 30LL;
    else if (c == 't' || c == 'T')
      num <<= 40LL;
  }

  return num;
}

size_t ParseSizeT(const std::string& value) {
  return static_cast<size_t>(ParseUint64(value));
}

uint32_t ParseUint32(const std::string& value) {
  uint64_t num = ParseUint64(value);
  if ((num >> 32LL) == 0) {
    return static_cast<uint32_t>(num);
  } else {
    throw std::out_of_range(value);
  }
}

int ParseInt(const std::string& value) {
  size_t endchar;
#ifndef CYGWIN
  int num = std::stoi(value.c_str(), &endchar);
#else
  char* endptr;
  int num = std::strtoul(value.c_str(), &endptr, 0);
  endchar = endptr - value.c_str();
#endif

  if (endchar < value.length()) {
    char c = value[endchar];
    if (c == 'k' || c == 'K')
      num <<= 10;
    else if (c == 'm' || c == 'M')
      num <<= 20;
    else if (c == 'g' || c == 'G')
      num <<= 30;
  }

  return num;
}

double ParseDouble(const std::string& value) {
#ifndef CYGWIN
  return std::stod(value);
#else
  return std::strtod(value.c_str(), 0);
#endif
}

static const std::unordered_map<char, std::string>
    compaction_style_to_string_map = {
        {kCompactionStyleLevel, "kCompactionStyleLevel"},
        {kCompactionStyleUniversal, "kCompactionStyleUniversal"},
        {kCompactionStyleFIFO, "kCompactionStyleFIFO"},
        {kCompactionStyleNone, "kCompactionStyleNone"}};

CompactionStyle ParseCompactionStyle(const std::string& type) {
  for (auto const& entry : compaction_style_to_string_map) {
    if (entry.second == type) {
      return static_cast<CompactionStyle>(entry.first);
    }
  }
  throw std::invalid_argument("unknown compaction style: " + type);
  return kCompactionStyleLevel;
}

std::string CompactionStyleToString(const CompactionStyle style) {
  auto iter = compaction_style_to_string_map.find(style);
  assert(iter != compaction_style_to_string_map.end());
  return iter->second;
}

bool ParseOptionHelper(char* opt_address, const OptionType& opt_type,
                       const std::string& value) {
  switch (opt_type) {
    case OptionType::kBoolean:
      *reinterpret_cast<bool*>(opt_address) = ParseBoolean("", value);
      break;
    case OptionType::kInt:
      *reinterpret_cast<int*>(opt_address) = ParseInt(value);
      break;
    case OptionType::kUInt:
      *reinterpret_cast<unsigned int*>(opt_address) = ParseUint32(value);
      break;
    case OptionType::kUInt32T:
      *reinterpret_cast<uint32_t*>(opt_address) = ParseUint32(value);
      break;
    case OptionType::kUInt64T:
      *reinterpret_cast<uint64_t*>(opt_address) = ParseUint64(value);
      break;
    case OptionType::kSizeT:
      *reinterpret_cast<size_t*>(opt_address) = ParseSizeT(value);
      break;
    case OptionType::kString:
      *reinterpret_cast<std::string*>(opt_address) = value;
      break;
    case OptionType::kDouble:
      *reinterpret_cast<double*>(opt_address) = ParseDouble(value);
      break;
    case OptionType::kCompactionStyle:
      *reinterpret_cast<CompactionStyle*>(opt_address) =
          ParseCompactionStyle(value);
      break;
    default:
      return false;
  }
  return true;
}

bool SerializeSingleOptionHelper(const char* opt_address,
                                 const OptionType opt_type,
                                 std::string* value) {
  assert(value);
  switch (opt_type) {
    case OptionType::kBoolean:
      *value = *(reinterpret_cast<const bool*>(opt_address)) ? "true" : "false";
      break;
    case OptionType::kInt:
      *value = ToString(*(reinterpret_cast<const int*>(opt_address)));
      break;
    case OptionType::kUInt:
      *value = ToString(*(reinterpret_cast<const unsigned int*>(opt_address)));
      break;
    case OptionType::kUInt32T:
      *value = ToString(*(reinterpret_cast<const uint32_t*>(opt_address)));
      break;
    case OptionType::kUInt64T:
      *value = ToString(*(reinterpret_cast<const uint64_t*>(opt_address)));
      break;
    case OptionType::kSizeT:
      *value = ToString(*(reinterpret_cast<const size_t*>(opt_address)));
      break;
    case OptionType::kDouble:
      *value = ToString(*(reinterpret_cast<const double*>(opt_address)));
      break;
    case OptionType::kString:
      *value = *(reinterpret_cast<const std::string*>(opt_address));
      break;
    case OptionType::kCompactionStyle:
      *value = CompactionStyleToString(
          *(reinterpret_cast<const CompactionStyle*>(opt_address)));
      break;
    default:
      return false;
  }
  return true;
}

}  // anonymouse namespace

template<typename OptionsType>
bool ParseMemtableOptions(const std::string& name, const std::string& value,
                          OptionsType* new_options) {
  if (name == "write_buffer_size") {
    new_options->write_buffer_size = ParseSizeT(value);
  } else if (name == "arena_block_size") {
    new_options->arena_block_size = ParseSizeT(value);
  } else if (name == "memtable_prefix_bloom_bits") {
    new_options->memtable_prefix_bloom_bits = ParseUint32(value);
  } else if (name == "memtable_prefix_bloom_probes") {
    new_options->memtable_prefix_bloom_probes = ParseUint32(value);
  } else if (name == "memtable_prefix_bloom_huge_page_tlb_size") {
    new_options->memtable_prefix_bloom_huge_page_tlb_size =
      ParseSizeT(value);
  } else if (name == "max_successive_merges") {
    new_options->max_successive_merges = ParseSizeT(value);
  } else if (name == "filter_deletes") {
    new_options->filter_deletes = ParseBoolean(name, value);
  } else if (name == "max_write_buffer_number") {
    new_options->max_write_buffer_number = ParseInt(value);
  } else if (name == "inplace_update_num_locks") {
    new_options->inplace_update_num_locks = ParseSizeT(value);
  } else {
    return false;
  }
  return true;
}

template<typename OptionsType>
bool ParseCompactionOptions(const std::string& name, const std::string& value,
                            OptionsType* new_options) {
  if (name == "disable_auto_compactions") {
    new_options->disable_auto_compactions = ParseBoolean(name, value);
  } else if (name == "soft_rate_limit") {
    new_options->soft_rate_limit = ParseDouble(value);
  } else if (name == "hard_rate_limit") {
    new_options->hard_rate_limit = ParseDouble(value);
  } else if (name == "level0_file_num_compaction_trigger") {
    new_options->level0_file_num_compaction_trigger = ParseInt(value);
  } else if (name == "level0_slowdown_writes_trigger") {
    new_options->level0_slowdown_writes_trigger = ParseInt(value);
  } else if (name == "level0_stop_writes_trigger") {
    new_options->level0_stop_writes_trigger = ParseInt(value);
  } else if (name == "max_grandparent_overlap_factor") {
    new_options->max_grandparent_overlap_factor = ParseInt(value);
  } else if (name == "expanded_compaction_factor") {
    new_options->expanded_compaction_factor = ParseInt(value);
  } else if (name == "source_compaction_factor") {
    new_options->source_compaction_factor = ParseInt(value);
  } else if (name == "target_file_size_base") {
    new_options->target_file_size_base = ParseInt(value);
  } else if (name == "target_file_size_multiplier") {
    new_options->target_file_size_multiplier = ParseInt(value);
  } else if (name == "max_bytes_for_level_base") {
    new_options->max_bytes_for_level_base = ParseUint64(value);
  } else if (name == "max_bytes_for_level_multiplier") {
    new_options->max_bytes_for_level_multiplier = ParseInt(value);
  } else if (name == "max_bytes_for_level_multiplier_additional") {
    new_options->max_bytes_for_level_multiplier_additional.clear();
    size_t start = 0;
    while (true) {
      size_t end = value.find(':', start);
      if (end == std::string::npos) {
        new_options->max_bytes_for_level_multiplier_additional.push_back(
            ParseInt(value.substr(start)));
        break;
      } else {
        new_options->max_bytes_for_level_multiplier_additional.push_back(
            ParseInt(value.substr(start, end - start)));
        start = end + 1;
      }
    }
  } else if (name == "verify_checksums_in_compaction") {
    new_options->verify_checksums_in_compaction = ParseBoolean(name, value);
  } else {
    return false;
  }
  return true;
}

template<typename OptionsType>
bool ParseMiscOptions(const std::string& name, const std::string& value,
                      OptionsType* new_options) {
  if (name == "max_sequential_skip_in_iterations") {
    new_options->max_sequential_skip_in_iterations = ParseUint64(value);
  } else if (name == "paranoid_file_checks") {
    new_options->paranoid_file_checks = ParseBoolean(name, value);
  } else {
    return false;
  }
  return true;
}

Status GetMutableOptionsFromStrings(
    const MutableCFOptions& base_options,
    const std::unordered_map<std::string, std::string>& options_map,
    MutableCFOptions* new_options) {
  assert(new_options);
  *new_options = base_options;
  for (const auto& o : options_map) {
    try {
      if (ParseMemtableOptions(o.first, o.second, new_options)) {
      } else if (ParseCompactionOptions(o.first, o.second, new_options)) {
      } else if (ParseMiscOptions(o.first, o.second, new_options)) {
      } else {
        return Status::InvalidArgument(
            "unsupported dynamic option: " + o.first);
      }
    } catch (std::exception& e) {
      return Status::InvalidArgument("error parsing " + o.first + ":" +
                                     std::string(e.what()));
    }
  }
  return Status::OK();
}

namespace {

std::string trim(const std::string& str) {
  if (str.empty()) return std::string();
  size_t start = 0;
  size_t end = str.size() - 1;
  while (isspace(str[start]) != 0 && start <= end) {
    ++start;
  }
  while (isspace(str[end]) != 0 && start <= end) {
    --end;
  }
  if (start <= end) {
    return str.substr(start, end - start + 1);
  }
  return std::string();
}

}  // anonymous namespace

Status StringToMap(const std::string& opts_str,
                   std::unordered_map<std::string, std::string>* opts_map) {
  assert(opts_map);
  // Example:
  //   opts_str = "write_buffer_size=1024;max_write_buffer_number=2;"
  //              "nested_opt={opt1=1;opt2=2};max_bytes_for_level_base=100"
  size_t pos = 0;
  std::string opts = trim(opts_str);
  while (pos < opts.size()) {
    size_t eq_pos = opts.find('=', pos);
    if (eq_pos == std::string::npos) {
      return Status::InvalidArgument("Mismatched key value pair, '=' expected");
    }
    std::string key = trim(opts.substr(pos, eq_pos - pos));
    if (key.empty()) {
      return Status::InvalidArgument("Empty key found");
    }

    // skip space after '=' and look for '{' for possible nested options
    pos = eq_pos + 1;
    while (pos < opts.size() && isspace(opts[pos])) {
      ++pos;
    }
    // Empty value at the end
    if (pos >= opts.size()) {
      (*opts_map)[key] = "";
      break;
    }
    if (opts[pos] == '{') {
      int count = 1;
      size_t brace_pos = pos + 1;
      while (brace_pos < opts.size()) {
        if (opts[brace_pos] == '{') {
          ++count;
        } else if (opts[brace_pos] == '}') {
          --count;
          if (count == 0) {
            break;
          }
        }
        ++brace_pos;
      }
      // found the matching closing brace
      if (count == 0) {
        (*opts_map)[key] = trim(opts.substr(pos + 1, brace_pos - pos - 1));
        // skip all whitespace and move to the next ';'
        // brace_pos points to the next position after the matching '}'
        pos = brace_pos + 1;
        while (pos < opts.size() && isspace(opts[pos])) {
          ++pos;
        }
        if (pos < opts.size() && opts[pos] != ';') {
          return Status::InvalidArgument(
              "Unexpected chars after nested options");
        }
        ++pos;
      } else {
        return Status::InvalidArgument(
            "Mismatched curly braces for nested options");
      }
    } else {
      size_t sc_pos = opts.find(';', pos);
      if (sc_pos == std::string::npos) {
        (*opts_map)[key] = trim(opts.substr(pos));
        // It either ends with a trailing semi-colon or the last key-value pair
        break;
      } else {
        (*opts_map)[key] = trim(opts.substr(pos, sc_pos - pos));
      }
      pos = sc_pos + 1;
    }
  }

  return Status::OK();
}

bool ParseColumnFamilyOption(const std::string& name, const std::string& value,
                             ColumnFamilyOptions* new_options) {
  try {
    if (name == "max_bytes_for_level_multiplier_additional") {
      new_options->max_bytes_for_level_multiplier_additional.clear();
      size_t start = 0;
      while (true) {
        size_t end = value.find(':', start);
        if (end == std::string::npos) {
          new_options->max_bytes_for_level_multiplier_additional.push_back(
              ParseInt(value.substr(start)));
          break;
        } else {
          new_options->max_bytes_for_level_multiplier_additional.push_back(
              ParseInt(value.substr(start, end - start)));
          start = end + 1;
        }
      }
    } else if (name == "block_based_table_factory") {
      // Nested options
      BlockBasedTableOptions table_opt, base_table_options;
      auto block_based_table_factory = dynamic_cast<BlockBasedTableFactory*>(
          new_options->table_factory.get());
      if (block_based_table_factory != nullptr) {
        base_table_options = block_based_table_factory->GetTableOptions();
      }
      Status table_opt_s = GetBlockBasedTableOptionsFromString(
          base_table_options, value, &table_opt);
      if (!table_opt_s.ok()) {
        return false;
      }
      new_options->table_factory.reset(NewBlockBasedTableFactory(table_opt));
    } else if (name == "compression") {
      new_options->compression = ParseCompressionType(value);
    } else if (name == "compression_per_level") {
      new_options->compression_per_level.clear();
      size_t start = 0;
      while (true) {
        size_t end = value.find(':', start);
        if (end == std::string::npos) {
          new_options->compression_per_level.push_back(
              ParseCompressionType(value.substr(start)));
          break;
        } else {
          new_options->compression_per_level.push_back(
              ParseCompressionType(value.substr(start, end - start)));
          start = end + 1;
        }
      }
    } else if (name == "compression_opts") {
      size_t start = 0;
      size_t end = value.find(':');
      if (end == std::string::npos) {
        return false;
      }
      new_options->compression_opts.window_bits =
          ParseInt(value.substr(start, end - start));
      start = end + 1;
      end = value.find(':', start);
      if (end == std::string::npos) {
        return false;
      }
      new_options->compression_opts.level =
          ParseInt(value.substr(start, end - start));
      start = end + 1;
      if (start >= value.size()) {
        return false;
      }
      new_options->compression_opts.strategy =
          ParseInt(value.substr(start, value.size() - start));
    } else if (name == "compaction_options_universal") {
      // TODO(ljin): add support
      return false;
    } else if (name == "compaction_options_fifo") {
      new_options->compaction_options_fifo.max_table_files_size =
          ParseUint64(value);
    } else if (name == "prefix_extractor") {
      const std::string kFixedPrefixName = "fixed:";
      const std::string kCappedPrefixName = "capped:";
      auto& pe_value = value;
      if (pe_value.size() > kFixedPrefixName.size() &&
          pe_value.compare(0, kFixedPrefixName.size(), kFixedPrefixName) == 0) {
        int prefix_length =
            ParseInt(trim(value.substr(kFixedPrefixName.size())));
        new_options->prefix_extractor.reset(
            NewFixedPrefixTransform(prefix_length));
      } else if (pe_value.size() > kCappedPrefixName.size() &&
                 pe_value.compare(0, kCappedPrefixName.size(),
                                  kCappedPrefixName) == 0) {
        int prefix_length =
            ParseInt(trim(pe_value.substr(kCappedPrefixName.size())));
        new_options->prefix_extractor.reset(
            NewCappedPrefixTransform(prefix_length));
      } else {
        return false;
      }
    } else {
      auto iter = cf_options_type_info.find(name);
      if (iter == cf_options_type_info.end()) {
        return false;
      }
      const auto& opt_info = iter->second;
      return ParseOptionHelper(
          reinterpret_cast<char*>(new_options) + opt_info.offset, opt_info.type,
          value);
    }
  } catch (std::exception& e) {
    return false;
  }
  return true;
}

bool SerializeSingleDBOption(const DBOptions& db_options,
                             const std::string& name, std::string* opt_string) {
  auto iter = db_options_type_info.find(name);
  if (iter == db_options_type_info.end()) {
    return false;
  }
  auto& opt_info = iter->second;
  const char* opt_address =
      reinterpret_cast<const char*>(&db_options) + opt_info.offset;
  std::string value;
  bool result = SerializeSingleOptionHelper(opt_address, opt_info.type, &value);
  if (result) {
    *opt_string = name + " = " + value + ";  ";
  }
  return result;
}

Status GetStringFromDBOptions(const DBOptions& db_options,
                              std::string* opt_string) {
  assert(opt_string);
  opt_string->clear();
  for (auto iter = db_options_type_info.begin();
       iter != db_options_type_info.end(); ++iter) {
    std::string single_output;
    bool result =
        SerializeSingleDBOption(db_options, iter->first, &single_output);
    assert(result);
    if (result) {
      opt_string->append(single_output);
    }
  }
  return Status::OK();
}

bool SerializeSingleColumnFamilyOption(const ColumnFamilyOptions& cf_options,
                                       const std::string& name,
                                       std::string* opt_string) {
  auto iter = cf_options_type_info.find(name);
  if (iter == cf_options_type_info.end()) {
    return false;
  }
  auto& opt_info = iter->second;
  const char* opt_address =
      reinterpret_cast<const char*>(&cf_options) + opt_info.offset;
  std::string value;
  bool result = SerializeSingleOptionHelper(opt_address, opt_info.type, &value);
  if (result) {
    *opt_string = name + " = " + value + ";  ";
  }
  return result;
}

Status GetStringFromColumnFamilyOptions(const ColumnFamilyOptions& cf_options,
                                        std::string* opt_string) {
  assert(opt_string);
  opt_string->clear();
  for (auto iter = cf_options_type_info.begin();
       iter != cf_options_type_info.end(); ++iter) {
    std::string single_output;
    bool result = SerializeSingleColumnFamilyOption(cf_options, iter->first,
                                                    &single_output);
    if (result) {
      opt_string->append(single_output);
    } else {
      printf("failed to serialize %s\n", iter->first.c_str());
    }
    assert(result);
  }
  return Status::OK();
}

bool ParseDBOption(const std::string& name, const std::string& value,
                   DBOptions* new_options) {
  try {
    if (name == "rate_limiter_bytes_per_sec") {
      new_options->rate_limiter.reset(
          NewGenericRateLimiter(static_cast<int64_t>(ParseUint64(value))));
    } else {
      auto iter = db_options_type_info.find(name);
      if (iter == db_options_type_info.end()) {
        return false;
      }
      const auto& opt_info = iter->second;
      return ParseOptionHelper(
          reinterpret_cast<char*>(new_options) + opt_info.offset, opt_info.type,
          value);
    }
  } catch (const std::exception& e) {
    return false;
  }
  return true;
}

Status GetBlockBasedTableOptionsFromMap(
    const BlockBasedTableOptions& table_options,
    const std::unordered_map<std::string, std::string>& opts_map,
    BlockBasedTableOptions* new_table_options) {

  assert(new_table_options);
  *new_table_options = table_options;
  for (const auto& o : opts_map) {
    try {
      if (o.first == "cache_index_and_filter_blocks") {
        new_table_options->cache_index_and_filter_blocks =
          ParseBoolean(o.first, o.second);
      } else if (o.first == "index_type") {
        new_table_options->index_type = ParseBlockBasedTableIndexType(o.second);
      } else if (o.first == "hash_index_allow_collision") {
        new_table_options->hash_index_allow_collision =
          ParseBoolean(o.first, o.second);
      } else if (o.first == "checksum") {
        new_table_options->checksum =
          ParseBlockBasedTableChecksumType(o.second);
      } else if (o.first == "no_block_cache") {
        new_table_options->no_block_cache = ParseBoolean(o.first, o.second);
      } else if (o.first == "block_cache") {
        new_table_options->block_cache = NewLRUCache(ParseSizeT(o.second));
      } else if (o.first == "block_cache_compressed") {
        new_table_options->block_cache_compressed =
          NewLRUCache(ParseSizeT(o.second));
      } else if (o.first == "block_size") {
        new_table_options->block_size = ParseSizeT(o.second);
      } else if (o.first == "block_size_deviation") {
        new_table_options->block_size_deviation = ParseInt(o.second);
      } else if (o.first == "block_restart_interval") {
        new_table_options->block_restart_interval = ParseInt(o.second);
      } else if (o.first == "filter_policy") {
        // Expect the following format
        // bloomfilter:int:bool
        const std::string kName = "bloomfilter:";
        if (o.second.compare(0, kName.size(), kName) != 0) {
          return Status::InvalidArgument("Invalid filter policy name");
        }
        size_t pos = o.second.find(':', kName.size());
        if (pos == std::string::npos) {
          return Status::InvalidArgument("Invalid filter policy config, "
                                         "missing bits_per_key");
        }
        int bits_per_key = ParseInt(
            trim(o.second.substr(kName.size(), pos - kName.size())));
        bool use_block_based_builder =
          ParseBoolean("use_block_based_builder",
                       trim(o.second.substr(pos + 1)));
        new_table_options->filter_policy.reset(
            NewBloomFilterPolicy(bits_per_key, use_block_based_builder));
      } else if (o.first == "whole_key_filtering") {
        new_table_options->whole_key_filtering =
          ParseBoolean(o.first, o.second);
      } else {
        return Status::InvalidArgument("Unrecognized option: " + o.first);
      }
    } catch (std::exception& e) {
      return Status::InvalidArgument("error parsing " + o.first + ":" +
                                     std::string(e.what()));
    }
  }
  return Status::OK();
}

Status GetBlockBasedTableOptionsFromString(
    const BlockBasedTableOptions& table_options,
    const std::string& opts_str,
    BlockBasedTableOptions* new_table_options) {
  std::unordered_map<std::string, std::string> opts_map;
  Status s = StringToMap(opts_str, &opts_map);
  if (!s.ok()) {
    return s;
  }
  return GetBlockBasedTableOptionsFromMap(table_options, opts_map,
                                          new_table_options);
}

Status GetPlainTableOptionsFromMap(
    const PlainTableOptions& table_options,
    const std::unordered_map<std::string, std::string>& opts_map,
    PlainTableOptions* new_table_options) {
  assert(new_table_options);
  *new_table_options = table_options;

  for (const auto& o : opts_map) {
    try {
      if (o.first == "user_key_len") {
        new_table_options->user_key_len = ParseUint32(o.second);
      } else if (o.first == "bloom_bits_per_key") {
        new_table_options->bloom_bits_per_key = ParseInt(o.second);
      } else if (o.first == "hash_table_ratio") {
        new_table_options->hash_table_ratio = ParseDouble(o.second);
      } else if (o.first == "index_sparseness") {
        new_table_options->index_sparseness = ParseSizeT(o.second);
      } else if (o.first == "huge_page_tlb_size") {
        new_table_options->huge_page_tlb_size = ParseSizeT(o.second);
      } else if (o.first == "encoding_type") {
        if (o.second == "kPlain") {
          new_table_options->encoding_type = kPlain;
        } else if (o.second == "kPrefix") {
          new_table_options->encoding_type = kPrefix;
        } else {
          throw std::invalid_argument("Unknown encoding_type: " + o.second);
        }
      } else if (o.first == "full_scan_mode") {
        new_table_options->full_scan_mode = ParseBoolean(o.first, o.second);
      } else if (o.first == "store_index_in_file") {
        new_table_options->store_index_in_file =
            ParseBoolean(o.first, o.second);
      } else {
        return Status::InvalidArgument("Unrecognized option: " + o.first);
      }
    } catch (std::exception& e) {
      return Status::InvalidArgument("error parsing " + o.first + ":" +
                                     std::string(e.what()));
    }
  }
  return Status::OK();
}

Status GetColumnFamilyOptionsFromMap(
    const ColumnFamilyOptions& base_options,
    const std::unordered_map<std::string, std::string>& opts_map,
    ColumnFamilyOptions* new_options) {
  assert(new_options);
  *new_options = base_options;
  for (const auto& o : opts_map) {
    if (!ParseColumnFamilyOption(o.first, o.second, new_options)) {
      return Status::InvalidArgument("Can't parse option " + o.first);
    }
  }
  return Status::OK();
}

Status GetColumnFamilyOptionsFromString(
    const ColumnFamilyOptions& base_options,
    const std::string& opts_str,
    ColumnFamilyOptions* new_options) {
  std::unordered_map<std::string, std::string> opts_map;
  Status s = StringToMap(opts_str, &opts_map);
  if (!s.ok()) {
    return s;
  }
  return GetColumnFamilyOptionsFromMap(base_options, opts_map, new_options);
}

Status GetDBOptionsFromMap(
    const DBOptions& base_options,
    const std::unordered_map<std::string, std::string>& opts_map,
    DBOptions* new_options) {
  assert(new_options);
  *new_options = base_options;
  for (const auto& o : opts_map) {
    if (!ParseDBOption(o.first, o.second, new_options)) {
      return Status::InvalidArgument("Can't parse option " + o.first);
    }
  }
  return Status::OK();
}

Status GetDBOptionsFromString(
    const DBOptions& base_options,
    const std::string& opts_str,
    DBOptions* new_options) {
  std::unordered_map<std::string, std::string> opts_map;
  Status s = StringToMap(opts_str, &opts_map);
  if (!s.ok()) {
    return s;
  }
  return GetDBOptionsFromMap(base_options, opts_map, new_options);
}

Status GetOptionsFromString(const Options& base_options,
                            const std::string& opts_str, Options* new_options) {
  std::unordered_map<std::string, std::string> opts_map;
  Status s = StringToMap(opts_str, &opts_map);
  if (!s.ok()) {
    return s;
  }
  DBOptions new_db_options(base_options);
  ColumnFamilyOptions new_cf_options(base_options);
  for (const auto& o : opts_map) {
    if (ParseDBOption(o.first, o.second, &new_db_options)) {
    } else if (ParseColumnFamilyOption(o.first, o.second, &new_cf_options)) {
    } else {
      return Status::InvalidArgument("Can't parse option " + o.first);
    }
  }
  *new_options = Options(new_db_options, new_cf_options);
  return Status::OK();
}

#endif  // ROCKSDB_LITE
}  // namespace rocksdb
