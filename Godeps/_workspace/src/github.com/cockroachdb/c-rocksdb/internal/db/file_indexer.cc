//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "db/file_indexer.h"
#include <algorithm>
#include "rocksdb/comparator.h"
#include "db/version_edit.h"

namespace rocksdb {

FileIndexer::FileIndexer(const Comparator* ucmp)
    : num_levels_(0), ucmp_(ucmp), level_rb_(nullptr) {}

size_t FileIndexer::NumLevelIndex() const { return next_level_index_.size(); }

size_t FileIndexer::LevelIndexSize(size_t level) const {
  if (level >= next_level_index_.size()) {
    return 0;
  }
  return next_level_index_[level].num_index;
}

void FileIndexer::GetNextLevelIndex(const size_t level, const size_t file_index,
                                    const int cmp_smallest,
                                    const int cmp_largest, int32_t* left_bound,
                                    int32_t* right_bound) const {
  assert(level > 0);

  // Last level, no hint
  if (level == num_levels_ - 1) {
    *left_bound = 0;
    *right_bound = -1;
    return;
  }

  assert(level < num_levels_ - 1);
  assert(static_cast<int32_t>(file_index) <= level_rb_[level]);

  const IndexUnit* index_units = next_level_index_[level].index_units;
  const auto& index = index_units[file_index];

  if (cmp_smallest < 0) {
    *left_bound = (level > 0 && file_index > 0)
                      ? index_units[file_index - 1].largest_lb
                      : 0;
    *right_bound = index.smallest_rb;
  } else if (cmp_smallest == 0) {
    *left_bound = index.smallest_lb;
    *right_bound = index.smallest_rb;
  } else if (cmp_smallest > 0 && cmp_largest < 0) {
    *left_bound = index.smallest_lb;
    *right_bound = index.largest_rb;
  } else if (cmp_largest == 0) {
    *left_bound = index.largest_lb;
    *right_bound = index.largest_rb;
  } else if (cmp_largest > 0) {
    *left_bound = index.largest_lb;
    *right_bound = level_rb_[level + 1];
  } else {
    assert(false);
  }

  assert(*left_bound >= 0);
  assert(*left_bound <= *right_bound + 1);
  assert(*right_bound <= level_rb_[level + 1]);
}

void FileIndexer::UpdateIndex(Arena* arena, const size_t num_levels,
                              std::vector<FileMetaData*>* const files) {
  if (files == nullptr) {
    return;
  }
  if (num_levels == 0) {  // uint_32 0-1 would cause bad behavior
    num_levels_ = num_levels;
    return;
  }
  assert(level_rb_ == nullptr);  // level_rb_ should be init here

  num_levels_ = num_levels;
  next_level_index_.resize(num_levels);

  char* mem = arena->AllocateAligned(num_levels_ * sizeof(int32_t));
  level_rb_ = new (mem) int32_t[num_levels_];
  for (size_t i = 0; i < num_levels_; i++) {
    level_rb_[i] = -1;
  }

  // L1 - Ln-1
  for (size_t level = 1; level < num_levels_ - 1; ++level) {
    const auto& upper_files = files[level];
    const int32_t upper_size = static_cast<int32_t>(upper_files.size());
    const auto& lower_files = files[level + 1];
    level_rb_[level] = static_cast<int32_t>(upper_files.size()) - 1;
    if (upper_size == 0) {
      continue;
    }
    IndexLevel& index_level = next_level_index_[level];
    index_level.num_index = upper_size;
    mem = arena->AllocateAligned(upper_size * sizeof(IndexUnit));
    index_level.index_units = new (mem) IndexUnit[upper_size];

    CalculateLB(
        upper_files, lower_files, &index_level,
        [this](const FileMetaData * a, const FileMetaData * b)->int {
          return ucmp_->Compare(a->smallest.user_key(), b->largest.user_key());
        },
        [](IndexUnit* index, int32_t f_idx) { index->smallest_lb = f_idx; });
    CalculateLB(
        upper_files, lower_files, &index_level,
        [this](const FileMetaData * a, const FileMetaData * b)->int {
          return ucmp_->Compare(a->largest.user_key(), b->largest.user_key());
        },
        [](IndexUnit* index, int32_t f_idx) { index->largest_lb = f_idx; });
    CalculateRB(
        upper_files, lower_files, &index_level,
        [this](const FileMetaData * a, const FileMetaData * b)->int {
          return ucmp_->Compare(a->smallest.user_key(), b->smallest.user_key());
        },
        [](IndexUnit* index, int32_t f_idx) { index->smallest_rb = f_idx; });
    CalculateRB(
        upper_files, lower_files, &index_level,
        [this](const FileMetaData * a, const FileMetaData * b)->int {
          return ucmp_->Compare(a->largest.user_key(), b->smallest.user_key());
        },
        [](IndexUnit* index, int32_t f_idx) { index->largest_rb = f_idx; });
  }

  level_rb_[num_levels_ - 1] =
      static_cast<int32_t>(files[num_levels_ - 1].size()) - 1;
}

void FileIndexer::CalculateLB(
    const std::vector<FileMetaData*>& upper_files,
    const std::vector<FileMetaData*>& lower_files, IndexLevel* index_level,
    std::function<int(const FileMetaData*, const FileMetaData*)> cmp_op,
    std::function<void(IndexUnit*, int32_t)> set_index) {
  const int32_t upper_size = static_cast<int32_t>(upper_files.size());
  const int32_t lower_size = static_cast<int32_t>(lower_files.size());
  int32_t upper_idx = 0;
  int32_t lower_idx = 0;

  IndexUnit* index = index_level->index_units;
  while (upper_idx < upper_size && lower_idx < lower_size) {
    int cmp = cmp_op(upper_files[upper_idx], lower_files[lower_idx]);

    if (cmp == 0) {
      set_index(&index[upper_idx], lower_idx);
      ++upper_idx;
      ++lower_idx;
    } else if (cmp > 0) {
      // Lower level's file (largest) is smaller, a key won't hit in that
      // file. Move to next lower file
      ++lower_idx;
    } else {
      // Lower level's file becomes larger, update the index, and
      // move to the next upper file
      set_index(&index[upper_idx], lower_idx);
      ++upper_idx;
    }
  }

  while (upper_idx < upper_size) {
    // Lower files are exhausted, that means the remaining upper files are
    // greater than any lower files. Set the index to be the lower level size.
    set_index(&index[upper_idx], lower_size);
    ++upper_idx;
  }
}

void FileIndexer::CalculateRB(
    const std::vector<FileMetaData*>& upper_files,
    const std::vector<FileMetaData*>& lower_files, IndexLevel* index_level,
    std::function<int(const FileMetaData*, const FileMetaData*)> cmp_op,
    std::function<void(IndexUnit*, int32_t)> set_index) {
  const int32_t upper_size = static_cast<int32_t>(upper_files.size());
  const int32_t lower_size = static_cast<int32_t>(lower_files.size());
  int32_t upper_idx = upper_size - 1;
  int32_t lower_idx = lower_size - 1;

  IndexUnit* index = index_level->index_units;
  while (upper_idx >= 0 && lower_idx >= 0) {
    int cmp = cmp_op(upper_files[upper_idx], lower_files[lower_idx]);

    if (cmp == 0) {
      set_index(&index[upper_idx], lower_idx);
      --upper_idx;
      --lower_idx;
    } else if (cmp < 0) {
      // Lower level's file (smallest) is larger, a key won't hit in that
      // file. Move to next lower file.
      --lower_idx;
    } else {
      // Lower level's file becomes smaller, update the index, and move to
      // the next the upper file
      set_index(&index[upper_idx], lower_idx);
      --upper_idx;
    }
  }
  while (upper_idx >= 0) {
    // Lower files are exhausted, that means the remaining upper files are
    // smaller than any lower files. Set it to -1.
    set_index(&index[upper_idx], -1);
    --upper_idx;
  }
}

}  // namespace rocksdb
