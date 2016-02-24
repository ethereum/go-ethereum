//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include <algorithm>
#include <map>
#include <string>
#include <tuple>

#include "db/compaction_job.h"
#include "db/column_family.h"
#include "db/version_set.h"
#include "db/writebuffer.h"
#include "rocksdb/cache.h"
#include "rocksdb/db.h"
#include "rocksdb/options.h"
#include "table/mock_table.h"
#include "util/file_reader_writer.h"
#include "util/string_util.h"
#include "util/testharness.h"
#include "util/testutil.h"
#include "utilities/merge_operators.h"

namespace rocksdb {

namespace {

void VerifyInitializationOfCompactionJobStats(
      const CompactionJobStats& compaction_job_stats) {
#if !defined(IOS_CROSS_COMPILE)
  ASSERT_EQ(compaction_job_stats.elapsed_micros, 0U);

  ASSERT_EQ(compaction_job_stats.num_input_records, 0U);
  ASSERT_EQ(compaction_job_stats.num_input_files, 0U);
  ASSERT_EQ(compaction_job_stats.num_input_files_at_output_level, 0U);

  ASSERT_EQ(compaction_job_stats.num_output_records, 0U);
  ASSERT_EQ(compaction_job_stats.num_output_files, 0U);

  ASSERT_EQ(compaction_job_stats.is_manual_compaction, true);

  ASSERT_EQ(compaction_job_stats.total_input_bytes, 0U);
  ASSERT_EQ(compaction_job_stats.total_output_bytes, 0U);

  ASSERT_EQ(compaction_job_stats.total_input_raw_key_bytes, 0U);
  ASSERT_EQ(compaction_job_stats.total_input_raw_value_bytes, 0U);

  ASSERT_EQ(compaction_job_stats.smallest_output_key_prefix[0], 0);
  ASSERT_EQ(compaction_job_stats.largest_output_key_prefix[0], 0);

  ASSERT_EQ(compaction_job_stats.num_records_replaced, 0U);

  ASSERT_EQ(compaction_job_stats.num_input_deletion_records, 0U);
  ASSERT_EQ(compaction_job_stats.num_expired_deletion_records, 0U);

  ASSERT_EQ(compaction_job_stats.num_corrupt_keys, 0U);
#endif  // !defined(IOS_CROSS_COMPILE)
}

}  // namespace

// TODO(icanadi) Make it simpler once we mock out VersionSet
class CompactionJobTest : public testing::Test {
 public:
  CompactionJobTest()
      : env_(Env::Default()),
        dbname_(test::TmpDir() + "/compaction_job_test"),
        mutable_cf_options_(Options(), ImmutableCFOptions(Options())),
        table_cache_(NewLRUCache(50000, 16)),
        write_buffer_(db_options_.db_write_buffer_size),
        versions_(new VersionSet(dbname_, &db_options_, env_options_,
                                 table_cache_.get(), &write_buffer_,
                                 &write_controller_)),
        shutting_down_(false),
        mock_table_factory_(new mock::MockTableFactory()) {
    EXPECT_OK(env_->CreateDirIfMissing(dbname_));
    db_options_.db_paths.emplace_back(dbname_,
                                      std::numeric_limits<uint64_t>::max());
  }

  std::string GenerateFileName(uint64_t file_number) {
    FileMetaData meta;
    std::vector<DbPath> db_paths;
    db_paths.emplace_back(dbname_, std::numeric_limits<uint64_t>::max());
    meta.fd = FileDescriptor(file_number, 0, 0);
    return TableFileName(db_paths, meta.fd.GetNumber(), meta.fd.GetPathId());
  }

  std::string KeyStr(const std::string& user_key, const SequenceNumber seq_num,
      const ValueType t) {
    return InternalKey(user_key, seq_num, t).Encode().ToString();
  }

