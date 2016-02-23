//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
#ifndef ROCKSDB_LITE
#include "util/ldb_cmd.h"

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <inttypes.h>

#include "db/dbformat.h"
#include "db/db_impl.h"
#include "db/log_reader.h"
#include "db/filename.h"
#include "db/writebuffer.h"
#include "db/write_batch_internal.h"
#include "rocksdb/write_batch.h"
#include "rocksdb/cache.h"
#include "rocksdb/table_properties.h"
#include "port/dirent.h"
#include "util/coding.h"
#include "util/sst_dump_tool_imp.h"
#include "util/string_util.h"
#include "util/scoped_arena_iterator.h"
#include "utilities/ttl/db_ttl_impl.h"

#include <cstdlib>
#include <ctime>
#include <limits>
#include <sstream>
#include <string>
#include <stdexcept>

namespace rocksdb {

using namespace std;

const string LDBCommand::ARG_DB = "db";
const string LDBCommand::ARG_HEX = "hex";
const string LDBCommand::ARG_KEY_HEX = "key_hex";
const string LDBCommand::ARG_VALUE_HEX = "value_hex";
const string LDBCommand::ARG_TTL = "ttl";
const string LDBCommand::ARG_TTL_START = "start_time";
const string LDBCommand::ARG_TTL_END = "end_time";
const string LDBCommand::ARG_TIMESTAMP = "timestamp";
const string LDBCommand::ARG_FROM = "from";
const string LDBCommand::ARG_TO = "to";
const string LDBCommand::ARG_MAX_KEYS = "max_keys";
const string LDBCommand::ARG_BLOOM_BITS = "bloom_bits";
const string LDBCommand::ARG_FIX_PREFIX_LEN = "fix_prefix_len";
const string LDBCommand::ARG_COMPRESSION_TYPE = "compression_type";
const string LDBCommand::ARG_BLOCK_SIZE = "block_size";
const string LDBCommand::ARG_AUTO_COMPACTION = "auto_compaction";
const string LDBCommand::ARG_DB_WRITE_BUFFER_SIZE = "db_write_buffer_size";
const string LDBCommand::ARG_WRITE_BUFFER_SIZE = "write_buffer_size";
const string LDBCommand::ARG_FILE_SIZE = "file_size";
const string LDBCommand::ARG_CREATE_IF_MISSING = "create_if_missing";

const char* LDBCommand::DELIM = " ==> ";

LDBCommand* LDBCommand::InitFromCmdLineArgs(
  int argc,
  char** argv,
  const Options& options,
  const LDBOptions& ldb_options
) {
  vector<string> args;
  for (int i = 1; i < argc; i++) {
    args.push_back(argv[i]);
  }
  return InitFromCmdLineArgs(args, options, ldb_options);
}

/**
 * Parse the command-line arguments and create the appropriate LDBCommand2
 * instance.
 * The command line arguments must be in the following format:
 * ./ldb --db=PATH_TO_DB [--commonOpt1=commonOpt1Val] ..
 *        COMMAND <PARAM1> <PARAM2> ... [-cmdSpecificOpt1=cmdSpecificOpt1Val] ..
 * This is similar to the command line format used by HBaseClientTool.
 * Command name is not included in args.
 * Returns nullptr if the command-line cannot be parsed.
 */
LDBCommand* LDBCommand::InitFromCmdLineArgs(
  const vector<string>& args,
  const Options& options,
  const LDBOptions& ldb_options
) {
  // --x=y command line arguments are added as x->y map entries.
  map<string, string> option_map;

  // Command-line arguments of the form --hex end up in this array as hex
  vector<string> flags;

  // Everything other than option_map and flags. Represents commands
  // and their parameters.  For eg: put key1 value1 go into this vector.
  vector<string> cmdTokens;

  const string OPTION_PREFIX = "--";

  for (const auto& arg : args) {
    if (arg[0] == '-' && arg[1] == '-'){
      vector<string> splits = StringSplit(arg, '=');
      if (splits.size() == 2) {
        string optionKey = splits[0].substr(OPTION_PREFIX.size());
        option_map[optionKey] = splits[1];
      } else {
        string optionKey = splits[0].substr(OPTION_PREFIX.size());
        flags.push_back(optionKey);
      }
    } else {
      cmdTokens.push_back(arg);
    }
  }

  if (cmdTokens.size() < 1) {
    fprintf(stderr, "Command not specified!");
    return nullptr;
  }

  string cmd = cmdTokens[0];
  vector<string> cmdParams(cmdTokens.begin()+1, cmdTokens.end());
  LDBCommand* command = LDBCommand::SelectCommand(
    cmd,
    cmdParams,
    option_map,
    flags
  );

  if (command) {
    command->SetDBOptions(options);
    command->SetLDBOptions(ldb_options);
  }
  return command;
}

LDBCommand* LDBCommand::SelectCommand(
    const std::string& cmd,
    const vector<string>& cmdParams,
    const map<string, string>& option_map,
    const vector<string>& flags
  ) {

  if (cmd == GetCommand::Name()) {
    return new GetCommand(cmdParams, option_map, flags);
  } else if (cmd == PutCommand::Name()) {
    return new PutCommand(cmdParams, option_map, flags);
  } else if (cmd == BatchPutCommand::Name()) {
    return new BatchPutCommand(cmdParams, option_map, flags);
  } else if (cmd == ScanCommand::Name()) {
    return new ScanCommand(cmdParams, option_map, flags);
  } else if (cmd == DeleteCommand::Name()) {
    return new DeleteCommand(cmdParams, option_map, flags);
  } else if (cmd == ApproxSizeCommand::Name()) {
    return new ApproxSizeCommand(cmdParams, option_map, flags);
  } else if (cmd == DBQuerierCommand::Name()) {
    return new DBQuerierCommand(cmdParams, option_map, flags);
  } else if (cmd == CompactorCommand::Name()) {
    return new CompactorCommand(cmdParams, option_map, flags);
  } else if (cmd == WALDumperCommand::Name()) {
    return new WALDumperCommand(cmdParams, option_map, flags);
  } else if (cmd == ReduceDBLevelsCommand::Name()) {
    return new ReduceDBLevelsCommand(cmdParams, option_map, flags);
  } else if (cmd == ChangeCompactionStyleCommand::Name()) {
    return new ChangeCompactionStyleCommand(cmdParams, option_map, flags);
  } else if (cmd == DBDumperCommand::Name()) {
    return new DBDumperCommand(cmdParams, option_map, flags);
  } else if (cmd == DBLoaderCommand::Name()) {
    return new DBLoaderCommand(cmdParams, option_map, flags);
  } else if (cmd == ManifestDumpCommand::Name()) {
    return new ManifestDumpCommand(cmdParams, option_map, flags);
  } else if (cmd == ListColumnFamiliesCommand::Name()) {
    return new ListColumnFamiliesCommand(cmdParams, option_map, flags);
  } else if (cmd == DBFileDumperCommand::Name()) {
    return new DBFileDumperCommand(cmdParams, option_map, flags);
  } else if (cmd == InternalDumpCommand::Name()) {
    return new InternalDumpCommand(cmdParams, option_map, flags);
  } else if (cmd == CheckConsistencyCommand::Name()) {
    return new CheckConsistencyCommand(cmdParams, option_map, flags);
  }
  return nullptr;
}


/**
 * Parses the specific integer option and fills in the value.
 * Returns true if the option is found.
 * Returns false if the option is not found or if there is an error parsing the
 * value.  If there is an error, the specified exec_state is also
 * updated.
 */
bool LDBCommand::ParseIntOption(const map<string, string>& options,
                                const string& option, int& value,
                                LDBCommandExecuteResult& exec_state) {

  map<string, string>::const_iterator itr = option_map_.find(option);
  if (itr != option_map_.end()) {
    try {
#if defined(CYGWIN)
      value = strtol(itr->second.c_str(), 0, 10);
#else
      value = stoi(itr->second);
#endif
      return true;
    } catch(const invalid_argument&) {
      exec_state =
          LDBCommandExecuteResult::Failed(option + " has an invalid value.");
    } catch(const out_of_range&) {
      exec_state = LDBCommandExecuteResult::Failed(
          option + " has a value out-of-range.");
    }
  }
  return false;
}

/**
 * Parses the specified option and fills in the value.
 * Returns true if the option is found.
 * Returns false otherwise.
 */
bool LDBCommand::ParseStringOption(const map<string, string>& options,
                                   const string& option, string* value) {
  auto itr = option_map_.find(option);
  if (itr != option_map_.end()) {
    *value = itr->second;
    return true;
  }
  return false;
}

Options LDBCommand::PrepareOptionsForOpenDB() {

  Options opt = options_;
  opt.create_if_missing = false;

  map<string, string>::const_iterator itr;

  BlockBasedTableOptions table_options;
  bool use_table_options = false;
  int bits;
  if (ParseIntOption(option_map_, ARG_BLOOM_BITS, bits, exec_state_)) {
    if (bits > 0) {
      use_table_options = true;
      table_options.filter_policy.reset(NewBloomFilterPolicy(bits));
    } else {
      exec_state_ =
          LDBCommandExecuteResult::Failed(ARG_BLOOM_BITS + " must be > 0.");
    }
  }

  int block_size;
  if (ParseIntOption(option_map_, ARG_BLOCK_SIZE, block_size, exec_state_)) {
    if (block_size > 0) {
      use_table_options = true;
      table_options.block_size = block_size;
    } else {
      exec_state_ =
          LDBCommandExecuteResult::Failed(ARG_BLOCK_SIZE + " must be > 0.");
    }
  }

  if (use_table_options) {
    opt.table_factory.reset(NewBlockBasedTableFactory(table_options));
  }

  itr = option_map_.find(ARG_AUTO_COMPACTION);
  if (itr != option_map_.end()) {
    opt.disable_auto_compactions = ! StringToBool(itr->second);
  }

  itr = option_map_.find(ARG_COMPRESSION_TYPE);
  if (itr != option_map_.end()) {
    string comp = itr->second;
    if (comp == "no") {
      opt.compression = kNoCompression;
    } else if (comp == "snappy") {
      opt.compression = kSnappyCompression;
    } else if (comp == "zlib") {
      opt.compression = kZlibCompression;
    } else if (comp == "bzip2") {
      opt.compression = kBZip2Compression;
    } else if (comp == "lz4") {
      opt.compression = kLZ4Compression;
    } else if (comp == "lz4hc") {
      opt.compression = kLZ4HCCompression;
    } else if (comp == "zstd") {
      opt.compression = kZSTDNotFinalCompression;
    } else {
      // Unknown compression.
      exec_state_ =
          LDBCommandExecuteResult::Failed("Unknown compression level: " + comp);
    }
  }

  int db_write_buffer_size;
  if (ParseIntOption(option_map_, ARG_DB_WRITE_BUFFER_SIZE,
        db_write_buffer_size, exec_state_)) {
    if (db_write_buffer_size >= 0) {
      opt.db_write_buffer_size = db_write_buffer_size;
    } else {
      exec_state_ = LDBCommandExecuteResult::Failed(ARG_DB_WRITE_BUFFER_SIZE +
                                                    " must be >= 0.");
    }
  }

  int write_buffer_size;
  if (ParseIntOption(option_map_, ARG_WRITE_BUFFER_SIZE, write_buffer_size,
        exec_state_)) {
    if (write_buffer_size > 0) {
      opt.write_buffer_size = write_buffer_size;
    } else {
      exec_state_ = LDBCommandExecuteResult::Failed(ARG_WRITE_BUFFER_SIZE +
                                                    " must be > 0.");
    }
  }

  int file_size;
  if (ParseIntOption(option_map_, ARG_FILE_SIZE, file_size, exec_state_)) {
    if (file_size > 0) {
      opt.target_file_size_base = file_size;
    } else {
      exec_state_ =
          LDBCommandExecuteResult::Failed(ARG_FILE_SIZE + " must be > 0.");
    }
  }

  if (opt.db_paths.size() == 0) {
    opt.db_paths.emplace_back(db_path_, std::numeric_limits<uint64_t>::max());
  }

  int fix_prefix_len;
  if (ParseIntOption(option_map_, ARG_FIX_PREFIX_LEN, fix_prefix_len,
                     exec_state_)) {
    if (fix_prefix_len > 0) {
      opt.prefix_extractor.reset(
          NewFixedPrefixTransform(static_cast<size_t>(fix_prefix_len)));
    } else {
      exec_state_ =
          LDBCommandExecuteResult::Failed(ARG_FIX_PREFIX_LEN + " must be > 0.");
    }
  }

  return opt;
}

bool LDBCommand::ParseKeyValue(const string& line, string* key, string* value,
                              bool is_key_hex, bool is_value_hex) {
  size_t pos = line.find(DELIM);
  if (pos != string::npos) {
    *key = line.substr(0, pos);
    *value = line.substr(pos + strlen(DELIM));
    if (is_key_hex) {
      *key = HexToString(*key);
    }
    if (is_value_hex) {
      *value = HexToString(*value);
    }
    return true;
  } else {
    return false;
  }
}

/**
 * Make sure that ONLY the command-line options and flags expected by this
 * command are specified on the command-line.  Extraneous options are usually
 * the result of user error.
 * Returns true if all checks pass.  Else returns false, and prints an
 * appropriate error msg to stderr.
 */
bool LDBCommand::ValidateCmdLineOptions() {

  for (map<string, string>::const_iterator itr = option_map_.begin();
        itr != option_map_.end(); ++itr) {
    if (find(valid_cmd_line_options_.begin(),
          valid_cmd_line_options_.end(), itr->first) ==
          valid_cmd_line_options_.end()) {
      fprintf(stderr, "Invalid command-line option %s\n", itr->first.c_str());
      return false;
    }
  }

  for (vector<string>::const_iterator itr = flags_.begin();
        itr != flags_.end(); ++itr) {
    if (find(valid_cmd_line_options_.begin(),
          valid_cmd_line_options_.end(), *itr) ==
          valid_cmd_line_options_.end()) {
      fprintf(stderr, "Invalid command-line flag %s\n", itr->c_str());
      return false;
    }
  }

  if (!NoDBOpen() && option_map_.find(ARG_DB) == option_map_.end()) {
    fprintf(stderr, "%s must be specified\n", ARG_DB.c_str());
    return false;
  }

  return true;
}

CompactorCommand::CompactorCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags) :
    LDBCommand(options, flags, false,
               BuildCmdLineOptions({ARG_FROM, ARG_TO, ARG_HEX, ARG_KEY_HEX,
                                    ARG_VALUE_HEX, ARG_TTL})),
    null_from_(true), null_to_(true) {

  map<string, string>::const_iterator itr = options.find(ARG_FROM);
  if (itr != options.end()) {
    null_from_ = false;
    from_ = itr->second;
  }

  itr = options.find(ARG_TO);
  if (itr != options.end()) {
    null_to_ = false;
    to_ = itr->second;
  }

  if (is_key_hex_) {
    if (!null_from_) {
      from_ = HexToString(from_);
    }
    if (!null_to_) {
      to_ = HexToString(to_);
    }
  }
}

void CompactorCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(CompactorCommand::Name());
  ret.append(HelpRangeCmdArgs());
  ret.append("\n");
}

void CompactorCommand::DoCommand() {

  Slice* begin = nullptr;
  Slice* end = nullptr;
  if (!null_from_) {
    begin = new Slice(from_);
  }
  if (!null_to_) {
    end = new Slice(to_);
  }

  db_->CompactRange(CompactRangeOptions(), begin, end);
  exec_state_ = LDBCommandExecuteResult::Succeed("");

  delete begin;
  delete end;
}

// ----------------------------------------------------------------------------

const string DBLoaderCommand::ARG_DISABLE_WAL = "disable_wal";
const string DBLoaderCommand::ARG_BULK_LOAD = "bulk_load";
const string DBLoaderCommand::ARG_COMPACT = "compact";

DBLoaderCommand::DBLoaderCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags) :
    LDBCommand(options, flags, false,
               BuildCmdLineOptions({ARG_HEX, ARG_KEY_HEX, ARG_VALUE_HEX,
                                    ARG_FROM, ARG_TO, ARG_CREATE_IF_MISSING,
                                    ARG_DISABLE_WAL, ARG_BULK_LOAD,
                                    ARG_COMPACT})),
    create_if_missing_(false), disable_wal_(false), bulk_load_(false),
    compact_(false) {

  create_if_missing_ = IsFlagPresent(flags, ARG_CREATE_IF_MISSING);
  disable_wal_ = IsFlagPresent(flags, ARG_DISABLE_WAL);
  bulk_load_ = IsFlagPresent(flags, ARG_BULK_LOAD);
  compact_ = IsFlagPresent(flags, ARG_COMPACT);
}

void DBLoaderCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(DBLoaderCommand::Name());
  ret.append(" [--" + ARG_CREATE_IF_MISSING + "]");
  ret.append(" [--" + ARG_DISABLE_WAL + "]");
  ret.append(" [--" + ARG_BULK_LOAD + "]");
  ret.append(" [--" + ARG_COMPACT + "]");
  ret.append("\n");
}

Options DBLoaderCommand::PrepareOptionsForOpenDB() {
  Options opt = LDBCommand::PrepareOptionsForOpenDB();
  opt.create_if_missing = create_if_missing_;
  if (bulk_load_) {
    opt.PrepareForBulkLoad();
  }
  return opt;
}

void DBLoaderCommand::DoCommand() {
  if (!db_) {
    return;
  }

  WriteOptions write_options;
  if (disable_wal_) {
    write_options.disableWAL = true;
  }

  int bad_lines = 0;
  string line;
  while (getline(cin, line, '\n')) {
    string key;
    string value;
    if (ParseKeyValue(line, &key, &value, is_key_hex_, is_value_hex_)) {
      db_->Put(write_options, Slice(key), Slice(value));
    } else if (0 == line.find("Keys in range:")) {
      // ignore this line
    } else if (0 == line.find("Created bg thread 0x")) {
      // ignore this line
    } else {
      bad_lines ++;
    }
  }

  if (bad_lines > 0) {
    cout << "Warning: " << bad_lines << " bad lines ignored." << endl;
  }
  if (compact_) {
    db_->CompactRange(CompactRangeOptions(), nullptr, nullptr);
  }
}

