//  Copyright (c) 2015, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright 2014 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

// This test uses a custom Env to keep track of the state of a filesystem as of
// the last "sync". It then checks for data loss errors by purposely dropping
// file data (or entire files) not protected by a "sync".

#if !(defined NDEBUG) || !defined(OS_WIN)

#include <map>
#include <set>
#include "db/db_impl.h"
#include "db/filename.h"
#include "db/log_format.h"
#include "db/version_set.h"
#include "rocksdb/cache.h"
#include "rocksdb/db.h"
#include "rocksdb/env.h"
#include "rocksdb/table.h"
#include "rocksdb/write_batch.h"
#include "util/logging.h"
#include "util/mock_env.h"
#include "util/mutexlock.h"
#include "util/sync_point.h"
#include "util/testharness.h"
#include "util/testutil.h"

namespace rocksdb {

static const int kValueSize = 1000;
static const int kMaxNumValues = 2000;
static const size_t kNumIterations = 3;

class TestWritableFile;
class FaultInjectionTestEnv;

namespace {

// Assume a filename, and not a directory name like "/foo/bar/"
static std::string GetDirName(const std::string filename) {
  size_t found = filename.find_last_of("/\\");
  if (found == std::string::npos) {
    return "";
  } else {
    return filename.substr(0, found);
  }
}

// Trim the tailing "/" in the end of `str`
static std::string TrimDirname(const std::string& str) {
  size_t found = str.find_last_not_of("/");
  if (found == std::string::npos) {
    return str;
  }
  return str.substr(0, found + 1);
}

// Return pair <parent directory name, file name> of a full path.
static std::pair<std::string, std::string> GetDirAndName(
    const std::string& name) {
  std::string dirname = GetDirName(name);
  std::string fname = name.substr(dirname.size() + 1);
  return std::make_pair(dirname, fname);
}

// A basic file truncation function suitable for this test.
Status Truncate(Env* env, const std::string& filename, uint64_t length) {
  unique_ptr<SequentialFile> orig_file;
  const EnvOptions options;
  Status s = env->NewSequentialFile(filename, &orig_file, options);
  if (!s.ok()) {
    fprintf(stderr, "Cannot truncate file %s: %s\n", filename.c_str(),
            s.ToString().c_str());
    return s;
  }

  std::unique_ptr<char[]> scratch(new char[length]);
  rocksdb::Slice result;
  s = orig_file->Read(length, &result, scratch.get());
#ifdef OS_WIN
  orig_file.reset();
#endif
  if (s.ok()) {
    std::string tmp_name = GetDirName(filename) + "/truncate.tmp";
    unique_ptr<WritableFile> tmp_file;
    s = env->NewWritableFile(tmp_name, &tmp_file, options);
    if (s.ok()) {
      s = tmp_file->Append(result);
      if (s.ok()) {
        s = env->RenameFile(tmp_name, filename);
      } else {
        fprintf(stderr, "Cannot rename file %s to %s: %s\n", tmp_name.c_str(),
                filename.c_str(), s.ToString().c_str());
        env->DeleteFile(tmp_name);
      }
    }
  }
  if (!s.ok()) {
    fprintf(stderr, "Cannot truncate file %s: %s\n", filename.c_str(),
            s.ToString().c_str());
  }

  return s;
}

struct FileState {
  std::string filename_;
  ssize_t pos_;
  ssize_t pos_at_last_sync_;
  ssize_t pos_at_last_flush_;

  explicit FileState(const std::string& filename)
      : filename_(filename),
        pos_(-1),
        pos_at_last_sync_(-1),
        pos_at_last_flush_(-1) { }

  FileState() : pos_(-1), pos_at_last_sync_(-1), pos_at_last_flush_(-1) {}

  bool IsFullySynced() const { return pos_ <= 0 || pos_ == pos_at_last_sync_; }

  Status DropUnsyncedData(Env* env) const;

