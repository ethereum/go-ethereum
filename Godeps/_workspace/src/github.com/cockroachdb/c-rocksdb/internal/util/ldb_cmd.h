//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
#pragma once

#ifndef ROCKSDB_LITE

#include <string>
#include <iostream>
#include <sstream>
#include <stdlib.h>
#include <algorithm>
#include <stdio.h>
#include <vector>
#include <map>

#include "db/version_set.h"
#include "rocksdb/env.h"
#include "rocksdb/iterator.h"
#include "rocksdb/ldb_tool.h"
#include "rocksdb/options.h"
#include "rocksdb/slice.h"
#include "rocksdb/utilities/db_ttl.h"
#include "util/logging.h"
#include "util/ldb_cmd_execute_result.h"
#include "util/string_util.h"
#include "utilities/ttl/db_ttl_impl.h"

using std::string;
using std::map;
using std::vector;
using std::ostringstream;

namespace rocksdb {

class LDBCommand {
public:

  // Command-line arguments
  static const string ARG_DB;
  static const string ARG_HEX;
  static const string ARG_KEY_HEX;
  static const string ARG_VALUE_HEX;
  static const string ARG_TTL;
  static const string ARG_TTL_START;
  static const string ARG_TTL_END;
  static const string ARG_TIMESTAMP;
  static const string ARG_FROM;
  static const string ARG_TO;
  static const string ARG_MAX_KEYS;
  static const string ARG_BLOOM_BITS;
  static const string ARG_FIX_PREFIX_LEN;
  static const string ARG_COMPRESSION_TYPE;
  static const string ARG_BLOCK_SIZE;
  static const string ARG_AUTO_COMPACTION;
  static const string ARG_DB_WRITE_BUFFER_SIZE;
  static const string ARG_WRITE_BUFFER_SIZE;
  static const string ARG_FILE_SIZE;
  static const string ARG_CREATE_IF_MISSING;

  static LDBCommand* InitFromCmdLineArgs(
    const vector<string>& args,
    const Options& options,
    const LDBOptions& ldb_options
  );

  static LDBCommand* InitFromCmdLineArgs(
    int argc,
    char** argv,
    const Options& options,
    const LDBOptions& ldb_options
  );

  bool ValidateCmdLineOptions();

  virtual Options PrepareOptionsForOpenDB();

  virtual void SetDBOptions(Options options) {
    options_ = options;
  }

  void SetLDBOptions(const LDBOptions& ldb_options) {
    ldb_options_ = ldb_options;
  }

  virtual bool NoDBOpen() {
    return false;
  }

  virtual ~LDBCommand() {
    if (db_ != nullptr) {
      delete db_;
      db_ = nullptr;
    }
  }

  /* Run the command, and return the execute result. */
  void Run() {
    if (!exec_state_.IsNotStarted()) {
      return;
    }

    if (db_ == nullptr && !NoDBOpen()) {
      OpenDB();
      if (!exec_state_.IsNotStarted()) {
        return;
      }
    }

    DoCommand();
    if (exec_state_.IsNotStarted()) {
      exec_state_ = LDBCommandExecuteResult::Succeed("");
    }

    if (db_ != nullptr) {
      CloseDB ();
    }
  }

  virtual void DoCommand() = 0;

  LDBCommandExecuteResult GetExecuteState() {
    return exec_state_;
  }

  void ClearPreviousRunState() {
    exec_state_.Reset();
  }

  static string HexToString(const string& str) {
    string parsed;
    if (str[0] != '0' || str[1] != 'x') {
      fprintf(stderr, "Invalid hex input %s.  Must start with 0x\n",
              str.c_str());
      throw "Invalid hex input";
    }

    for (unsigned int i = 2; i < str.length();) {
      int c;
      sscanf(str.c_str() + i, "%2X", &c);
      parsed.push_back(c);
      i += 2;
    }
    return parsed;
  }

  static string StringToHex(const string& str) {
    string result = "0x";
    char buf[10];
    for (size_t i = 0; i < str.length(); i++) {
      snprintf(buf, 10, "%02X", (unsigned char)str[i]);
      result += buf;
    }
    return result;
  }

  static const char* DELIM;

protected:

  LDBCommandExecuteResult exec_state_;
  string db_path_;
  DB* db_;
  DBWithTTL* db_ttl_;

  /**
   * true implies that this command can work if the db is opened in read-only
   * mode.
   */
  bool is_read_only_;

  /** If true, the key is input/output as hex in get/put/scan/delete etc. */
  bool is_key_hex_;

  /** If true, the value is input/output as hex in get/put/scan/delete etc. */
  bool is_value_hex_;