// ----------------------------------------------------------------------------

namespace {

void DumpManifestFile(std::string file, bool verbose, bool hex, bool json) {
  Options options;
  EnvOptions sopt;
  std::string dbname("dummy");
  std::shared_ptr<Cache> tc(NewLRUCache(options.max_open_files - 10,
                                        options.table_cache_numshardbits));
  // Notice we are using the default options not through SanitizeOptions(),
  // if VersionSet::DumpManifest() depends on any option done by
  // SanitizeOptions(), we need to initialize it manually.
  options.db_paths.emplace_back("dummy", 0);
  options.num_levels = 64;
  WriteController wc(options.delayed_write_rate);
  WriteBuffer wb(options.db_write_buffer_size);
  VersionSet versions(dbname, &options, sopt, tc.get(), &wb, &wc);
  Status s = versions.DumpManifest(options, file, verbose, hex, json);
  if (!s.ok()) {
    printf("Error in processing file %s %s\n", file.c_str(),
           s.ToString().c_str());
  }
}

}  // namespace

const string ManifestDumpCommand::ARG_VERBOSE = "verbose";
const string ManifestDumpCommand::ARG_JSON = "json";
const string ManifestDumpCommand::ARG_PATH = "path";

void ManifestDumpCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(ManifestDumpCommand::Name());
  ret.append(" [--" + ARG_VERBOSE + "]");
  ret.append(" [--" + ARG_JSON + "]");
  ret.append(" [--" + ARG_PATH + "=<path_to_manifest_file>]");
  ret.append("\n");
}

ManifestDumpCommand::ManifestDumpCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags) :
    LDBCommand(options, flags, false,
               BuildCmdLineOptions({ARG_VERBOSE, ARG_PATH, ARG_HEX, ARG_JSON})),
    verbose_(false),
    json_(false),
    path_("")
{
  verbose_ = IsFlagPresent(flags, ARG_VERBOSE);
  json_ = IsFlagPresent(flags, ARG_JSON);

  map<string, string>::const_iterator itr = options.find(ARG_PATH);
  if (itr != options.end()) {
    path_ = itr->second;
    if (path_.empty()) {
      exec_state_ = LDBCommandExecuteResult::Failed("--path: missing pathname");
    }
  }
}

void ManifestDumpCommand::DoCommand() {

  std::string manifestfile;

  if (!path_.empty()) {
    manifestfile = path_;
  } else {
    bool found = false;
    // We need to find the manifest file by searching the directory
    // containing the db for files of the form MANIFEST_[0-9]+

    auto CloseDir = [](DIR* p) { closedir(p); };
    std::unique_ptr<DIR, decltype(CloseDir)> d(opendir(db_path_.c_str()),
                                               CloseDir);

    if (d == nullptr) {
      exec_state_ =
          LDBCommandExecuteResult::Failed(db_path_ + " is not a directory");
      return;
    }
    struct dirent* entry;
    while ((entry = readdir(d.get())) != nullptr) {
      unsigned int match;
      uint64_t num;
      if (sscanf(entry->d_name, "MANIFEST-%" PRIu64 "%n", &num, &match) &&
          match == strlen(entry->d_name)) {
        if (!found) {
          manifestfile = db_path_ + "/" + std::string(entry->d_name);
          found = true;
        } else {
          exec_state_ = LDBCommandExecuteResult::Failed(
              "Multiple MANIFEST files found; use --path to select one");
          return;
        }
      }
    }
  }

  if (verbose_) {
    printf("Processing Manifest file %s\n", manifestfile.c_str());
  }

  DumpManifestFile(manifestfile, verbose_, is_key_hex_, json_);

  if (verbose_) {
    printf("Processing Manifest file %s done\n", manifestfile.c_str());
  }
}

// ----------------------------------------------------------------------------

void ListColumnFamiliesCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(ListColumnFamiliesCommand::Name());
  ret.append(" full_path_to_db_directory ");
  ret.append("\n");
}

ListColumnFamiliesCommand::ListColumnFamiliesCommand(
    const vector<string>& params, const map<string, string>& options,
    const vector<string>& flags)
    : LDBCommand(options, flags, false, {}) {

  if (params.size() != 1) {
    exec_state_ = LDBCommandExecuteResult::Failed(
        "dbname must be specified for the list_column_families command");
  } else {
    dbname_ = params[0];
  }
}

void ListColumnFamiliesCommand::DoCommand() {
  vector<string> column_families;
  Status s = DB::ListColumnFamilies(DBOptions(), dbname_, &column_families);
  if (!s.ok()) {
    printf("Error in processing db %s %s\n", dbname_.c_str(),
           s.ToString().c_str());
  } else {
    printf("Column families in %s: \n{", dbname_.c_str());
    bool first = true;
    for (auto cf : column_families) {
      if (!first) {
        printf(", ");
      }
      first = false;
      printf("%s", cf.c_str());
    }
    printf("}\n");
  }
}

// ----------------------------------------------------------------------------

namespace {

string ReadableTime(int unixtime) {
  char time_buffer [80];
  time_t rawtime = unixtime;
  struct tm tInfo;
  struct tm* timeinfo = localtime_r(&rawtime, &tInfo);
  assert(timeinfo == &tInfo);
  strftime(time_buffer, 80, "%c", timeinfo);
  return string(time_buffer);
}

// This function only called when it's the sane case of >1 buckets in time-range
// Also called only when timekv falls between ttl_start and ttl_end provided
void IncBucketCounts(vector<uint64_t>& bucket_counts, int ttl_start,
      int time_range, int bucket_size, int timekv, int num_buckets) {
  assert(time_range > 0 && timekv >= ttl_start && bucket_size > 0 &&
    timekv < (ttl_start + time_range) && num_buckets > 1);
  int bucket = (timekv - ttl_start) / bucket_size;
  bucket_counts[bucket]++;
}

void PrintBucketCounts(const vector<uint64_t>& bucket_counts, int ttl_start,
      int ttl_end, int bucket_size, int num_buckets) {
  int time_point = ttl_start;
  for(int i = 0; i < num_buckets - 1; i++, time_point += bucket_size) {
    fprintf(stdout, "Keys in range %s to %s : %lu\n",
            ReadableTime(time_point).c_str(),
            ReadableTime(time_point + bucket_size).c_str(),
            (unsigned long)bucket_counts[i]);
  }
  fprintf(stdout, "Keys in range %s to %s : %lu\n",
          ReadableTime(time_point).c_str(),
          ReadableTime(ttl_end).c_str(),
          (unsigned long)bucket_counts[num_buckets - 1]);
}

}  // namespace

const string InternalDumpCommand::ARG_COUNT_ONLY = "count_only";
const string InternalDumpCommand::ARG_COUNT_DELIM = "count_delim";
const string InternalDumpCommand::ARG_STATS = "stats";
const string InternalDumpCommand::ARG_INPUT_KEY_HEX = "input_key_hex";

InternalDumpCommand::InternalDumpCommand(const vector<string>& params,
                                         const map<string, string>& options,
                                         const vector<string>& flags) :
    LDBCommand(options, flags, true,
               BuildCmdLineOptions({ ARG_HEX, ARG_KEY_HEX, ARG_VALUE_HEX,
                                     ARG_FROM, ARG_TO, ARG_MAX_KEYS,
                                     ARG_COUNT_ONLY, ARG_COUNT_DELIM, ARG_STATS,
                                     ARG_INPUT_KEY_HEX})),
    has_from_(false),
    has_to_(false),
    max_keys_(-1),
    delim_("."),
    count_only_(false),
    count_delim_(false),
    print_stats_(false),
    is_input_key_hex_(false) {

  has_from_ = ParseStringOption(options, ARG_FROM, &from_);
  has_to_ = ParseStringOption(options, ARG_TO, &to_);

  ParseIntOption(options, ARG_MAX_KEYS, max_keys_, exec_state_);
  map<string, string>::const_iterator itr = options.find(ARG_COUNT_DELIM);
  if (itr != options.end()) {
    delim_ = itr->second;
    count_delim_ = true;
   // fprintf(stdout,"delim = %c\n",delim_[0]);
  } else {
    count_delim_ = IsFlagPresent(flags, ARG_COUNT_DELIM);
    delim_=".";
  }

  print_stats_ = IsFlagPresent(flags, ARG_STATS);
  count_only_ = IsFlagPresent(flags, ARG_COUNT_ONLY);
  is_input_key_hex_ = IsFlagPresent(flags, ARG_INPUT_KEY_HEX);

  if (is_input_key_hex_) {
    if (has_from_) {
      from_ = HexToString(from_);
    }
    if (has_to_) {
      to_ = HexToString(to_);
    }
  }
}

void InternalDumpCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(InternalDumpCommand::Name());
  ret.append(HelpRangeCmdArgs());
  ret.append(" [--" + ARG_INPUT_KEY_HEX + "]");
  ret.append(" [--" + ARG_MAX_KEYS + "=<N>]");
  ret.append(" [--" + ARG_COUNT_ONLY + "]");
  ret.append(" [--" + ARG_COUNT_DELIM + "=<char>]");
  ret.append(" [--" + ARG_STATS + "]");
  ret.append("\n");
}

void InternalDumpCommand::DoCommand() {
  if (!db_) {
    return;
  }

  if (print_stats_) {
    string stats;
    if (db_->GetProperty("rocksdb.stats", &stats)) {
      fprintf(stdout, "%s\n", stats.c_str());
    }
  }

  // Cast as DBImpl to get internal iterator
  DBImpl* idb = dynamic_cast<DBImpl*>(db_);
  if (!idb) {
    exec_state_ = LDBCommandExecuteResult::Failed("DB is not DBImpl");
    return;
  }
  string rtype1,rtype2,row,val;
  rtype2 = "";
  uint64_t c=0;
  uint64_t s1=0,s2=0;
  // Setup internal key iterator
  Arena arena;
  ScopedArenaIterator iter(idb->TEST_NewInternalIterator(&arena));
  Status st = iter->status();
  if (!st.ok()) {
    exec_state_ =
        LDBCommandExecuteResult::Failed("Iterator error:" + st.ToString());
  }

  if (has_from_) {
    InternalKey ikey;
    ikey.SetMaxPossibleForUserKey(from_);
    iter->Seek(ikey.Encode());
  } else {
    iter->SeekToFirst();
  }

  long long count = 0;
  for (; iter->Valid(); iter->Next()) {
    ParsedInternalKey ikey;
    if (!ParseInternalKey(iter->key(), &ikey)) {
      fprintf(stderr, "Internal Key [%s] parse error!\n",
              iter->key().ToString(true /* in hex*/).data());
      // TODO: add error counter
      continue;
    }

    // If end marker was specified, we stop before it
    if (has_to_ && options_.comparator->Compare(ikey.user_key, to_) >= 0) {
      break;
    }

    ++count;
    int k;
    if (count_delim_) {
      rtype1 = "";
      s1=0;
      row = iter->key().ToString();
      val = iter->value().ToString();
      for(k=0;row[k]!='\x01' && row[k]!='\0';k++)
        s1++;
      for(k=0;val[k]!='\x01' && val[k]!='\0';k++)
        s1++;
      for(int j=0;row[j]!=delim_[0] && row[j]!='\0' && row[j]!='\x01';j++)
        rtype1+=row[j];
      if(rtype2.compare("") && rtype2.compare(rtype1)!=0) {
        fprintf(stdout,"%s => count:%lld\tsize:%lld\n",rtype2.c_str(),
            (long long)c,(long long)s2);
        c=1;
        s2=s1;
        rtype2 = rtype1;
      } else {
        c++;
        s2+=s1;
        rtype2=rtype1;
    }
  }

    if (!count_only_ && !count_delim_) {
      string key = ikey.DebugString(is_key_hex_);
      string value = iter->value().ToString(is_value_hex_);
      std::cout << key << " => " << value << "\n";
    }

    // Terminate if maximum number of keys have been dumped
    if (max_keys_ > 0 && count >= max_keys_) break;
  }
  if(count_delim_) {
    fprintf(stdout,"%s => count:%lld\tsize:%lld\n", rtype2.c_str(),
        (long long)c,(long long)s2);
  } else
  fprintf(stdout, "Internal keys in range: %lld\n", (long long) count);
}


const string DBDumperCommand::ARG_COUNT_ONLY = "count_only";
const string DBDumperCommand::ARG_COUNT_DELIM = "count_delim";
const string DBDumperCommand::ARG_STATS = "stats";
const string DBDumperCommand::ARG_TTL_BUCKET = "bucket";

DBDumperCommand::DBDumperCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags) :
    LDBCommand(options, flags, true,
               BuildCmdLineOptions({ARG_TTL, ARG_HEX, ARG_KEY_HEX,
                                    ARG_VALUE_HEX, ARG_FROM, ARG_TO,
                                    ARG_MAX_KEYS, ARG_COUNT_ONLY,
                                    ARG_COUNT_DELIM, ARG_STATS, ARG_TTL_START,
                                    ARG_TTL_END, ARG_TTL_BUCKET,
                                    ARG_TIMESTAMP})),
    null_from_(true),
    null_to_(true),
    max_keys_(-1),
    count_only_(false),
    count_delim_(false),
    print_stats_(false) {

  map<string, string>::const_iterator itr = options.find(ARG_FROM);
  if (itr != options.end()) {
    null_from_ = false;
    from_ = itr->second;
  }

  itr = options.find(ARG_TO);
  if (itr != options.end()) {
    null_to_ = false;
    to_ = itr->second;
  }

  itr = options.find(ARG_MAX_KEYS);
  if (itr != options.end()) {
    try {
#if defined(CYGWIN)
      max_keys_ = strtol(itr->second.c_str(), 0, 10);
#else
      max_keys_ = stoi(itr->second);
#endif
    } catch(const invalid_argument&) {
      exec_state_ = LDBCommandExecuteResult::Failed(ARG_MAX_KEYS +
                                                    " has an invalid value");
    } catch(const out_of_range&) {
      exec_state_ = LDBCommandExecuteResult::Failed(
          ARG_MAX_KEYS + " has a value out-of-range");
    }
  }
  itr = options.find(ARG_COUNT_DELIM);
  if (itr != options.end()) {
    delim_ = itr->second;
    count_delim_ = true;
  } else {
    count_delim_ = IsFlagPresent(flags, ARG_COUNT_DELIM);
    delim_=".";
  }

  print_stats_ = IsFlagPresent(flags, ARG_STATS);
  count_only_ = IsFlagPresent(flags, ARG_COUNT_ONLY);

  if (is_key_hex_) {
    if (!null_from_) {
      from_ = HexToString(from_);
    }
    if (!null_to_) {
      to_ = HexToString(to_);
    }
  }
}

void DBDumperCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(DBDumperCommand::Name());
  ret.append(HelpRangeCmdArgs());
  ret.append(" [--" + ARG_TTL + "]");
  ret.append(" [--" + ARG_MAX_KEYS + "=<N>]");
  ret.append(" [--" + ARG_TIMESTAMP + "]");
  ret.append(" [--" + ARG_COUNT_ONLY + "]");
  ret.append(" [--" + ARG_COUNT_DELIM + "=<char>]");
  ret.append(" [--" + ARG_STATS + "]");
  ret.append(" [--" + ARG_TTL_BUCKET + "=<N>]");
  ret.append(" [--" + ARG_TTL_START + "=<N>:- is inclusive]");
  ret.append(" [--" + ARG_TTL_END + "=<N>:- is exclusive]");
  ret.append("\n");
}