  Status DropRandomUnsyncedData(Env* env, Random* rand) const;
};

}  // anonymous namespace

// A wrapper around WritableFileWriter* file
// is written to or sync'ed.
class TestWritableFile : public WritableFile {
 public:
  explicit TestWritableFile(const std::string& fname,
                            unique_ptr<WritableFile>&& f,
                            FaultInjectionTestEnv* env);
  virtual ~TestWritableFile();
  virtual Status Append(const Slice& data) override;
  virtual Status Close() override;
  virtual Status Flush() override;
  virtual Status Sync() override;
  virtual bool IsSyncThreadSafe() const override { return true; }

 private:
  FileState state_;
  unique_ptr<WritableFile> target_;
  bool writable_file_opened_;
  FaultInjectionTestEnv* env_;
};

class TestDirectory : public Directory {
 public:
  explicit TestDirectory(FaultInjectionTestEnv* env, std::string dirname,
                         Directory* dir)
      : env_(env), dirname_(dirname), dir_(dir) {}
  ~TestDirectory() {}

  virtual Status Fsync() override;

 private:
  FaultInjectionTestEnv* env_;
  std::string dirname_;
  unique_ptr<Directory> dir_;
};

class FaultInjectionTestEnv : public EnvWrapper {
 public:
  explicit FaultInjectionTestEnv(Env* base)
      : EnvWrapper(base),
        filesystem_active_(true) {}
  virtual ~FaultInjectionTestEnv() { }

  Status NewDirectory(const std::string& name,
                      unique_ptr<Directory>* result) override {
    unique_ptr<Directory> r;
    Status s = target()->NewDirectory(name, &r);
    EXPECT_OK(s);
    if (!s.ok()) {
      return s;
    }
    result->reset(new TestDirectory(this, TrimDirname(name), r.release()));
    return Status::OK();
  }

  Status NewWritableFile(const std::string& fname,
                         unique_ptr<WritableFile>* result,
                         const EnvOptions& soptions) override {
    if (!IsFilesystemActive()) {
      return Status::Corruption("Not Active");
    }
    // Not allow overwriting files
    Status s = target()->FileExists(fname);
    if (s.ok()) {
      return Status::Corruption("File already exists.");
    } else if (!s.IsNotFound()) {
      assert(s.IsIOError());
      return s;
    }
    s = target()->NewWritableFile(fname, result, soptions);
    if (s.ok()) {
      result->reset(new TestWritableFile(fname, std::move(*result), this));
      // WritableFileWriter* file is opened
      // again then it will be truncated - so forget our saved state.
      UntrackFile(fname);
      MutexLock l(&mutex_);
      open_files_.insert(fname);
      auto dir_and_name = GetDirAndName(fname);
      auto& list = dir_to_new_files_since_last_sync_[dir_and_name.first];
      list.insert(dir_and_name.second);
    }
    return s;
  }

  virtual Status DeleteFile(const std::string& f) override {
    if (!IsFilesystemActive()) {
      return Status::Corruption("Not Active");
    }
    Status s = EnvWrapper::DeleteFile(f);
    if (!s.ok()) {
      fprintf(stderr, "Cannot delete file %s: %s\n", f.c_str(),
              s.ToString().c_str());
    }
    EXPECT_OK(s);
    if (s.ok()) {
      UntrackFile(f);
    }
    return s;
  }

  virtual Status RenameFile(const std::string& s,
                            const std::string& t) override {
    if (!IsFilesystemActive()) {
      return Status::Corruption("Not Active");
    }
    Status ret = EnvWrapper::RenameFile(s, t);

    if (ret.ok()) {
      MutexLock l(&mutex_);
      if (db_file_state_.find(s) != db_file_state_.end()) {
        db_file_state_[t] = db_file_state_[s];
        db_file_state_.erase(s);
      }

      auto sdn = GetDirAndName(s);
      auto tdn = GetDirAndName(t);
      if (dir_to_new_files_since_last_sync_[sdn.first].erase(sdn.second) != 0) {
        auto& tlist = dir_to_new_files_since_last_sync_[tdn.first];
        assert(tlist.find(tdn.second) == tlist.end());
        tlist.insert(tdn.second);
      }
    }

    return ret;
  }

