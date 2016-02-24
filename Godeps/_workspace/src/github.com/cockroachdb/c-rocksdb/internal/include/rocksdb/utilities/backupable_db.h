//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#pragma once
#ifndef ROCKSDB_LITE

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <inttypes.h>
#include <string>
#include <map>
#include <vector>

#include "rocksdb/utilities/stackable_db.h"

#include "rocksdb/env.h"
#include "rocksdb/status.h"

namespace rocksdb {

struct BackupableDBOptions {
  // Where to keep the backup files. Has to be different than dbname_
  // Best to set this to dbname_ + "/backups"
  // Required
  std::string backup_dir;

  // Backup Env object. It will be used for backup file I/O. If it's
  // nullptr, backups will be written out using DBs Env. If it's
  // non-nullptr, backup's I/O will be performed using this object.
  // If you want to have backups on HDFS, use HDFS Env here!
  // Default: nullptr
  Env* backup_env;

  // If share_table_files == true, backup will assume that table files with
  // same name have the same contents. This enables incremental backups and
  // avoids unnecessary data copies.
  // If share_table_files == false, each backup will be on its own and will
  // not share any data with other backups.
  // default: true
  bool share_table_files;

  // Backup info and error messages will be written to info_log
  // if non-nullptr.
  // Default: nullptr
  Logger* info_log;

  // If sync == true, we can guarantee you'll get consistent backup even
  // on a machine crash/reboot. Backup process is slower with sync enabled.
  // If sync == false, we don't guarantee anything on machine reboot. However,
  // chances are some of the backups are consistent.
  // Default: true
  bool sync;

  // If true, it will delete whatever backups there are already
  // Default: false
  bool destroy_old_data;

  // If false, we won't backup log files. This option can be useful for backing
  // up in-memory databases where log file are persisted, but table files are in
  // memory.
  // Default: true
  bool backup_log_files;

  // Max bytes that can be transferred in a second during backup.
  // If 0, go as fast as you can
  // Default: 0
  uint64_t backup_rate_limit;

  // Max bytes that can be transferred in a second during restore.
  // If 0, go as fast as you can
  // Default: 0
  uint64_t restore_rate_limit;

  // Only used if share_table_files is set to true. If true, will consider that
  // backups can come from different databases, hence a sst is not uniquely
  // identifed by its name, but by the triple (file name, crc32, file length)
  // Default: false
  // Note: this is an experimental option, and you'll need to set it manually
  // *turn it on only if you know what you're doing*
  bool share_files_with_checksum;

  // Up to this many background threads will copy files for CreateNewBackup()
  // and RestoreDBFromBackup()
  // Default: 1
  int max_background_operations;

  void Dump(Logger* logger) const;

  explicit BackupableDBOptions(const std::string& _backup_dir,
                               Env* _backup_env = nullptr,
                               bool _share_table_files = true,
                               Logger* _info_log = nullptr, bool _sync = true,
                               bool _destroy_old_data = false,
                               bool _backup_log_files = true,
                               uint64_t _backup_rate_limit = 0,
                               uint64_t _restore_rate_limit = 0,
                               int _max_background_operations = 1)
      : backup_dir(_backup_dir),
        backup_env(_backup_env),
        share_table_files(_share_table_files),
        info_log(_info_log),
        sync(_sync),
        destroy_old_data(_destroy_old_data),
        backup_log_files(_backup_log_files),
        backup_rate_limit(_backup_rate_limit),
        restore_rate_limit(_restore_rate_limit),
        share_files_with_checksum(false),
        max_background_operations(_max_background_operations) {
    assert(share_table_files || !share_files_with_checksum);
  }
};

struct RestoreOptions {
  // If true, restore won't overwrite the existing log files in wal_dir. It will
  // also move all log files from archive directory to wal_dir. Use this option
  // in combination with BackupableDBOptions::backup_log_files = false for
  // persisting in-memory databases.
  // Default: false
  bool keep_log_files;

  explicit RestoreOptions(bool _keep_log_files = false)
      : keep_log_files(_keep_log_files) {}
};

typedef uint32_t BackupID;

struct BackupInfo {
  BackupID backup_id;
  int64_t timestamp;
  uint64_t size;

  uint32_t number_files;

  BackupInfo() {}

  BackupInfo(BackupID _backup_id, int64_t _timestamp, uint64_t _size,
             uint32_t _number_files)
      : backup_id(_backup_id), timestamp(_timestamp), size(_size),
        number_files(_number_files) {}
};

class BackupStatistics {
 public:
  BackupStatistics() {
    number_success_backup = 0;
    number_fail_backup = 0;
  }

  BackupStatistics(uint32_t _number_success_backup,
                   uint32_t _number_fail_backup)
      : number_success_backup(_number_success_backup),
        number_fail_backup(_number_fail_backup) {}

  ~BackupStatistics() {}

  void IncrementNumberSuccessBackup();
  void IncrementNumberFailBackup();

  uint32_t GetNumberSuccessBackup() const;
  uint32_t GetNumberFailBackup() const;

  std::string ToString() const;

 private:
  uint32_t number_success_backup;
  uint32_t number_fail_backup;
};

class BackupEngineReadOnly {
 public:
  virtual ~BackupEngineReadOnly() {}

  static Status Open(Env* db_env, const BackupableDBOptions& options,
                     BackupEngineReadOnly** backup_engine_ptr);

  // You can GetBackupInfo safely, even with other BackupEngine performing
  // backups on the same directory
  virtual void GetBackupInfo(std::vector<BackupInfo>* backup_info) = 0;
  virtual void GetCorruptedBackups(
      std::vector<BackupID>* corrupt_backup_ids) = 0;