void DBDumperCommand::DoCommand() {
  if (!db_) {
    return;
  }
  // Parse command line args
  uint64_t count = 0;
  if (print_stats_) {
    string stats;
    if (db_->GetProperty("rocksdb.stats", &stats)) {
      fprintf(stdout, "%s\n", stats.c_str());
    }
  }

  // Setup key iterator
  Iterator* iter = db_->NewIterator(ReadOptions());
  Status st = iter->status();
  if (!st.ok()) {
    exec_state_ =
        LDBCommandExecuteResult::Failed("Iterator error." + st.ToString());
  }

  if (!null_from_) {
    iter->Seek(from_);
  } else {
    iter->SeekToFirst();
  }

  int max_keys = max_keys_;
  int ttl_start;
  if (!ParseIntOption(option_map_, ARG_TTL_START, ttl_start, exec_state_)) {
    ttl_start = DBWithTTLImpl::kMinTimestamp;  // TTL introduction time
  }
  int ttl_end;
  if (!ParseIntOption(option_map_, ARG_TTL_END, ttl_end, exec_state_)) {
    ttl_end = DBWithTTLImpl::kMaxTimestamp;  // Max time allowed by TTL feature
  }
  if (ttl_end < ttl_start) {
    fprintf(stderr, "Error: End time can't be less than start time\n");
    delete iter;
    return;
  }
  int time_range = ttl_end - ttl_start;
  int bucket_size;
  if (!ParseIntOption(option_map_, ARG_TTL_BUCKET, bucket_size, exec_state_) ||
      bucket_size <= 0) {
    bucket_size = time_range; // Will have just 1 bucket by default
  }
  //cretaing variables for row count of each type
  string rtype1,rtype2,row,val;
  rtype2 = "";
  uint64_t c=0;
  uint64_t s1=0,s2=0;

  // At this point, bucket_size=0 => time_range=0
  int num_buckets = (bucket_size >= time_range)
                        ? 1
                        : ((time_range + bucket_size - 1) / bucket_size);
  vector<uint64_t> bucket_counts(num_buckets, 0);
  if (is_db_ttl_ && !count_only_ && timestamp_ && !count_delim_) {
    fprintf(stdout, "Dumping key-values from %s to %s\n",
            ReadableTime(ttl_start).c_str(), ReadableTime(ttl_end).c_str());
  }

  for (; iter->Valid(); iter->Next()) {
    int rawtime = 0;
    // If end marker was specified, we stop before it
    if (!null_to_ && (iter->key().ToString() >= to_))
      break;
    // Terminate if maximum number of keys have been dumped
    if (max_keys == 0)
      break;
    if (is_db_ttl_) {
      TtlIterator* it_ttl = dynamic_cast<TtlIterator*>(iter);
      assert(it_ttl);
      rawtime = it_ttl->timestamp();
      if (rawtime < ttl_start || rawtime >= ttl_end) {
        continue;
      }
    }
    if (max_keys > 0) {
      --max_keys;
    }
    if (is_db_ttl_ && num_buckets > 1) {
      IncBucketCounts(bucket_counts, ttl_start, time_range, bucket_size,
                      rawtime, num_buckets);
    }
    ++count;
    if (count_delim_) {
      rtype1 = "";
      row = iter->key().ToString();
      val = iter->value().ToString();
      s1 = row.size()+val.size();
      for(int j=0;row[j]!=delim_[0] && row[j]!='\0';j++)
        rtype1+=row[j];
      if(rtype2.compare("") && rtype2.compare(rtype1)!=0) {
        fprintf(stdout,"%s => count:%lld\tsize:%lld\n",rtype2.c_str(),
            (long long )c,(long long)s2);
        c=1;
        s2=s1;
        rtype2 = rtype1;
      } else {
          c++;
          s2+=s1;
          rtype2=rtype1;
      }

    }



    if (!count_only_ && !count_delim_) {
      if (is_db_ttl_ && timestamp_) {
        fprintf(stdout, "%s ", ReadableTime(rawtime).c_str());
      }
      string str = PrintKeyValue(iter->key().ToString(),
                                 iter->value().ToString(), is_key_hex_,
                                 is_value_hex_);
      fprintf(stdout, "%s\n", str.c_str());
    }
  }

  if (num_buckets > 1 && is_db_ttl_) {
    PrintBucketCounts(bucket_counts, ttl_start, ttl_end, bucket_size,
                      num_buckets);
  } else if(count_delim_) {
    fprintf(stdout,"%s => count:%lld\tsize:%lld\n",rtype2.c_str(),
        (long long )c,(long long)s2);
  } else {
    fprintf(stdout, "Keys in range: %lld\n", (long long) count);
  }
  // Clean up
  delete iter;
}

const string ReduceDBLevelsCommand::ARG_NEW_LEVELS = "new_levels";
const string  ReduceDBLevelsCommand::ARG_PRINT_OLD_LEVELS = "print_old_levels";

ReduceDBLevelsCommand::ReduceDBLevelsCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags) :
    LDBCommand(options, flags, false,
               BuildCmdLineOptions({ARG_NEW_LEVELS, ARG_PRINT_OLD_LEVELS})),
    old_levels_(1 << 7),
    new_levels_(-1),
    print_old_levels_(false) {


  ParseIntOption(option_map_, ARG_NEW_LEVELS, new_levels_, exec_state_);
  print_old_levels_ = IsFlagPresent(flags, ARG_PRINT_OLD_LEVELS);

  if(new_levels_ <= 0) {
    exec_state_ = LDBCommandExecuteResult::Failed(
        " Use --" + ARG_NEW_LEVELS + " to specify a new level number\n");
  }
}

vector<string> ReduceDBLevelsCommand::PrepareArgs(const string& db_path,
    int new_levels, bool print_old_level) {
  vector<string> ret;
  ret.push_back("reduce_levels");
  ret.push_back("--" + ARG_DB + "=" + db_path);
  ret.push_back("--" + ARG_NEW_LEVELS + "=" + rocksdb::ToString(new_levels));
  if(print_old_level) {
    ret.push_back("--" + ARG_PRINT_OLD_LEVELS);
  }
  return ret;
}

void ReduceDBLevelsCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(ReduceDBLevelsCommand::Name());
  ret.append(" --" + ARG_NEW_LEVELS + "=<New number of levels>");
  ret.append(" [--" + ARG_PRINT_OLD_LEVELS + "]");
  ret.append("\n");
}

Options ReduceDBLevelsCommand::PrepareOptionsForOpenDB() {
  Options opt = LDBCommand::PrepareOptionsForOpenDB();
  opt.num_levels = old_levels_;
  opt.max_bytes_for_level_multiplier_additional.resize(opt.num_levels, 1);
  // Disable size compaction
  opt.max_bytes_for_level_base = 1ULL << 50;
  opt.max_bytes_for_level_multiplier = 1;
  return opt;
}

Status ReduceDBLevelsCommand::GetOldNumOfLevels(Options& opt,
    int* levels) {
  EnvOptions soptions;
  std::shared_ptr<Cache> tc(
      NewLRUCache(opt.max_open_files - 10, opt.table_cache_numshardbits));
  const InternalKeyComparator cmp(opt.comparator);
  WriteController wc(opt.delayed_write_rate);
  WriteBuffer wb(opt.db_write_buffer_size);
  VersionSet versions(db_path_, &opt, soptions, tc.get(), &wb, &wc);
  std::vector<ColumnFamilyDescriptor> dummy;
  ColumnFamilyDescriptor dummy_descriptor(kDefaultColumnFamilyName,
                                          ColumnFamilyOptions(opt));
  dummy.push_back(dummy_descriptor);
  // We rely the VersionSet::Recover to tell us the internal data structures
  // in the db. And the Recover() should never do any change
  // (like LogAndApply) to the manifest file.
  Status st = versions.Recover(dummy);
  if (!st.ok()) {
    return st;
  }
  int max = -1;
  auto default_cfd = versions.GetColumnFamilySet()->GetDefault();
  for (int i = 0; i < default_cfd->NumberLevels(); i++) {
    if (default_cfd->current()->storage_info()->NumLevelFiles(i)) {
      max = i;
    }
  }

  *levels = max + 1;
  return st;
}

void ReduceDBLevelsCommand::DoCommand() {
  if (new_levels_ <= 1) {
    exec_state_ =
        LDBCommandExecuteResult::Failed("Invalid number of levels.\n");
    return;
  }

  Status st;
  Options opt = PrepareOptionsForOpenDB();
  int old_level_num = -1;
  st = GetOldNumOfLevels(opt, &old_level_num);
  if (!st.ok()) {
    exec_state_ = LDBCommandExecuteResult::Failed(st.ToString());
    return;
  }

  if (print_old_levels_) {
    fprintf(stdout, "The old number of levels in use is %d\n", old_level_num);
  }

  if (old_level_num <= new_levels_) {
    return;
  }

  old_levels_ = old_level_num;

  OpenDB();
  if (!db_) {
    return;
  }
  // Compact the whole DB to put all files to the highest level.
  fprintf(stdout, "Compacting the db...\n");
  db_->CompactRange(CompactRangeOptions(), nullptr, nullptr);
  CloseDB();

  EnvOptions soptions;
  st = VersionSet::ReduceNumberOfLevels(db_path_, &opt, soptions, new_levels_);
  if (!st.ok()) {
    exec_state_ = LDBCommandExecuteResult::Failed(st.ToString());
    return;
  }
}

const string ChangeCompactionStyleCommand::ARG_OLD_COMPACTION_STYLE =
  "old_compaction_style";
const string ChangeCompactionStyleCommand::ARG_NEW_COMPACTION_STYLE =
  "new_compaction_style";

ChangeCompactionStyleCommand::ChangeCompactionStyleCommand(
      const vector<string>& params, const map<string, string>& options,
      const vector<string>& flags) :
    LDBCommand(options, flags, false,
               BuildCmdLineOptions({ARG_OLD_COMPACTION_STYLE,
                                    ARG_NEW_COMPACTION_STYLE})),
    old_compaction_style_(-1),
    new_compaction_style_(-1) {

  ParseIntOption(option_map_, ARG_OLD_COMPACTION_STYLE, old_compaction_style_,
    exec_state_);
  if (old_compaction_style_ != kCompactionStyleLevel &&
     old_compaction_style_ != kCompactionStyleUniversal) {
    exec_state_ = LDBCommandExecuteResult::Failed(
        "Use --" + ARG_OLD_COMPACTION_STYLE + " to specify old compaction " +
        "style. Check ldb help for proper compaction style value.\n");
    return;
  }

  ParseIntOption(option_map_, ARG_NEW_COMPACTION_STYLE, new_compaction_style_,
    exec_state_);
  if (new_compaction_style_ != kCompactionStyleLevel &&
     new_compaction_style_ != kCompactionStyleUniversal) {
    exec_state_ = LDBCommandExecuteResult::Failed(
        "Use --" + ARG_NEW_COMPACTION_STYLE + " to specify new compaction " +
        "style. Check ldb help for proper compaction style value.\n");
    return;
  }

  if (new_compaction_style_ == old_compaction_style_) {
    exec_state_ = LDBCommandExecuteResult::Failed(
        "Old compaction style is the same as new compaction style. "
        "Nothing to do.\n");
    return;
  }

  if (old_compaction_style_ == kCompactionStyleUniversal &&
      new_compaction_style_ == kCompactionStyleLevel) {
    exec_state_ = LDBCommandExecuteResult::Failed(
        "Convert from universal compaction to level compaction. "
        "Nothing to do.\n");
    return;
  }
}

void ChangeCompactionStyleCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(ChangeCompactionStyleCommand::Name());
  ret.append(" --" + ARG_OLD_COMPACTION_STYLE + "=<Old compaction style: 0 " +
             "for level compaction, 1 for universal compaction>");
  ret.append(" --" + ARG_NEW_COMPACTION_STYLE + "=<New compaction style: 0 " +
             "for level compaction, 1 for universal compaction>");
  ret.append("\n");
}

Options ChangeCompactionStyleCommand::PrepareOptionsForOpenDB() {
  Options opt = LDBCommand::PrepareOptionsForOpenDB();

  if (old_compaction_style_ == kCompactionStyleLevel &&
      new_compaction_style_ == kCompactionStyleUniversal) {
    // In order to convert from level compaction to universal compaction, we
    // need to compact all data into a single file and move it to level 0.
    opt.disable_auto_compactions = true;
    opt.target_file_size_base = INT_MAX;
    opt.target_file_size_multiplier = 1;
    opt.max_bytes_for_level_base = INT_MAX;
    opt.max_bytes_for_level_multiplier = 1;
  }

  return opt;
}

void ChangeCompactionStyleCommand::DoCommand() {
  // print db stats before we have made any change
  std::string property;
  std::string files_per_level;
  for (int i = 0; i < db_->NumberLevels(); i++) {
    db_->GetProperty("rocksdb.num-files-at-level" + NumberToString(i),
                     &property);

    // format print string
    char buf[100];
    snprintf(buf, sizeof(buf), "%s%s", (i ? "," : ""), property.c_str());
    files_per_level += buf;
  }
  fprintf(stdout, "files per level before compaction: %s\n",
          files_per_level.c_str());

  // manual compact into a single file and move the file to level 0
  CompactRangeOptions compact_options;
  compact_options.change_level = true;
  compact_options.target_level = 0;
  db_->CompactRange(compact_options, nullptr, nullptr);

  // verify compaction result
  files_per_level = "";
  int num_files = 0;
  for (int i = 0; i < db_->NumberLevels(); i++) {
    db_->GetProperty("rocksdb.num-files-at-level" + NumberToString(i),
                     &property);

    // format print string
    char buf[100];
    snprintf(buf, sizeof(buf), "%s%s", (i ? "," : ""), property.c_str());
    files_per_level += buf;

    num_files = atoi(property.c_str());

    // level 0 should have only 1 file
    if (i == 0 && num_files != 1) {
      exec_state_ = LDBCommandExecuteResult::Failed(
          "Number of db files at "
          "level 0 after compaction is " +
          ToString(num_files) + ", not 1.\n");
      return;
    }
    // other levels should have no file
    if (i > 0 && num_files != 0) {
      exec_state_ = LDBCommandExecuteResult::Failed(
          "Number of db files at "
          "level " +
          ToString(i) + " after compaction is " + ToString(num_files) +
          ", not 0.\n");
      return;
    }
  }

  fprintf(stdout, "files per level after compaction: %s\n",
          files_per_level.c_str());
}

// ----------------------------------------------------------------------------

namespace {

struct StdErrReporter : public log::Reader::Reporter {
  virtual void Corruption(size_t bytes, const Status& s) override {
    cerr << "Corruption detected in log file " << s.ToString() << "\n";
  }
};

class InMemoryHandler : public WriteBatch::Handler {
 public:
  InMemoryHandler(stringstream& row, bool print_values) : Handler(), row_(row) {
    print_values_ = print_values;
  }

  void commonPutMerge(const Slice& key, const Slice& value) {
    string k = LDBCommand::StringToHex(key.ToString());
    if (print_values_) {
      string v = LDBCommand::StringToHex(value.ToString());
      row_ << k << " : ";
      row_ << v << " ";
    } else {
      row_ << k << " ";
    }
  }

  virtual void Put(const Slice& key, const Slice& value) override {
    row_ << "PUT : ";
    commonPutMerge(key, value);
  }

  virtual void Merge(const Slice& key, const Slice& value) override {
    row_ << "MERGE : ";
    commonPutMerge(key, value);
  }

  virtual void Delete(const Slice& key) override {
    row_ <<",DELETE : ";
    row_ << LDBCommand::StringToHex(key.ToString()) << " ";
  }

  virtual ~InMemoryHandler() {}

 private:
  stringstream & row_;
  bool print_values_;
};

void DumpWalFile(std::string wal_file, bool print_header, bool print_values,
                 LDBCommandExecuteResult* exec_state) {
  Env* env_ = Env::Default();
  EnvOptions soptions;
  unique_ptr<SequentialFileReader> wal_file_reader;

  Status status;
  {
    unique_ptr<SequentialFile> file;
    status = env_->NewSequentialFile(wal_file, &file, soptions);
    if (status.ok()) {
      wal_file_reader.reset(new SequentialFileReader(std::move(file)));
    }
  }
  if (!status.ok()) {
    if (exec_state) {
      *exec_state = LDBCommandExecuteResult::Failed("Failed to open WAL file " +
                                                    status.ToString());
    } else {
      cerr << "Error: Failed to open WAL file " << status.ToString()
           << std::endl;
    }
  } else {
    StdErrReporter reporter;
    log::Reader reader(move(wal_file_reader), &reporter, true, 0);
    string scratch;
    WriteBatch batch;
    Slice record;
    stringstream row;
    if (print_header) {
      cout << "Sequence,Count,ByteSize,Physical Offset,Key(s)";
      if (print_values) {
        cout << " : value ";
      }
      cout << "\n";
    }
    while (reader.ReadRecord(&record, &scratch)) {
      row.str("");
      if (record.size() < 12) {
        reporter.Corruption(record.size(),
                            Status::Corruption("log record too small"));
      } else {
        WriteBatchInternal::SetContents(&batch, record);
        row << WriteBatchInternal::Sequence(&batch) << ",";
        row << WriteBatchInternal::Count(&batch) << ",";
        row << WriteBatchInternal::ByteSize(&batch) << ",";
        row << reader.LastRecordOffset() << ",";
        InMemoryHandler handler(row, print_values);
        batch.Iterate(&handler);
        row << "\n";
      }
      cout << row.str();
    }
  }
}

}  // namespace

const string WALDumperCommand::ARG_WAL_FILE = "walfile";
const string WALDumperCommand::ARG_PRINT_VALUE = "print_value";
const string WALDumperCommand::ARG_PRINT_HEADER = "header";

WALDumperCommand::WALDumperCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags) :
    LDBCommand(options, flags, true,
               BuildCmdLineOptions(
                {ARG_WAL_FILE, ARG_PRINT_HEADER, ARG_PRINT_VALUE})),
    print_header_(false), print_values_(false) {

  wal_file_.clear();

  map<string, string>::const_iterator itr = options.find(ARG_WAL_FILE);
  if (itr != options.end()) {
    wal_file_ = itr->second;
  }


  print_header_ = IsFlagPresent(flags, ARG_PRINT_HEADER);
  print_values_ = IsFlagPresent(flags, ARG_PRINT_VALUE);
  if (wal_file_.empty()) {
    exec_state_ = LDBCommandExecuteResult::Failed("Argument " + ARG_WAL_FILE +
                                                  " must be specified.");
  }
}

void WALDumperCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(WALDumperCommand::Name());
  ret.append(" --" + ARG_WAL_FILE + "=<write_ahead_log_file_path>");
  ret.append(" [--" + ARG_PRINT_HEADER + "] ");
  ret.append(" [--" + ARG_PRINT_VALUE + "] ");
  ret.append("\n");
}

void WALDumperCommand::DoCommand() {
  DumpWalFile(wal_file_, print_header_, print_values_, &exec_state_);
}

// ----------------------------------------------------------------------------

GetCommand::GetCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags) :
  LDBCommand(options, flags, true, BuildCmdLineOptions({ARG_TTL, ARG_HEX,
                                                        ARG_KEY_HEX,
                                                        ARG_VALUE_HEX})) {

  if (params.size() != 1) {
    exec_state_ = LDBCommandExecuteResult::Failed(
        "<key> must be specified for the get command");
  } else {
    key_ = params.at(0);
  }

  if (is_key_hex_) {
    key_ = HexToString(key_);
  }
}

void GetCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(GetCommand::Name());
  ret.append(" <key>");
  ret.append(" [--" + ARG_TTL + "]");
  ret.append("\n");
}

void GetCommand::DoCommand() {
  string value;
  Status st = db_->Get(ReadOptions(), key_, &value);
  if (st.ok()) {
    fprintf(stdout, "%s\n",
              (is_value_hex_ ? StringToHex(value) : value).c_str());
  } else {
    exec_state_ = LDBCommandExecuteResult::Failed(st.ToString());
  }
}