  void WritableFileClosed(const FileState& state) {
    MutexLock l(&mutex_);
    if (open_files_.find(state.filename_) != open_files_.end()) {
      db_file_state_[state.filename_] = state;
      open_files_.erase(state.filename_);
    }
  }

  // For every file that is not fully synced, make a call to `func` with
  // FileState of the file as the parameter.
  Status DropFileData(std::function<Status(Env*, FileState)> func) {
    Status s;
    MutexLock l(&mutex_);
    for (std::map<std::string, FileState>::const_iterator it =
             db_file_state_.begin();
         s.ok() && it != db_file_state_.end(); ++it) {
      const FileState& state = it->second;
      if (!state.IsFullySynced()) {
        s = func(target(), state);
      }
    }
    return s;
  }

  Status DropUnsyncedFileData() {
    return DropFileData([&](Env* env, const FileState& state) {
      return state.DropUnsyncedData(env);
    });
  }

  Status DropRandomUnsyncedFileData(Random* rnd) {
    return DropFileData([&](Env* env, const FileState& state) {
      return state.DropRandomUnsyncedData(env, rnd);
    });
  }

  Status DeleteFilesCreatedAfterLastDirSync() {
    // Because DeleteFile access this container make a copy to avoid deadlock
    std::map<std::string, std::set<std::string>> map_copy;
    {
      MutexLock l(&mutex_);
      map_copy.insert(dir_to_new_files_since_last_sync_.begin(),
                      dir_to_new_files_since_last_sync_.end());
    }

    for (auto& pair : map_copy) {
      for (std::string name : pair.second) {
        Status s = DeleteFile(pair.first + "/" + name);
        if (!s.ok()) {
          return s;
        }
      }
    }
    return Status::OK();
  }
  void ResetState() {
    MutexLock l(&mutex_);
    db_file_state_.clear();
    dir_to_new_files_since_last_sync_.clear();
    SetFilesystemActiveNoLock(true);
  }

  void UntrackFile(const std::string& f) {
    MutexLock l(&mutex_);
    auto dir_and_name = GetDirAndName(f);
    dir_to_new_files_since_last_sync_[dir_and_name.first].erase(
        dir_and_name.second);
    db_file_state_.erase(f);
    open_files_.erase(f);
  }

  void SyncDir(const std::string& dirname) {
    MutexLock l(&mutex_);
    dir_to_new_files_since_last_sync_.erase(dirname);
  }

  // Setting the filesystem to inactive is the test equivalent to simulating a
  // system reset. Setting to inactive will freeze our saved filesystem state so
  // that it will stop being recorded. It can then be reset back to the state at
  // the time of the reset.
  bool IsFilesystemActive() {
    MutexLock l(&mutex_);
    return filesystem_active_;
  }
  void SetFilesystemActiveNoLock(bool active) { filesystem_active_ = active; }
  void SetFilesystemActive(bool active) {
    MutexLock l(&mutex_);
    SetFilesystemActiveNoLock(active);
  }
  void AssertNoOpenFile() { ASSERT_TRUE(open_files_.empty()); }