  /** If true, the value is treated as timestamp suffixed */
  bool is_db_ttl_;

  // If true, the kvs are output with their insert/modify timestamp in a ttl db
  bool timestamp_;

  /**
   * Map of options passed on the command-line.
   */
  const map<string, string> option_map_;

  /**
   * Flags passed on the command-line.
   */
  const vector<string> flags_;

  /** List of command-line options valid for this command */
  const vector<string> valid_cmd_line_options_;

  bool ParseKeyValue(const string& line, string* key, string* value,
                      bool is_key_hex, bool is_value_hex);

  LDBCommand(const map<string, string>& options, const vector<string>& flags,
             bool is_read_only, const vector<string>& valid_cmd_line_options) :
      db_(nullptr),
      is_read_only_(is_read_only),
      is_key_hex_(false),
      is_value_hex_(false),
      is_db_ttl_(false),
      timestamp_(false),
      option_map_(options),
      flags_(flags),
      valid_cmd_line_options_(valid_cmd_line_options) {

    map<string, string>::const_iterator itr = options.find(ARG_DB);
    if (itr != options.end()) {
      db_path_ = itr->second;
    }

    is_key_hex_ = IsKeyHex(options, flags);
    is_value_hex_ = IsValueHex(options, flags);
    is_db_ttl_ = IsFlagPresent(flags, ARG_TTL);
    timestamp_ = IsFlagPresent(flags, ARG_TIMESTAMP);
  }

  void OpenDB() {
    Options opt = PrepareOptionsForOpenDB();
    if (!exec_state_.IsNotStarted()) {
      return;
    }
    // Open the DB.
    Status st;
    if (is_db_ttl_) {
      if (is_read_only_) {
        st = DBWithTTL::Open(opt, db_path_, &db_ttl_, 0, true);
      } else {
        st = DBWithTTL::Open(opt, db_path_, &db_ttl_);
      }
      db_ = db_ttl_;
    } else if (is_read_only_) {
      st = DB::OpenForReadOnly(opt, db_path_, &db_);
    } else {
      st = DB::Open(opt, db_path_, &db_);
    }
    if (!st.ok()) {
      string msg = st.ToString();
      exec_state_ = LDBCommandExecuteResult::Failed(msg);
    }

    options_ = opt;
  }

  void CloseDB () {
    if (db_ != nullptr) {
      delete db_;
      db_ = nullptr;
    }
  }

  static string PrintKeyValue(const string& key, const string& value,
        bool is_key_hex, bool is_value_hex) {
    string result;
    result.append(is_key_hex ? StringToHex(key) : key);
    result.append(DELIM);
    result.append(is_value_hex ? StringToHex(value) : value);
    return result;
  }

  static string PrintKeyValue(const string& key, const string& value,
        bool is_hex) {
    return PrintKeyValue(key, value, is_hex, is_hex);
  }

  /**
   * Return true if the specified flag is present in the specified flags vector
   */
  static bool IsFlagPresent(const vector<string>& flags, const string& flag) {
    return (std::find(flags.begin(), flags.end(), flag) != flags.end());
  }

  static string HelpRangeCmdArgs() {
    ostringstream str_stream;
    str_stream << " ";
    str_stream << "[--" << ARG_FROM << "] ";
    str_stream << "[--" << ARG_TO << "] ";
    return str_stream.str();
  }

  /**
   * A helper function that returns a list of command line options
   * used by this command.  It includes the common options and the ones
   * passed in.
   */
  static vector<string> BuildCmdLineOptions(vector<string> options) {
    vector<string> ret = {ARG_DB,               ARG_BLOOM_BITS,
                          ARG_BLOCK_SIZE,       ARG_AUTO_COMPACTION,
                          ARG_COMPRESSION_TYPE, ARG_WRITE_BUFFER_SIZE,
                          ARG_FILE_SIZE,        ARG_FIX_PREFIX_LEN};
    ret.insert(ret.end(), options.begin(), options.end());
    return ret;
  }

  bool ParseIntOption(const map<string, string>& options, const string& option,
                      int& value, LDBCommandExecuteResult& exec_state);

  bool ParseStringOption(const map<string, string>& options,
                         const string& option, string* value);

  Options options_;
  LDBOptions ldb_options_;

private:

  /**
   * Interpret command line options and flags to determine if the key
   * should be input/output in hex.
   */
  bool IsKeyHex(const map<string, string>& options,
      const vector<string>& flags) {
    return (IsFlagPresent(flags, ARG_HEX) ||
        IsFlagPresent(flags, ARG_KEY_HEX) ||
        ParseBooleanOption(options, ARG_HEX, false) ||
        ParseBooleanOption(options, ARG_KEY_HEX, false));
  }