  // Restoring DB from backup is NOT safe when there is another BackupEngine
  // running that might call DeleteBackup() or PurgeOldBackups(). It is caller's
  // responsibility to synchronize the operation, i.e. don't delete the backup
  // when you're restoring from it
  virtual Status RestoreDBFromBackup(
      BackupID backup_id, const std::string& db_dir, const std::string& wal_dir,
      const RestoreOptions& restore_options = RestoreOptions()) = 0;
  virtual Status RestoreDBFromLatestBackup(
      const std::string& db_dir, const std::string& wal_dir,
      const RestoreOptions& restore_options = RestoreOptions()) = 0;

  // checks that each file exists and that the size of the file matches our
  // expectations. it does not check file checksum.
  // Returns Status::OK() if all checks are good
  virtual Status VerifyBackup(BackupID backup_id) = 0;
};

// Please see the documentation in BackupableDB and RestoreBackupableDB
class BackupEngine {
 public:
  virtual ~BackupEngine() {}

  static Status Open(Env* db_env,
                     const BackupableDBOptions& options,
                     BackupEngine** backup_engine_ptr);

  virtual Status CreateNewBackup(DB* db, bool flush_before_backup = false) = 0;
  virtual Status PurgeOldBackups(uint32_t num_backups_to_keep) = 0;
  virtual Status DeleteBackup(BackupID backup_id) = 0;
  virtual void StopBackup() = 0;

  virtual void GetBackupInfo(std::vector<BackupInfo>* backup_info) = 0;
  virtual void GetCorruptedBackups(
      std::vector<BackupID>* corrupt_backup_ids) = 0;
  virtual Status RestoreDBFromBackup(
      BackupID backup_id, const std::string& db_dir, const std::string& wal_dir,
      const RestoreOptions& restore_options = RestoreOptions()) = 0;
  virtual Status RestoreDBFromLatestBackup(
      const std::string& db_dir, const std::string& wal_dir,
      const RestoreOptions& restore_options = RestoreOptions()) = 0;

  // checks that each file exists and that the size of the file matches our
  // expectations. it does not check file checksum.
  // Returns Status::OK() if all checks are good
  virtual Status VerifyBackup(BackupID backup_id) = 0;

  virtual Status GarbageCollect() = 0;
};

// Stack your DB with BackupableDB to be able to backup the DB
class BackupableDB : public StackableDB {
 public:
  // BackupableDBOptions have to be the same as the ones used in a previous
  // incarnation of the DB
  //
  // BackupableDB ownes the pointer `DB* db` now. You should not delete it or
  // use it after the invocation of BackupableDB
  BackupableDB(DB* db, const BackupableDBOptions& options);
  virtual ~BackupableDB();

  // Captures the state of the database in the latest backup
  // NOT a thread safe call
  Status CreateNewBackup(bool flush_before_backup = false);
  // Returns info about backups in backup_info
  void GetBackupInfo(std::vector<BackupInfo>* backup_info);
  // Returns info about corrupt backups in corrupt_backups
  void GetCorruptedBackups(std::vector<BackupID>* corrupt_backup_ids);
  // deletes old backups, keeping latest num_backups_to_keep alive
  Status PurgeOldBackups(uint32_t num_backups_to_keep);
  // deletes a specific backup
  Status DeleteBackup(BackupID backup_id);
  // Call this from another thread if you want to stop the backup
  // that is currently happening. It will return immediatelly, will
  // not wait for the backup to stop.
  // The backup will stop ASAP and the call to CreateNewBackup will
  // return Status::Incomplete(). It will not clean up after itself, but
  // the state will remain consistent. The state will be cleaned up
  // next time you create BackupableDB or RestoreBackupableDB.
  void StopBackup();

  // Will delete all the files we don't need anymore
  // It will do the full scan of the files/ directory and delete all the
  // files that are not referenced.
  Status GarbageCollect();

 private:
  BackupEngine* backup_engine_;
  Status status_;
};

// Use this class to access information about backups and restore from them
class RestoreBackupableDB {
 public:
  RestoreBackupableDB(Env* db_env, const BackupableDBOptions& options);
  ~RestoreBackupableDB();

  // Returns info about backups in backup_info
  void GetBackupInfo(std::vector<BackupInfo>* backup_info);
  // Returns info about corrupt backups in corrupt_backups
  void GetCorruptedBackups(std::vector<BackupID>* corrupt_backup_ids);

  // restore from backup with backup_id
  // IMPORTANT -- if options_.share_table_files == true and you restore DB
  // from some backup that is not the latest, and you start creating new
  // backups from the new DB, they will probably fail
  //
  // Example: Let's say you have backups 1, 2, 3, 4, 5 and you restore 3.
  // If you add new data to the DB and try creating a new backup now, the
  // database will diverge from backups 4 and 5 and the new backup will fail.
  // If you want to create new backup, you will first have to delete backups 4
  // and 5.
  Status RestoreDBFromBackup(BackupID backup_id, const std::string& db_dir,
                             const std::string& wal_dir,
                             const RestoreOptions& restore_options =
                                 RestoreOptions());

  // restore from the latest backup
  Status RestoreDBFromLatestBackup(const std::string& db_dir,
                                   const std::string& wal_dir,
                                   const RestoreOptions& restore_options =
                                       RestoreOptions());
  // deletes old backups, keeping latest num_backups_to_keep alive
  Status PurgeOldBackups(uint32_t num_backups_to_keep);
  // deletes a specific backup
  Status DeleteBackup(BackupID backup_id);

  // Will delete all the files we don't need anymore
  // It will do the full scan of the files/ directory and delete all the
  // files that are not referenced.
  Status GarbageCollect();

 private:
  BackupEngine* backup_engine_;
  Status status_;
};

}  // namespace rocksdb
#endif  // ROCKSDB_LITE