 private:
  port::Mutex mutex_;
  std::map<std::string, FileState> db_file_state_;
  std::set<std::string> open_files_;
  std::unordered_map<std::string, std::set<std::string>>
      dir_to_new_files_since_last_sync_;
  bool filesystem_active_;  // Record flushes, syncs, writes
};

Status FileState::DropUnsyncedData(Env* env) const {
  ssize_t sync_pos = pos_at_last_sync_ == -1 ? 0 : pos_at_last_sync_;
  return Truncate(env, filename_, sync_pos);
}

Status FileState::DropRandomUnsyncedData(Env* env, Random* rand) const {
  ssize_t sync_pos = pos_at_last_sync_ == -1 ? 0 : pos_at_last_sync_;
  assert(pos_ >= sync_pos);
  int range = static_cast<int>(pos_ - sync_pos);
  uint64_t truncated_size =
      static_cast<uint64_t>(sync_pos) + rand->Uniform(range);
  return Truncate(env, filename_, truncated_size);
}

Status TestDirectory::Fsync() {
  env_->SyncDir(dirname_);
  return dir_->Fsync();
}

TestWritableFile::TestWritableFile(const std::string& fname,
                                   unique_ptr<WritableFile>&& f,
                                   FaultInjectionTestEnv* env)
      : state_(fname),
        target_(std::move(f)),
        writable_file_opened_(true),
        env_(env) {
  assert(target_ != nullptr);
  state_.pos_ = 0;
}

TestWritableFile::~TestWritableFile() {
  if (writable_file_opened_) {
    Close();
  }
}

Status TestWritableFile::Append(const Slice& data) {
  if (!env_->IsFilesystemActive()) {
    return Status::Corruption("Not Active");
  }
  Status s = target_->Append(data);
  if (s.ok()) {
    state_.pos_ += data.size();
  }
  return s;
}

Status TestWritableFile::Close() {
  writable_file_opened_ = false;
  Status s = target_->Close();
  if (s.ok()) {
    env_->WritableFileClosed(state_);
  }
  return s;
}

Status TestWritableFile::Flush() {
  Status s = target_->Flush();
  if (s.ok() && env_->IsFilesystemActive()) {
    state_.pos_at_last_flush_ = state_.pos_;
  }
  return s;
}

Status TestWritableFile::Sync() {
  if (!env_->IsFilesystemActive()) {
    return Status::OK();
  }
  // No need to actual sync.
  state_.pos_at_last_sync_ = state_.pos_;
  return Status::OK();
}

class FaultInjectionTest : public testing::Test,
                           public testing::WithParamInterface<bool> {
 protected:
  enum OptionConfig {
    kDefault,
    kDifferentDataDir,
    kWalDir,
    kSyncWal,
    kWalDirSyncWal,
    kMultiLevels,
    kEnd,
  };
  int option_config_;
  // When need to make sure data is persistent, sync WAL
  bool sync_use_wal_;
  // When need to make sure data is persistent, call DB::CompactRange()
  bool sync_use_compact_;

  bool sequential_order_;

 protected:
 public:
  enum ExpectedVerifResult { kValExpectFound, kValExpectNoError };
  enum ResetMethod {
    kResetDropUnsyncedData,
    kResetDropRandomUnsyncedData,
    kResetDeleteUnsyncedFiles,
    kResetDropAndDeleteUnsynced
  };

  std::unique_ptr<Env> base_env_;
  FaultInjectionTestEnv* env_;
  std::string dbname_;
  shared_ptr<Cache> tiny_cache_;
  Options options_;
  DB* db_;

  FaultInjectionTest()
      : option_config_(kDefault),
        sync_use_wal_(false),
        sync_use_compact_(true),
        base_env_(nullptr),
        env_(NULL),
        db_(NULL) {
  }

  ~FaultInjectionTest() {
    rocksdb::SyncPoint::GetInstance()->DisableProcessing();
    rocksdb::SyncPoint::GetInstance()->ClearAllCallBacks();
  }

  bool ChangeOptions() {
    option_config_++;
    if (option_config_ >= kEnd) {
      return false;
    } else {
      if (option_config_ == kMultiLevels) {
        base_env_.reset(new MockEnv(Env::Default()));
      }
      return true;
    }
  }

  // Return the current option configuration.
  Options CurrentOptions() {
    sync_use_wal_ = false;
    sync_use_compact_ = true;
    Options options;
    switch (option_config_) {
      case kWalDir:
        options.wal_dir = test::TmpDir(env_) + "/fault_test_wal";
        break;
      case kDifferentDataDir:
        options.db_paths.emplace_back(test::TmpDir(env_) + "/fault_test_data",
                                      1000000U);
        break;
      case kSyncWal:
        sync_use_wal_ = true;
        sync_use_compact_ = false;
        break;
      case kWalDirSyncWal:
        options.wal_dir = test::TmpDir(env_) + "/fault_test_wal";
        sync_use_wal_ = true;
        sync_use_compact_ = false;
        break;
      case kMultiLevels:
        options.write_buffer_size = 64 * 1024;
        options.target_file_size_base = 64 * 1024;
        options.level0_file_num_compaction_trigger = 2;
        options.level0_slowdown_writes_trigger = 2;
        options.level0_stop_writes_trigger = 4;
        options.max_bytes_for_level_base = 128 * 1024;
        options.max_write_buffer_number = 2;
        options.max_background_compactions = 8;
        options.max_background_flushes = 8;
        sync_use_wal_ = true;
        sync_use_compact_ = false;
        break;
      default:
        break;
    }
    return options;
  }

  Status NewDB() {
    assert(db_ == NULL);
    assert(tiny_cache_ == nullptr);
    assert(env_ == NULL);

    env_ =
        new FaultInjectionTestEnv(base_env_ ? base_env_.get() : Env::Default());

    options_ = CurrentOptions();
    options_.env = env_;
    options_.paranoid_checks = true;

    BlockBasedTableOptions table_options;
    tiny_cache_ = NewLRUCache(100);
    table_options.block_cache = tiny_cache_;
    options_.table_factory.reset(NewBlockBasedTableFactory(table_options));

    dbname_ = test::TmpDir() + "/fault_test";

    EXPECT_OK(DestroyDB(dbname_, options_));

    options_.create_if_missing = true;
    Status s = OpenDB();
    options_.create_if_missing = false;
    return s;
  }

  void SetUp() override {
    sequential_order_ = GetParam();
    ASSERT_OK(NewDB());
  }

  void TearDown() override {
    CloseDB();

    Status s = DestroyDB(dbname_, options_);

    delete env_;
    env_ = NULL;

    tiny_cache_.reset();

    ASSERT_OK(s);
  }

  void Build(const WriteOptions& write_options, int start_idx, int num_vals) {
    std::string key_space, value_space;
    WriteBatch batch;
    for (int i = start_idx; i < start_idx + num_vals; i++) {
      Slice key = Key(i, &key_space);
      batch.Clear();
      batch.Put(key, Value(i, &value_space));
      ASSERT_OK(db_->Write(write_options, &batch));
    }
  }

  Status ReadValue(int i, std::string* val) const {
    std::string key_space, value_space;
    Slice key = Key(i, &key_space);
    Value(i, &value_space);
    ReadOptions options;
    return db_->Get(options, key, val);
  }

  Status Verify(int start_idx, int num_vals,
                ExpectedVerifResult expected) const {
    std::string val;
    std::string value_space;
    Status s;
    for (int i = start_idx; i < start_idx + num_vals && s.ok(); i++) {
      Value(i, &value_space);
      s = ReadValue(i, &val);
      if (s.ok()) {
        EXPECT_EQ(value_space, val);
      }
      if (expected == kValExpectFound) {
        if (!s.ok()) {
          fprintf(stderr, "Error when read %dth record (expect found): %s\n", i,
                  s.ToString().c_str());
          return s;
        }
      } else if (!s.ok() && !s.IsNotFound()) {
        fprintf(stderr, "Error when read %dth record: %s\n", i,
                s.ToString().c_str());
        return s;
      }
    }
    return Status::OK();
  }

  // Return the ith key
  Slice Key(int i, std::string* storage) const {
    int num = i;
    if (!sequential_order_) {
      // random transfer
      const int m = 0x5bd1e995;
      num *= m;
      num ^= num << 24;
    }
    char buf[100];
    snprintf(buf, sizeof(buf), "%016d", num);
    storage->assign(buf, strlen(buf));
    return Slice(*storage);
  }

  // Return the value to associate with the specified key
  Slice Value(int k, std::string* storage) const {
    Random r(k);
    return test::RandomString(&r, kValueSize, storage);
  }

  Status OpenDB() {
    delete db_;
    db_ = NULL;
    env_->ResetState();
    return DB::Open(options_, dbname_, &db_);
  }

  void CloseDB() {
    delete db_;
    db_ = NULL;
  }

  void DeleteAllData() {
    Iterator* iter = db_->NewIterator(ReadOptions());
    WriteOptions options;
    for (iter->SeekToFirst(); iter->Valid(); iter->Next()) {
      ASSERT_OK(db_->Delete(WriteOptions(), iter->key()));
    }

    delete iter;

    FlushOptions flush_options;
    flush_options.wait = true;
    db_->Flush(flush_options);
  }

  // rnd cannot be null for kResetDropRandomUnsyncedData
  void ResetDBState(ResetMethod reset_method, Random* rnd = nullptr) {
    env_->AssertNoOpenFile();
    switch (reset_method) {
      case kResetDropUnsyncedData:
        ASSERT_OK(env_->DropUnsyncedFileData());
        break;
      case kResetDropRandomUnsyncedData:
        ASSERT_OK(env_->DropRandomUnsyncedFileData(rnd));
        break;
      case kResetDeleteUnsyncedFiles:
        ASSERT_OK(env_->DeleteFilesCreatedAfterLastDirSync());
        break;
      case kResetDropAndDeleteUnsynced:
        ASSERT_OK(env_->DropUnsyncedFileData());
        ASSERT_OK(env_->DeleteFilesCreatedAfterLastDirSync());
        break;
      default:
        assert(false);
    }
  }

  void PartialCompactTestPreFault(int num_pre_sync, int num_post_sync) {
    DeleteAllData();

    WriteOptions write_options;
    write_options.sync = sync_use_wal_;

    Build(write_options, 0, num_pre_sync);
    if (sync_use_compact_) {
      db_->CompactRange(CompactRangeOptions(), nullptr, nullptr);
    }
    write_options.sync = false;
    Build(write_options, num_pre_sync, num_post_sync);
  }

  void PartialCompactTestReopenWithFault(ResetMethod reset_method,
                                         int num_pre_sync, int num_post_sync,
                                         Random* rnd = nullptr) {
    env_->SetFilesystemActive(false);
    CloseDB();
    ResetDBState(reset_method, rnd);
    ASSERT_OK(OpenDB());
    ASSERT_OK(Verify(0, num_pre_sync, FaultInjectionTest::kValExpectFound));
    ASSERT_OK(Verify(num_pre_sync, num_post_sync,
                     FaultInjectionTest::kValExpectNoError));
    WaitCompactionFinish();
    ASSERT_OK(Verify(0, num_pre_sync, FaultInjectionTest::kValExpectFound));
    ASSERT_OK(Verify(num_pre_sync, num_post_sync,
                     FaultInjectionTest::kValExpectNoError));
  }

  void NoWriteTestPreFault() {
  }

  void NoWriteTestReopenWithFault(ResetMethod reset_method) {
    CloseDB();
    ResetDBState(reset_method);
    ASSERT_OK(OpenDB());
  }

  void WaitCompactionFinish() {
    static_cast<DBImpl*>(db_)->TEST_WaitForCompact();
    ASSERT_OK(db_->Put(WriteOptions(), "", ""));
  }
};

TEST_P(FaultInjectionTest, FaultTest) {
  do {
    Random rnd(301);

    for (size_t idx = 0; idx < kNumIterations; idx++) {
      int num_pre_sync = rnd.Uniform(kMaxNumValues);
      int num_post_sync = rnd.Uniform(kMaxNumValues);

      PartialCompactTestPreFault(num_pre_sync, num_post_sync);
      PartialCompactTestReopenWithFault(kResetDropUnsyncedData, num_pre_sync,
                                        num_post_sync);
      NoWriteTestPreFault();
      NoWriteTestReopenWithFault(kResetDropUnsyncedData);

      PartialCompactTestPreFault(num_pre_sync, num_post_sync);
      PartialCompactTestReopenWithFault(kResetDropRandomUnsyncedData,
                                        num_pre_sync, num_post_sync, &rnd);
      NoWriteTestPreFault();
      NoWriteTestReopenWithFault(kResetDropUnsyncedData);

      // Setting a separate data path won't pass the test as we don't sync
      // it after creating new files,
      PartialCompactTestPreFault(num_pre_sync, num_post_sync);
      PartialCompactTestReopenWithFault(kResetDropAndDeleteUnsynced,
                                        num_pre_sync, num_post_sync);
      NoWriteTestPreFault();
      NoWriteTestReopenWithFault(kResetDropAndDeleteUnsynced);

      PartialCompactTestPreFault(num_pre_sync, num_post_sync);
      // No new files created so we expect all values since no files will be
      // dropped.
      PartialCompactTestReopenWithFault(kResetDeleteUnsyncedFiles, num_pre_sync,
                                        num_post_sync);
      NoWriteTestPreFault();
      NoWriteTestReopenWithFault(kResetDeleteUnsyncedFiles);
    }
  } while (ChangeOptions());
}

class SleepingBackgroundTask {
 public:
  SleepingBackgroundTask()
      : bg_cv_(&mutex_), should_sleep_(true), done_with_sleep_(false) {}
  void DoSleep() {
    MutexLock l(&mutex_);
    while (should_sleep_) {
      bg_cv_.Wait();
    }
    done_with_sleep_ = true;
    bg_cv_.SignalAll();
  }
  void WakeUp() {
    MutexLock l(&mutex_);
    should_sleep_ = false;
    bg_cv_.SignalAll();
    while (!done_with_sleep_) {
      bg_cv_.Wait();
    }
  }