  /**
   * Interpret command line options and flags to determine if the value
   * should be input/output in hex.
   */
  bool IsValueHex(const map<string, string>& options,
      const vector<string>& flags) {
    return (IsFlagPresent(flags, ARG_HEX) ||
          IsFlagPresent(flags, ARG_VALUE_HEX) ||
          ParseBooleanOption(options, ARG_HEX, false) ||
          ParseBooleanOption(options, ARG_VALUE_HEX, false));
  }

  /**
   * Returns the value of the specified option as a boolean.
   * default_val is used if the option is not found in options.
   * Throws an exception if the value of the option is not
   * "true" or "false" (case insensitive).
   */
  bool ParseBooleanOption(const map<string, string>& options,
      const string& option, bool default_val) {

    map<string, string>::const_iterator itr = options.find(option);
    if (itr != options.end()) {
      string option_val = itr->second;
      return StringToBool(itr->second);
    }
    return default_val;
  }

  /**
   * Converts val to a boolean.
   * val must be either true or false (case insensitive).
   * Otherwise an exception is thrown.
   */
  bool StringToBool(string val) {
    std::transform(val.begin(), val.end(), val.begin(),
                   [](char ch) -> char { return ::tolower(ch); });

    if (val == "true") {
      return true;
    } else if (val == "false") {
      return false;
    } else {
      throw "Invalid value for boolean argument";
    }
  }

  static LDBCommand* SelectCommand(
    const string& cmd,
    const vector<string>& cmdParams,
    const map<string, string>& option_map,
    const vector<string>& flags
  );

};

class CompactorCommand: public LDBCommand {
public:
  static string Name() { return "compact"; }

  CompactorCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags);

  static void Help(string& ret);

  virtual void DoCommand() override;

private:
  bool null_from_;
  string from_;
  bool null_to_;
  string to_;
};

class DBFileDumperCommand : public LDBCommand {
 public:
  static string Name() { return "dump_live_files"; }

  DBFileDumperCommand(const vector<string>& params,
                      const map<string, string>& options,
                      const vector<string>& flags);

  static void Help(string& ret);

  virtual void DoCommand() override;
};

class DBDumperCommand: public LDBCommand {
public:
  static string Name() { return "dump"; }

  DBDumperCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags);

  static void Help(string& ret);

  virtual void DoCommand() override;

private:
  bool null_from_;
  string from_;
  bool null_to_;
  string to_;
  int max_keys_;
  string delim_;
  bool count_only_;
  bool count_delim_;
  bool print_stats_;

  static const string ARG_COUNT_ONLY;
  static const string ARG_COUNT_DELIM;
  static const string ARG_STATS;
  static const string ARG_TTL_BUCKET;
};

class InternalDumpCommand: public LDBCommand {
public:
  static string Name() { return "idump"; }

  InternalDumpCommand(const vector<string>& params,
                      const map<string, string>& options,
                      const vector<string>& flags);

  static void Help(string& ret);

  virtual void DoCommand() override;

private:
  bool has_from_;
  string from_;
  bool has_to_;
  string to_;
  int max_keys_;
  string delim_;
  bool count_only_;
  bool count_delim_;
  bool print_stats_;
  bool is_input_key_hex_;

  static const string ARG_DELIM;
  static const string ARG_COUNT_ONLY;
  static const string ARG_COUNT_DELIM;
  static const string ARG_STATS;
  static const string ARG_INPUT_KEY_HEX;
};

class DBLoaderCommand: public LDBCommand {
public:
  static string Name() { return "load"; }

  DBLoaderCommand(string& db_name, vector<string>& args);

  DBLoaderCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags);

  static void Help(string& ret);
  virtual void DoCommand() override;

  virtual Options PrepareOptionsForOpenDB() override;

private:
  bool create_if_missing_;
  bool disable_wal_;
  bool bulk_load_;
  bool compact_;

  static const string ARG_DISABLE_WAL;
  static const string ARG_BULK_LOAD;
  static const string ARG_COMPACT;
};

class ManifestDumpCommand: public LDBCommand {
public:
  static string Name() { return "manifest_dump"; }

  ManifestDumpCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags);

  static void Help(string& ret);
  virtual void DoCommand() override;

  virtual bool NoDBOpen() override { return true; }

private:
  bool verbose_;
  bool json_;
  string path_;

  static const string ARG_VERBOSE;
  static const string ARG_JSON;
  static const string ARG_PATH;
};

class ListColumnFamiliesCommand : public LDBCommand {
 public:
  static string Name() { return "list_column_families"; }