// ----------------------------------------------------------------------------

ApproxSizeCommand::ApproxSizeCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags) :
  LDBCommand(options, flags, true,
             BuildCmdLineOptions({ARG_HEX, ARG_KEY_HEX, ARG_VALUE_HEX,
                                  ARG_FROM, ARG_TO})) {

  if (options.find(ARG_FROM) != options.end()) {
    start_key_ = options.find(ARG_FROM)->second;
  } else {
    exec_state_ = LDBCommandExecuteResult::Failed(
        ARG_FROM + " must be specified for approxsize command");
    return;
  }

  if (options.find(ARG_TO) != options.end()) {
    end_key_ = options.find(ARG_TO)->second;
  } else {
    exec_state_ = LDBCommandExecuteResult::Failed(
        ARG_TO + " must be specified for approxsize command");
    return;
  }

  if (is_key_hex_) {
    start_key_ = HexToString(start_key_);
    end_key_ = HexToString(end_key_);
  }
}

void ApproxSizeCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(ApproxSizeCommand::Name());
  ret.append(HelpRangeCmdArgs());
  ret.append("\n");
}

void ApproxSizeCommand::DoCommand() {

  Range ranges[1];
  ranges[0] = Range(start_key_, end_key_);
  uint64_t sizes[1];
  db_->GetApproximateSizes(ranges, 1, sizes);
  fprintf(stdout, "%lu\n", (unsigned long)sizes[0]);
  /* Weird that GetApproximateSizes() returns void, although documentation
   * says that it returns a Status object.
  if (!st.ok()) {
    exec_state_ = LDBCommandExecuteResult::Failed(st.ToString());
  }
  */
}

// ----------------------------------------------------------------------------

BatchPutCommand::BatchPutCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags) :
  LDBCommand(options, flags, false,
             BuildCmdLineOptions({ARG_TTL, ARG_HEX, ARG_KEY_HEX, ARG_VALUE_HEX,
                                  ARG_CREATE_IF_MISSING})) {

  if (params.size() < 2) {
    exec_state_ = LDBCommandExecuteResult::Failed(
        "At least one <key> <value> pair must be specified batchput.");
  } else if (params.size() % 2 != 0) {
    exec_state_ = LDBCommandExecuteResult::Failed(
        "Equal number of <key>s and <value>s must be specified for batchput.");
  } else {
    for (size_t i = 0; i < params.size(); i += 2) {
      string key = params.at(i);
      string value = params.at(i+1);
      key_values_.push_back(pair<string, string>(
                    is_key_hex_ ? HexToString(key) : key,
                    is_value_hex_ ? HexToString(value) : value));
    }
  }
}

void BatchPutCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(BatchPutCommand::Name());
  ret.append(" <key> <value> [<key> <value>] [..]");
  ret.append(" [--" + ARG_TTL + "]");
  ret.append("\n");
}

void BatchPutCommand::DoCommand() {
  WriteBatch batch;

  for (vector<pair<string, string>>::const_iterator itr
        = key_values_.begin(); itr != key_values_.end(); ++itr) {
      batch.Put(itr->first, itr->second);
  }
  Status st = db_->Write(WriteOptions(), &batch);
  if (st.ok()) {
    fprintf(stdout, "OK\n");
  } else {
    exec_state_ = LDBCommandExecuteResult::Failed(st.ToString());
  }
}

Options BatchPutCommand::PrepareOptionsForOpenDB() {
  Options opt = LDBCommand::PrepareOptionsForOpenDB();
  opt.create_if_missing = IsFlagPresent(flags_, ARG_CREATE_IF_MISSING);
  return opt;
}

// ----------------------------------------------------------------------------

ScanCommand::ScanCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags) :
    LDBCommand(options, flags, true,
               BuildCmdLineOptions({ARG_TTL, ARG_HEX, ARG_KEY_HEX, ARG_TO,
                                    ARG_VALUE_HEX, ARG_FROM, ARG_TIMESTAMP,
                                    ARG_MAX_KEYS, ARG_TTL_START, ARG_TTL_END})),
    start_key_specified_(false),
    end_key_specified_(false),
    max_keys_scanned_(-1) {

  map<string, string>::const_iterator itr = options.find(ARG_FROM);
  if (itr != options.end()) {
    start_key_ = itr->second;
    if (is_key_hex_) {
      start_key_ = HexToString(start_key_);
    }
    start_key_specified_ = true;
  }
  itr = options.find(ARG_TO);
  if (itr != options.end()) {
    end_key_ = itr->second;
    if (is_key_hex_) {
      end_key_ = HexToString(end_key_);
    }
    end_key_specified_ = true;
  }

  itr = options.find(ARG_MAX_KEYS);
  if (itr != options.end()) {
    try {
#if defined(CYGWIN)
      max_keys_scanned_ = strtol(itr->second.c_str(), 0, 10);
#else
      max_keys_scanned_ = stoi(itr->second);
#endif
    } catch(const invalid_argument&) {
      exec_state_ = LDBCommandExecuteResult::Failed(ARG_MAX_KEYS +
                                                    " has an invalid value");
    } catch(const out_of_range&) {
      exec_state_ = LDBCommandExecuteResult::Failed(
          ARG_MAX_KEYS + " has a value out-of-range");
    }
  }
}

void ScanCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(ScanCommand::Name());
  ret.append(HelpRangeCmdArgs());
  ret.append(" [--" + ARG_TTL + "]");
  ret.append(" [--" + ARG_TIMESTAMP + "]");
  ret.append(" [--" + ARG_MAX_KEYS + "=<N>q] ");
  ret.append(" [--" + ARG_TTL_START + "=<N>:- is inclusive]");
  ret.append(" [--" + ARG_TTL_END + "=<N>:- is exclusive]");
  ret.append("\n");
}

void ScanCommand::DoCommand() {

  int num_keys_scanned = 0;
  Iterator* it = db_->NewIterator(ReadOptions());
  if (start_key_specified_) {
    it->Seek(start_key_);
  } else {
    it->SeekToFirst();
  }
  int ttl_start;
  if (!ParseIntOption(option_map_, ARG_TTL_START, ttl_start, exec_state_)) {
    ttl_start = DBWithTTLImpl::kMinTimestamp;  // TTL introduction time
  }
  int ttl_end;
  if (!ParseIntOption(option_map_, ARG_TTL_END, ttl_end, exec_state_)) {
    ttl_end = DBWithTTLImpl::kMaxTimestamp;  // Max time allowed by TTL feature
  }
  if (ttl_end < ttl_start) {
    fprintf(stderr, "Error: End time can't be less than start time\n");
    delete it;
    return;
  }
  if (is_db_ttl_ && timestamp_) {
    fprintf(stdout, "Scanning key-values from %s to %s\n",
            ReadableTime(ttl_start).c_str(), ReadableTime(ttl_end).c_str());
  }
  for ( ;
        it->Valid() && (!end_key_specified_ || it->key().ToString() < end_key_);
        it->Next()) {
    string key = ldb_options_.key_formatter->Format(it->key());
    if (is_db_ttl_) {
      TtlIterator* it_ttl = dynamic_cast<TtlIterator*>(it);
      assert(it_ttl);
      int rawtime = it_ttl->timestamp();
      if (rawtime < ttl_start || rawtime >= ttl_end) {
        continue;
      }
      if (timestamp_) {
        fprintf(stdout, "%s ", ReadableTime(rawtime).c_str());
      }
    }
    string value = it->value().ToString();
    fprintf(stdout, "%s : %s\n",
            (is_key_hex_ ? "0x" + it->key().ToString(true) : key).c_str(),
            (is_value_hex_ ? StringToHex(value) : value).c_str()
        );
    num_keys_scanned++;
    if (max_keys_scanned_ >= 0 && num_keys_scanned >= max_keys_scanned_) {
      break;
    }
  }
  if (!it->status().ok()) {  // Check for any errors found during the scan
    exec_state_ = LDBCommandExecuteResult::Failed(it->status().ToString());
  }
  delete it;
}

// ----------------------------------------------------------------------------

DeleteCommand::DeleteCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags) :
  LDBCommand(options, flags, false,
             BuildCmdLineOptions({ARG_HEX, ARG_KEY_HEX, ARG_VALUE_HEX})) {

  if (params.size() != 1) {
    exec_state_ = LDBCommandExecuteResult::Failed(
        "KEY must be specified for the delete command");
  } else {
    key_ = params.at(0);
    if (is_key_hex_) {
      key_ = HexToString(key_);
    }
  }
}

void DeleteCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(DeleteCommand::Name() + " <key>");
  ret.append("\n");
}

void DeleteCommand::DoCommand() {
  Status st = db_->Delete(WriteOptions(), key_);
  if (st.ok()) {
    fprintf(stdout, "OK\n");
  } else {
    exec_state_ = LDBCommandExecuteResult::Failed(st.ToString());
  }
}