  void AddMockFile(const stl_wrappers::KVMap& contents, int level = 0) {
    assert(contents.size() > 0);

    bool first_key = true;
    std::string smallest, largest;
    InternalKey smallest_key, largest_key;
    SequenceNumber smallest_seqno = kMaxSequenceNumber;
    SequenceNumber largest_seqno = 0;
    for (auto kv : contents) {
      ParsedInternalKey key;
      std::string skey;
      std::string value;
      std::tie(skey, value) = kv;
      ParseInternalKey(skey, &key);

      smallest_seqno = std::min(smallest_seqno, key.sequence);
      largest_seqno = std::max(largest_seqno, key.sequence);

      if (first_key ||
          cfd_->user_comparator()->Compare(key.user_key, smallest) < 0) {
        smallest.assign(key.user_key.data(), key.user_key.size());
        smallest_key.DecodeFrom(skey);
      }
      if (first_key ||
          cfd_->user_comparator()->Compare(key.user_key, largest) > 0) {
        largest.assign(key.user_key.data(), key.user_key.size());
        largest_key.DecodeFrom(skey);
      }

      first_key = false;
    }

    uint64_t file_number = versions_->NewFileNumber();
    EXPECT_OK(mock_table_factory_->CreateMockTable(
        env_, GenerateFileName(file_number), std::move(contents)));

    VersionEdit edit;
    edit.AddFile(level, file_number, 0, 10, smallest_key, largest_key,
        smallest_seqno, largest_seqno, false);

    mutex_.Lock();
    versions_->LogAndApply(versions_->GetColumnFamilySet()->GetDefault(),
                           mutable_cf_options_, &edit, &mutex_);
    mutex_.Unlock();
  }

  void SetLastSequence(const SequenceNumber sequence_number) {
    versions_->SetLastSequence(sequence_number + 1);
  }

  // returns expected result after compaction
  stl_wrappers::KVMap CreateTwoFiles(bool gen_corrupted_keys) {
    auto expected_results = mock::MakeMockFile();
    const int kKeysPerFile = 10000;
    const int kCorruptKeysPerFile = 200;
    const int kMatchingKeys = kKeysPerFile / 2;
    SequenceNumber sequence_number = 0;

    auto corrupt_id = [&](int id) {
      return gen_corrupted_keys && id > 0 && id <= kCorruptKeysPerFile;
    };

    for (int i = 0; i < 2; ++i) {
      auto contents = mock::MakeMockFile();
      for (int k = 0; k < kKeysPerFile; ++k) {
        auto key = ToString(i * kMatchingKeys + k);
        auto value = ToString(i * kKeysPerFile + k);
        InternalKey internal_key(key, ++sequence_number, kTypeValue);
        // This is how the key will look like once it's written in bottommost
        // file
        InternalKey bottommost_internal_key(key, 0, kTypeValue);
        if (corrupt_id(k)) {
          test::CorruptKeyType(&internal_key);
          test::CorruptKeyType(&bottommost_internal_key);
        }
        contents.insert({ internal_key.Encode().ToString(), value });
        if (i == 1 || k < kMatchingKeys || corrupt_id(k - kMatchingKeys)) {
          expected_results.insert(
              { bottommost_internal_key.Encode().ToString(), value });
        }
      }

      AddMockFile(contents);
    }

    SetLastSequence(sequence_number);

    return expected_results;
  }

  void NewDB(std::shared_ptr<MergeOperator> merge_operator = nullptr) {
    VersionEdit new_db;
    new_db.SetLogNumber(0);
    new_db.SetNextFile(2);
    new_db.SetLastSequence(0);

    const std::string manifest = DescriptorFileName(dbname_, 1);
    unique_ptr<WritableFile> file;
    Status s = env_->NewWritableFile(
        manifest, &file, env_->OptimizeForManifestWrite(env_options_));
    ASSERT_OK(s);
    unique_ptr<WritableFileWriter> file_writer(
        new WritableFileWriter(std::move(file), env_options_));
    {
      log::Writer log(std::move(file_writer));
      std::string record;
      new_db.EncodeTo(&record);
      s = log.AddRecord(record);
    }
    ASSERT_OK(s);
    // Make "CURRENT" file that points to the new manifest file.
    s = SetCurrentFile(env_, dbname_, 1, nullptr);

    std::vector<ColumnFamilyDescriptor> column_families;
    cf_options_.table_factory = mock_table_factory_;
    cf_options_.merge_operator = merge_operator;
    column_families.emplace_back(kDefaultColumnFamilyName, cf_options_);

    EXPECT_OK(versions_->Recover(column_families, false));
    cfd_ = versions_->GetColumnFamilySet()->GetDefault();
  }