  static void DoSleepTask(void* arg) {
    reinterpret_cast<SleepingBackgroundTask*>(arg)->DoSleep();
  }

 private:
  port::Mutex mutex_;
  port::CondVar bg_cv_;  // Signalled when background work finishes
  bool should_sleep_;
  bool done_with_sleep_;
};

// Previous log file is not fsynced if sync is forced after log rolling.
TEST_P(FaultInjectionTest, WriteOptionSyncTest) {
  SleepingBackgroundTask sleeping_task_low;
  env_->SetBackgroundThreads(1, Env::HIGH);
  // Block the job queue to prevent flush job from running.
  env_->Schedule(&SleepingBackgroundTask::DoSleepTask, &sleeping_task_low,
                 Env::Priority::HIGH);

  WriteOptions write_options;
  write_options.sync = false;

  std::string key_space, value_space;
  ASSERT_OK(
      db_->Put(write_options, Key(1, &key_space), Value(1, &value_space)));
  FlushOptions flush_options;
  flush_options.wait = false;
  ASSERT_OK(db_->Flush(flush_options));
  write_options.sync = true;
  ASSERT_OK(
      db_->Put(write_options, Key(2, &key_space), Value(2, &value_space)));

  env_->SetFilesystemActive(false);
  NoWriteTestReopenWithFault(kResetDropAndDeleteUnsynced);
  sleeping_task_low.WakeUp();

  ASSERT_OK(OpenDB());
  std::string val;
  Value(2, &value_space);
  ASSERT_OK(ReadValue(2, &val));
  ASSERT_EQ(value_space, val);

  Value(1, &value_space);
  ASSERT_OK(ReadValue(1, &val));
  ASSERT_EQ(value_space, val);
}

TEST_P(FaultInjectionTest, UninstalledCompaction) {
  options_.target_file_size_base = 32 * 1024;
  options_.write_buffer_size = 100 << 10;  // 100KB
  options_.level0_file_num_compaction_trigger = 6;
  options_.level0_stop_writes_trigger = 1 << 10;
  options_.level0_slowdown_writes_trigger = 1 << 10;
  options_.max_background_compactions = 1;
  OpenDB();

  if (!sequential_order_) {
    rocksdb::SyncPoint::GetInstance()->LoadDependency({
        {"FaultInjectionTest::FaultTest:0", "DBImpl::BGWorkCompaction"},
        {"CompactionJob::Run():End", "FaultInjectionTest::FaultTest:1"},
        {"FaultInjectionTest::FaultTest:2",
         "DBImpl::BackgroundCompaction:NonTrivial:AfterRun"},
    });
  }
  rocksdb::SyncPoint::GetInstance()->EnableProcessing();

  int kNumKeys = 1000;
  Build(WriteOptions(), 0, kNumKeys);
  FlushOptions flush_options;
  flush_options.wait = true;
  db_->Flush(flush_options);
  ASSERT_OK(db_->Put(WriteOptions(), "", ""));
  TEST_SYNC_POINT("FaultInjectionTest::FaultTest:0");
  TEST_SYNC_POINT("FaultInjectionTest::FaultTest:1");
  env_->SetFilesystemActive(false);
  TEST_SYNC_POINT("FaultInjectionTest::FaultTest:2");
  CloseDB();
  rocksdb::SyncPoint::GetInstance()->DisableProcessing();
  ResetDBState(kResetDropUnsyncedData);

  std::atomic<bool> opened(false);
  rocksdb::SyncPoint::GetInstance()->SetCallBack(
      "DBImpl::Open:Opened", [&](void* arg) { opened.store(true); });
  rocksdb::SyncPoint::GetInstance()->SetCallBack(
      "DBImpl::BGWorkCompaction",
      [&](void* arg) { ASSERT_TRUE(opened.load()); });
  rocksdb::SyncPoint::GetInstance()->EnableProcessing();
  ASSERT_OK(OpenDB());
  ASSERT_OK(Verify(0, kNumKeys, FaultInjectionTest::kValExpectFound));
  WaitCompactionFinish();
  ASSERT_OK(Verify(0, kNumKeys, FaultInjectionTest::kValExpectFound));
  rocksdb::SyncPoint::GetInstance()->DisableProcessing();
  rocksdb::SyncPoint::GetInstance()->ClearAllCallBacks();
}

TEST_P(FaultInjectionTest, ManualLogSyncTest) {
  SleepingBackgroundTask sleeping_task_low;
  env_->SetBackgroundThreads(1, Env::HIGH);
  // Block the job queue to prevent flush job from running.
  env_->Schedule(&SleepingBackgroundTask::DoSleepTask, &sleeping_task_low,
                 Env::Priority::HIGH);

  WriteOptions write_options;
  write_options.sync = false;

  std::string key_space, value_space;
  ASSERT_OK(
      db_->Put(write_options, Key(1, &key_space), Value(1, &value_space)));
  FlushOptions flush_options;
  flush_options.wait = false;
  ASSERT_OK(db_->Flush(flush_options));
  ASSERT_OK(
      db_->Put(write_options, Key(2, &key_space), Value(2, &value_space)));
  ASSERT_OK(db_->SyncWAL());

  env_->SetFilesystemActive(false);
  NoWriteTestReopenWithFault(kResetDropAndDeleteUnsynced);
  sleeping_task_low.WakeUp();

  ASSERT_OK(OpenDB());
  std::string val;
  Value(2, &value_space);
  ASSERT_OK(ReadValue(2, &val));
  ASSERT_EQ(value_space, val);

  Value(1, &value_space);
  ASSERT_OK(ReadValue(1, &val));
  ASSERT_EQ(value_space, val);
}

INSTANTIATE_TEST_CASE_P(FaultTest, FaultInjectionTest, ::testing::Bool());

}  // namespace rocksdb

#endif // #if !(defined NDEBUG) || !defined(OS_WIN)

int main(int argc, char** argv) {
#if !(defined NDEBUG) || !defined(OS_WIN)
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
#else
  return 0;
#endif
}