PutCommand::PutCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags) :
  LDBCommand(options, flags, false,
             BuildCmdLineOptions({ARG_TTL, ARG_HEX, ARG_KEY_HEX, ARG_VALUE_HEX,
                                  ARG_CREATE_IF_MISSING})) {

  if (params.size() != 2) {
    exec_state_ = LDBCommandExecuteResult::Failed(
        "<key> and <value> must be specified for the put command");
  } else {
    key_ = params.at(0);
    value_ = params.at(1);
  }

  if (is_key_hex_) {
    key_ = HexToString(key_);
  }

  if (is_value_hex_) {
    value_ = HexToString(value_);
  }
}

void PutCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(PutCommand::Name());
  ret.append(" <key> <value> ");
  ret.append(" [--" + ARG_TTL + "]");
  ret.append("\n");
}

void PutCommand::DoCommand() {
  Status st = db_->Put(WriteOptions(), key_, value_);
  if (st.ok()) {
    fprintf(stdout, "OK\n");
  } else {
    exec_state_ = LDBCommandExecuteResult::Failed(st.ToString());
  }
}

Options PutCommand::PrepareOptionsForOpenDB() {
  Options opt = LDBCommand::PrepareOptionsForOpenDB();
  opt.create_if_missing = IsFlagPresent(flags_, ARG_CREATE_IF_MISSING);
  return opt;
}

// ----------------------------------------------------------------------------

const char* DBQuerierCommand::HELP_CMD = "help";
const char* DBQuerierCommand::GET_CMD = "get";
const char* DBQuerierCommand::PUT_CMD = "put";
const char* DBQuerierCommand::DELETE_CMD = "delete";

DBQuerierCommand::DBQuerierCommand(const vector<string>& params,
    const map<string, string>& options, const vector<string>& flags) :
  LDBCommand(options, flags, false,
             BuildCmdLineOptions({ARG_TTL, ARG_HEX, ARG_KEY_HEX,
                                  ARG_VALUE_HEX})) {

}

void DBQuerierCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(DBQuerierCommand::Name());
  ret.append(" [--" + ARG_TTL + "]");
  ret.append("\n");
  ret.append("    Starts a REPL shell.  Type help for list of available "
             "commands.");
  ret.append("\n");
}

void DBQuerierCommand::DoCommand() {
  if (!db_) {
    return;
  }

  ReadOptions read_options;
  WriteOptions write_options;

  string line;
  string key;
  string value;
  while (getline(cin, line, '\n')) {

    // Parse line into vector<string>
    vector<string> tokens;
    size_t pos = 0;
    while (true) {
      size_t pos2 = line.find(' ', pos);
      if (pos2 == string::npos) {
        break;
      }
      tokens.push_back(line.substr(pos, pos2-pos));
      pos = pos2 + 1;
    }
    tokens.push_back(line.substr(pos));

    const string& cmd = tokens[0];

    if (cmd == HELP_CMD) {
      fprintf(stdout,
              "get <key>\n"
              "put <key> <value>\n"
              "delete <key>\n");
    } else if (cmd == DELETE_CMD && tokens.size() == 2) {
      key = (is_key_hex_ ? HexToString(tokens[1]) : tokens[1]);
      db_->Delete(write_options, Slice(key));
      fprintf(stdout, "Successfully deleted %s\n", tokens[1].c_str());
    } else if (cmd == PUT_CMD && tokens.size() == 3) {
      key = (is_key_hex_ ? HexToString(tokens[1]) : tokens[1]);
      value = (is_value_hex_ ? HexToString(tokens[2]) : tokens[2]);
      db_->Put(write_options, Slice(key), Slice(value));
      fprintf(stdout, "Successfully put %s %s\n",
              tokens[1].c_str(), tokens[2].c_str());
    } else if (cmd == GET_CMD && tokens.size() == 2) {
      key = (is_key_hex_ ? HexToString(tokens[1]) : tokens[1]);
      if (db_->Get(read_options, Slice(key), &value).ok()) {
        fprintf(stdout, "%s\n", PrintKeyValue(key, value,
              is_key_hex_, is_value_hex_).c_str());
      } else {
        fprintf(stdout, "Not found %s\n", tokens[1].c_str());
      }
    } else {
      fprintf(stdout, "Unknown command %s\n", line.c_str());
    }
  }
}

// ----------------------------------------------------------------------------

CheckConsistencyCommand::CheckConsistencyCommand(const vector<string>& params,
    const map<string, string>& options, const vector<string>& flags) :
  LDBCommand(options, flags, false,
             BuildCmdLineOptions({})) {
}

void CheckConsistencyCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(CheckConsistencyCommand::Name());
  ret.append("\n");
}

void CheckConsistencyCommand::DoCommand() {
  Options opt = PrepareOptionsForOpenDB();
  opt.paranoid_checks = true;
  if (!exec_state_.IsNotStarted()) {
    return;
  }
  DB* db;
  Status st = DB::OpenForReadOnly(opt, db_path_, &db, false);
  delete db;
  if (st.ok()) {
    fprintf(stdout, "OK\n");
  } else {
    exec_state_ = LDBCommandExecuteResult::Failed(st.ToString());
  }
}

// ----------------------------------------------------------------------------

namespace {

void DumpSstFile(std::string filename, bool output_hex, bool show_properties) {
  std::string from_key;
  std::string to_key;
  if (filename.length() <= 4 ||
      filename.rfind(".sst") != filename.length() - 4) {
    std::cout << "Invalid sst file name." << std::endl;
    return;
  }
  // no verification
  rocksdb::SstFileReader reader(filename, false, output_hex);
  Status st = reader.ReadSequential(true, -1, false,  // has_from
                                    from_key, false,  // has_to
                                    to_key);
  if (!st.ok()) {
    std::cerr << "Error in reading SST file " << filename << st.ToString()
              << std::endl;
    return;
  }

  if (show_properties) {
    const rocksdb::TableProperties* table_properties;

    std::shared_ptr<const rocksdb::TableProperties>
        table_properties_from_reader;
    st = reader.ReadTableProperties(&table_properties_from_reader);
    if (!st.ok()) {
      std::cerr << filename << ": " << st.ToString()
                << ". Try to use initial table properties" << std::endl;
      table_properties = reader.GetInitTableProperties();
    } else {
      table_properties = table_properties_from_reader.get();
    }
    if (table_properties != nullptr) {
      std::cout << std::endl << "Table Properties:" << std::endl;
      std::cout << table_properties->ToString("\n") << std::endl;
      std::cout << "# deleted keys: "
                << rocksdb::GetDeletedKeys(
                       table_properties->user_collected_properties)
                << std::endl;
    }
  }
}

}  // namespace

DBFileDumperCommand::DBFileDumperCommand(const vector<string>& params,
                                         const map<string, string>& options,
                                         const vector<string>& flags)
    : LDBCommand(options, flags, true, BuildCmdLineOptions({})) {}

void DBFileDumperCommand::Help(string& ret) {
  ret.append("  ");
  ret.append(DBFileDumperCommand::Name());
  ret.append("\n");
}

void DBFileDumperCommand::DoCommand() {
  if (!db_) {
    return;
  }
  Status s;

  std::cout << "Manifest File" << std::endl;
  std::cout << "==============================" << std::endl;
  std::string manifest_filename;
  s = ReadFileToString(db_->GetEnv(), CurrentFileName(db_->GetName()),
                       &manifest_filename);
  if (!s.ok() || manifest_filename.empty() ||
      manifest_filename.back() != '\n') {
    std::cerr << "Error when reading CURRENT file "
              << CurrentFileName(db_->GetName()) << std::endl;
  }
  // remove the trailing '\n'
  manifest_filename.resize(manifest_filename.size() - 1);
  string manifest_filepath = db_->GetName() + "/" + manifest_filename;
  std::cout << manifest_filepath << std::endl;
  DumpManifestFile(manifest_filepath, false, false, false);
  std::cout << std::endl;

  std::cout << "SST Files" << std::endl;
  std::cout << "==============================" << std::endl;
  std::vector<LiveFileMetaData> metadata;
  db_->GetLiveFilesMetaData(&metadata);
  for (auto& fileMetadata : metadata) {
    std::string filename = fileMetadata.db_path + fileMetadata.name;
    std::cout << filename << " level:" << fileMetadata.level << std::endl;
    std::cout << "------------------------------" << std::endl;
    DumpSstFile(filename, false, true);
    std::cout << std::endl;
  }
  std::cout << std::endl;

  std::cout << "Write Ahead Log Files" << std::endl;
  std::cout << "==============================" << std::endl;
  rocksdb::VectorLogPtr wal_files;
  s = db_->GetSortedWalFiles(wal_files);
  if (!s.ok()) {
    std::cerr << "Error when getting WAL files" << std::endl;
  } else {
    for (auto& wal : wal_files) {
      // TODO(qyang): option.wal_dir should be passed into ldb command
      std::string filename = db_->GetOptions().wal_dir + wal->PathName();
      std::cout << filename << std::endl;
      DumpWalFile(filename, true, true, &exec_state_);
    }
  }
}

}   // namespace rocksdb
#endif  // ROCKSDB_LITE