  void RunCompaction(const std::vector<std::vector<FileMetaData*>>& input_files,
                     const stl_wrappers::KVMap& expected_results) {
    auto cfd = versions_->GetColumnFamilySet()->GetDefault();

    size_t num_input_files = 0;
    std::vector<CompactionInputFiles> compaction_input_files;
    for (size_t level = 0; level < input_files.size(); level++) {
      auto level_files = input_files[level];
      CompactionInputFiles compaction_level;
      compaction_level.level = static_cast<int>(level);
      compaction_level.files.insert(compaction_level.files.end(),
          level_files.begin(), level_files.end());
      compaction_input_files.push_back(compaction_level);
      num_input_files += level_files.size();
    }

    Compaction compaction(cfd->current()->storage_info(),
                          *cfd->GetLatestMutableCFOptions(),
                          compaction_input_files, 1, 1024 * 1024, 10, 0,
                          kNoCompression, {}, true);
    compaction.SetInputVersion(cfd->current());

    LogBuffer log_buffer(InfoLogLevel::INFO_LEVEL, db_options_.info_log.get());
    mutex_.Lock();
    EventLogger event_logger(db_options_.info_log.get());
    CompactionJob compaction_job(0, &compaction, db_options_, env_options_,
                                 versions_.get(), &shutting_down_, &log_buffer,
                                 nullptr, nullptr, nullptr, {}, table_cache_,
                                 &event_logger, false, false, dbname_,
                                 &compaction_job_stats_);

    VerifyInitializationOfCompactionJobStats(compaction_job_stats_);

    compaction_job.Prepare();
    mutex_.Unlock();
    Status s;
    s = compaction_job.Run();
    ASSERT_OK(s);
    mutex_.Lock();
    ASSERT_OK(compaction_job.Install(*cfd->GetLatestMutableCFOptions(),
                                     &mutex_));
    mutex_.Unlock();

    ASSERT_GE(compaction_job_stats_.elapsed_micros, 0U);
    ASSERT_EQ(compaction_job_stats_.num_input_files, num_input_files);
    ASSERT_EQ(compaction_job_stats_.num_output_files, 1U);
    mock_table_factory_->AssertLatestFile(expected_results);
  }