  ListColumnFamiliesCommand(const vector<string>& params,
                            const map<string, string>& options,
                            const vector<string>& flags);

  static void Help(string& ret);
  virtual void DoCommand() override;

  virtual bool NoDBOpen() override { return true; }

 private:
  string dbname_;
};

class ReduceDBLevelsCommand : public LDBCommand {
public:
  static string Name() { return "reduce_levels"; }

  ReduceDBLevelsCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags);

  virtual Options PrepareOptionsForOpenDB() override;

  virtual void DoCommand() override;

  virtual bool NoDBOpen() override { return true; }

  static void Help(string& msg);

  static vector<string> PrepareArgs(const string& db_path, int new_levels,
      bool print_old_level = false);

private:
  int old_levels_;
  int new_levels_;
  bool print_old_levels_;

  static const string ARG_NEW_LEVELS;
  static const string ARG_PRINT_OLD_LEVELS;

  Status GetOldNumOfLevels(Options& opt, int* levels);
};

class ChangeCompactionStyleCommand : public LDBCommand {
public:
  static string Name() { return "change_compaction_style"; }

  ChangeCompactionStyleCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags);

  virtual Options PrepareOptionsForOpenDB() override;

  virtual void DoCommand() override;

  static void Help(string& msg);

private:
  int old_compaction_style_;
  int new_compaction_style_;

  static const string ARG_OLD_COMPACTION_STYLE;
  static const string ARG_NEW_COMPACTION_STYLE;
};

class WALDumperCommand : public LDBCommand {
public:
  static string Name() { return "dump_wal"; }

  WALDumperCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags);

  virtual bool NoDBOpen() override { return true; }

  static void Help(string& ret);
  virtual void DoCommand() override;

private:
  bool print_header_;
  string wal_file_;
  bool print_values_;

  static const string ARG_WAL_FILE;
  static const string ARG_PRINT_HEADER;
  static const string ARG_PRINT_VALUE;
};


class GetCommand : public LDBCommand {
public:
  static string Name() { return "get"; }

  GetCommand(const vector<string>& params, const map<string, string>& options,
      const vector<string>& flags);

  virtual void DoCommand() override;

  static void Help(string& ret);

private:
  string key_;
};

class ApproxSizeCommand : public LDBCommand {
public:
  static string Name() { return "approxsize"; }

  ApproxSizeCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags);

  virtual void DoCommand() override;

  static void Help(string& ret);

private:
  string start_key_;
  string end_key_;
};

class BatchPutCommand : public LDBCommand {
public:
  static string Name() { return "batchput"; }

  BatchPutCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags);

  virtual void DoCommand() override;

  static void Help(string& ret);

  virtual Options PrepareOptionsForOpenDB() override;

private:
  /**
   * The key-values to be inserted.
   */
  vector<std::pair<string, string>> key_values_;
};

class ScanCommand : public LDBCommand {
public:
  static string Name() { return "scan"; }

  ScanCommand(const vector<string>& params, const map<string, string>& options,
      const vector<string>& flags);

  virtual void DoCommand() override;

  static void Help(string& ret);

private:
  string start_key_;
  string end_key_;
  bool start_key_specified_;
  bool end_key_specified_;
  int max_keys_scanned_;
};

class DeleteCommand : public LDBCommand {
public:
  static string Name() { return "delete"; }

  DeleteCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags);

  virtual void DoCommand() override;

  static void Help(string& ret);

private:
  string key_;
};

class PutCommand : public LDBCommand {
public:
  static string Name() { return "put"; }

  PutCommand(const vector<string>& params, const map<string, string>& options,
      const vector<string>& flags);

  virtual void DoCommand() override;

  static void Help(string& ret);

  virtual Options PrepareOptionsForOpenDB() override;

private:
  string key_;
  string value_;
};

/**
 * Command that starts up a REPL shell that allows
 * get/put/delete.
 */
class DBQuerierCommand: public LDBCommand {
public:
  static string Name() { return "query"; }

  DBQuerierCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags);

  static void Help(string& ret);

  virtual void DoCommand() override;

private:
  static const char* HELP_CMD;
  static const char* GET_CMD;
  static const char* PUT_CMD;
  static const char* DELETE_CMD;
};

class CheckConsistencyCommand : public LDBCommand {
public:
  static string Name() { return "checkconsistency"; }

  CheckConsistencyCommand(const vector<string>& params,
      const map<string, string>& options, const vector<string>& flags);

  virtual void DoCommand() override;

  virtual bool NoDBOpen() override { return true; }

  static void Help(string& ret);
};

} // namespace rocksdb

#endif  // ROCKSDB_LITE