  Env* env_;
  std::string dbname_;
  EnvOptions env_options_;
  MutableCFOptions mutable_cf_options_;
  std::shared_ptr<Cache> table_cache_;
  WriteController write_controller_;
  DBOptions db_options_;
  ColumnFamilyOptions cf_options_;
  WriteBuffer write_buffer_;
  std::unique_ptr<VersionSet> versions_;
  InstrumentedMutex mutex_;
  std::atomic<bool> shutting_down_;
  std::shared_ptr<mock::MockTableFactory> mock_table_factory_;
  CompactionJobStats compaction_job_stats_;
  ColumnFamilyData* cfd_;
};

TEST_F(CompactionJobTest, Simple) {
  NewDB();

  auto expected_results = CreateTwoFiles(false);
  auto cfd = versions_->GetColumnFamilySet()->GetDefault();
  auto files = cfd->current()->storage_info()->LevelFiles(0);
  ASSERT_EQ(2U, files.size());
  RunCompaction({ files }, expected_results);
}

TEST_F(CompactionJobTest, SimpleCorrupted) {
  NewDB();

  auto expected_results = CreateTwoFiles(true);
  auto cfd = versions_->GetColumnFamilySet()->GetDefault();
  auto files = cfd->current()->storage_info()->LevelFiles(0);
  RunCompaction({ files }, expected_results);
  ASSERT_EQ(compaction_job_stats_.num_corrupt_keys, 400U);
}

TEST_F(CompactionJobTest, SimpleDeletion) {
  NewDB();

  auto file1 = mock::MakeMockFile({{KeyStr("c", 4U, kTypeDeletion), ""},
                                   {KeyStr("c", 3U, kTypeValue), "val"}});
  AddMockFile(file1);

  auto file2 = mock::MakeMockFile({{KeyStr("b", 2U, kTypeValue), "val"},
                                   {KeyStr("b", 1U, kTypeValue), "val"}});
  AddMockFile(file2);

  auto expected_results =
      mock::MakeMockFile({{KeyStr("b", 0U, kTypeValue), "val"}});

  SetLastSequence(4U);
  auto files = cfd_->current()->storage_info()->LevelFiles(0);
  RunCompaction({ files }, expected_results);
}

TEST_F(CompactionJobTest, SimpleOverwrite) {
  NewDB();

  auto file1 = mock::MakeMockFile({
      {KeyStr("a", 3U, kTypeValue), "val2"},
      {KeyStr("b", 4U, kTypeValue), "val3"},
  });
  AddMockFile(file1);

  auto file2 = mock::MakeMockFile({{KeyStr("a", 1U, kTypeValue), "val"},
                                   {KeyStr("b", 2U, kTypeValue), "val"}});
  AddMockFile(file2);

  auto expected_results =
      mock::MakeMockFile({{KeyStr("a", 0U, kTypeValue), "val2"},
                          {KeyStr("b", 0U, kTypeValue), "val3"}});

  SetLastSequence(4U);
  auto files = cfd_->current()->storage_info()->LevelFiles(0);
  RunCompaction({ files }, expected_results);
}

TEST_F(CompactionJobTest, SimpleNonLastLevel) {
  NewDB();

  auto file1 = mock::MakeMockFile({
      {KeyStr("a", 5U, kTypeValue), "val2"},
      {KeyStr("b", 6U, kTypeValue), "val3"},
  });
  AddMockFile(file1);

  auto file2 = mock::MakeMockFile({{KeyStr("a", 3U, kTypeValue), "val"},
                                   {KeyStr("b", 4U, kTypeValue), "val"}});
  AddMockFile(file2, 1);

  auto file3 = mock::MakeMockFile({{KeyStr("a", 1U, kTypeValue), "val"},
                                   {KeyStr("b", 2U, kTypeValue), "val"}});
  AddMockFile(file3, 2);

  // Because level 1 is not the last level, the sequence numbers of a and b
  // cannot be set to 0
  auto expected_results =
      mock::MakeMockFile({{KeyStr("a", 5U, kTypeValue), "val2"},
                          {KeyStr("b", 6U, kTypeValue), "val3"}});

  SetLastSequence(6U);
  auto lvl0_files = cfd_->current()->storage_info()->LevelFiles(0);
  auto lvl1_files = cfd_->current()->storage_info()->LevelFiles(1);
  RunCompaction({ lvl0_files, lvl1_files }, expected_results);
}

TEST_F(CompactionJobTest, SimpleMerge) {
  auto merge_op = MergeOperators::CreateStringAppendOperator();
  NewDB(merge_op);

  auto file1 = mock::MakeMockFile({
      {KeyStr("a", 5U, kTypeMerge), "5"},
      {KeyStr("a", 4U, kTypeMerge), "4"},
      {KeyStr("a", 3U, kTypeValue), "3"},
  });
  AddMockFile(file1);

  auto file2 = mock::MakeMockFile(
      {{KeyStr("b", 2U, kTypeMerge), "2"}, {KeyStr("b", 1U, kTypeValue), "1"}});
  AddMockFile(file2);

  auto expected_results =
      mock::MakeMockFile({{KeyStr("a", 0U, kTypeValue), "3,4,5"},
                          {KeyStr("b", 0U, kTypeValue), "1,2"}});

  SetLastSequence(5U);
  auto files = cfd_->current()->storage_info()->LevelFiles(0);
  RunCompaction({ files }, expected_results);
}

TEST_F(CompactionJobTest, NonAssocMerge) {
  auto merge_op = MergeOperators::CreateStringAppendTESTOperator();
  NewDB(merge_op);

  auto file1 = mock::MakeMockFile({
      {KeyStr("a", 5U, kTypeMerge), "5"},
      {KeyStr("a", 4U, kTypeMerge), "4"},
      {KeyStr("a", 3U, kTypeMerge), "3"},
  });
  AddMockFile(file1);

  auto file2 = mock::MakeMockFile(
      {{KeyStr("b", 2U, kTypeMerge), "2"}, {KeyStr("b", 1U, kTypeMerge), "1"}});
  AddMockFile(file2);

  auto expected_results =
      mock::MakeMockFile({{KeyStr("a", 0U, kTypeValue), "3,4,5"},
                          {KeyStr("b", 2U, kTypeMerge), "2"},
                          {KeyStr("b", 1U, kTypeMerge), "1"}});

  SetLastSequence(5U);
  auto files = cfd_->current()->storage_info()->LevelFiles(0);
  RunCompaction({ files }, expected_results);
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
