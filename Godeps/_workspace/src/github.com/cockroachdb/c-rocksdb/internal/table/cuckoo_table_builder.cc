//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE
#include "table/cuckoo_table_builder.h"

#include <assert.h>
#include <algorithm>
#include <limits>
#include <string>
#include <vector>

#include "db/dbformat.h"
#include "rocksdb/env.h"
#include "rocksdb/table.h"
#include "table/block_builder.h"
#include "table/cuckoo_table_factory.h"
#include "table/format.h"
#include "table/meta_blocks.h"
#include "util/autovector.h"
#include "util/file_reader_writer.h"
#include "util/random.h"
#include "util/string_util.h"

namespace rocksdb {
const std::string CuckooTablePropertyNames::kEmptyKey =
      "rocksdb.cuckoo.bucket.empty.key";
const std::string CuckooTablePropertyNames::kNumHashFunc =
      "rocksdb.cuckoo.hash.num";
const std::string CuckooTablePropertyNames::kHashTableSize =
      "rocksdb.cuckoo.hash.size";
const std::string CuckooTablePropertyNames::kValueLength =
      "rocksdb.cuckoo.value.length";
const std::string CuckooTablePropertyNames::kIsLastLevel =
      "rocksdb.cuckoo.file.islastlevel";
const std::string CuckooTablePropertyNames::kCuckooBlockSize =
      "rocksdb.cuckoo.hash.cuckooblocksize";
const std::string CuckooTablePropertyNames::kIdentityAsFirstHash =
      "rocksdb.cuckoo.hash.identityfirst";
const std::string CuckooTablePropertyNames::kUseModuleHash =
      "rocksdb.cuckoo.hash.usemodule";
const std::string CuckooTablePropertyNames::kUserKeyLength =
      "rocksdb.cuckoo.hash.userkeylength";

// Obtained by running echo rocksdb.table.cuckoo | sha1sum
extern const uint64_t kCuckooTableMagicNumber = 0x926789d0c5f17873ull;

CuckooTableBuilder::CuckooTableBuilder(
    WritableFileWriter* file, double max_hash_table_ratio,
    uint32_t max_num_hash_table, uint32_t max_search_depth,
    const Comparator* user_comparator, uint32_t cuckoo_block_size,
    bool use_module_hash, bool identity_as_first_hash,
    uint64_t (*get_slice_hash)(const Slice&, uint32_t, uint64_t))
    : num_hash_func_(2),
      file_(file),
      max_hash_table_ratio_(max_hash_table_ratio),
      max_num_hash_func_(max_num_hash_table),
      max_search_depth_(max_search_depth),
      cuckoo_block_size_(std::max(1U, cuckoo_block_size)),
      hash_table_size_(use_module_hash ? 0 : 2),
      is_last_level_file_(false),
      has_seen_first_key_(false),
      has_seen_first_value_(false),
      key_size_(0),
      value_size_(0),
      num_entries_(0),
      num_values_(0),
      ucomp_(user_comparator),
      use_module_hash_(use_module_hash),
      identity_as_first_hash_(identity_as_first_hash),
      get_slice_hash_(get_slice_hash),
      closed_(false) {
  // Data is in a huge block.
  properties_.num_data_blocks = 1;
  properties_.index_size = 0;
  properties_.filter_size = 0;
}

void CuckooTableBuilder::Add(const Slice& key, const Slice& value) {
  if (num_entries_ >= kMaxVectorIdx - 1) {
    status_ = Status::NotSupported("Number of keys in a file must be < 2^32-1");
    return;
  }
  ParsedInternalKey ikey;
  if (!ParseInternalKey(key, &ikey)) {
    status_ = Status::Corruption("Unable to parse key into inernal key.");
    return;
  }
  if (ikey.type != kTypeDeletion && ikey.type != kTypeValue) {
    status_ = Status::NotSupported("Unsupported key type " +
                                   ToString(ikey.type));
    return;
  }

  // Determine if we can ignore the sequence number and value type from
  // internal keys by looking at sequence number from first key. We assume
  // that if first key has a zero sequence number, then all the remaining
  // keys will have zero seq. no.
  if (!has_seen_first_key_) {
    is_last_level_file_ = ikey.sequence == 0;
    has_seen_first_key_ = true;
    smallest_user_key_.assign(ikey.user_key.data(), ikey.user_key.size());
    largest_user_key_.assign(ikey.user_key.data(), ikey.user_key.size());
    key_size_ = is_last_level_file_ ? ikey.user_key.size() : key.size();
  }
  if (key_size_ != (is_last_level_file_ ? ikey.user_key.size() : key.size())) {
    status_ = Status::NotSupported("all keys have to be the same size");
    return;
  }
  // Even if one sequence number is non-zero, then it is not last level.
  assert(!is_last_level_file_ || ikey.sequence == 0);

  if (ikey.type == kTypeValue) {
    if (!has_seen_first_value_) {
      has_seen_first_value_ = true;
      value_size_ = value.size();
    }
    if (value_size_ != value.size()) {
      status_ = Status::NotSupported("all values have to be the same size");
      return;
    }

    if (is_last_level_file_) {
      kvs_.append(ikey.user_key.data(), ikey.user_key.size());
    } else {
      kvs_.append(key.data(), key.size());
    }
    kvs_.append(value.data(), value.size());
    ++num_values_;
  } else {
    if (is_last_level_file_) {
      deleted_keys_.append(ikey.user_key.data(), ikey.user_key.size());
    } else {
      deleted_keys_.append(key.data(), key.size());
    }
  }
  ++num_entries_;

  // In order to fill the empty buckets in the hash table, we identify a
  // key which is not used so far (unused_user_key). We determine this by
  // maintaining smallest and largest keys inserted so far in bytewise order
  // and use them to find a key outside this range in Finish() operation.
  // Note that this strategy is independent of user comparator used here.
  if (ikey.user_key.compare(smallest_user_key_) < 0) {
    smallest_user_key_.assign(ikey.user_key.data(), ikey.user_key.size());
  } else if (ikey.user_key.compare(largest_user_key_) > 0) {
    largest_user_key_.assign(ikey.user_key.data(), ikey.user_key.size());
  }
  if (!use_module_hash_) {
    if (hash_table_size_ < num_entries_ / max_hash_table_ratio_) {
      hash_table_size_ *= 2;
    }
  }
}

bool CuckooTableBuilder::IsDeletedKey(uint64_t idx) const {
  assert(closed_);
  return idx >= num_values_;
}

Slice CuckooTableBuilder::GetKey(uint64_t idx) const {
  assert(closed_);
  if (IsDeletedKey(idx)) {
    return Slice(&deleted_keys_[(idx - num_values_) * key_size_], key_size_);
  }
  return Slice(&kvs_[idx * (key_size_ + value_size_)], key_size_);
}

Slice CuckooTableBuilder::GetUserKey(uint64_t idx) const {
  assert(closed_);
  return is_last_level_file_ ? GetKey(idx) : ExtractUserKey(GetKey(idx));
}

Slice CuckooTableBuilder::GetValue(uint64_t idx) const {
  assert(closed_);
  if (IsDeletedKey(idx)) {
    static std::string empty_value(value_size_, 'a');
    return Slice(empty_value);
  }
  return Slice(&kvs_[idx * (key_size_ + value_size_) + key_size_], value_size_);
}

Status CuckooTableBuilder::MakeHashTable(std::vector<CuckooBucket>* buckets) {
  buckets->resize(hash_table_size_ + cuckoo_block_size_ - 1);
  uint32_t make_space_for_key_call_id = 0;
  for (uint32_t vector_idx = 0; vector_idx < num_entries_; vector_idx++) {
    uint64_t bucket_id;
    bool bucket_found = false;
    autovector<uint64_t> hash_vals;
    Slice user_key = GetUserKey(vector_idx);
    for (uint32_t hash_cnt = 0; hash_cnt < num_hash_func_ && !bucket_found;
        ++hash_cnt) {
      uint64_t hash_val = CuckooHash(user_key, hash_cnt, use_module_hash_,
          hash_table_size_, identity_as_first_hash_, get_slice_hash_);
      // If there is a collision, check next cuckoo_block_size_ locations for
      // empty locations. While checking, if we reach end of the hash table,
      // stop searching and proceed for next hash function.
      for (uint32_t block_idx = 0; block_idx < cuckoo_block_size_;
          ++block_idx, ++hash_val) {
        if ((*buckets)[hash_val].vector_idx == kMaxVectorIdx) {
          bucket_id = hash_val;
          bucket_found = true;
          break;
        } else {
          if (ucomp_->Compare(user_key,
                GetUserKey((*buckets)[hash_val].vector_idx)) == 0) {
            return Status::NotSupported("Same key is being inserted again.");
          }
          hash_vals.push_back(hash_val);
        }
      }
    }
    while (!bucket_found && !MakeSpaceForKey(hash_vals,
          ++make_space_for_key_call_id, buckets, &bucket_id)) {
      // Rehash by increashing number of hash tables.
      if (num_hash_func_ >= max_num_hash_func_) {
        return Status::NotSupported("Too many collisions. Unable to hash.");
      }
      // We don't really need to rehash the entire table because old hashes are
      // still valid and we only increased the number of hash functions.
      uint64_t hash_val = CuckooHash(user_key, num_hash_func_, use_module_hash_,
          hash_table_size_, identity_as_first_hash_, get_slice_hash_);
      ++num_hash_func_;
      for (uint32_t block_idx = 0; block_idx < cuckoo_block_size_;
          ++block_idx, ++hash_val) {
        if ((*buckets)[hash_val].vector_idx == kMaxVectorIdx) {
          bucket_found = true;
          bucket_id = hash_val;
          break;
        } else {
          hash_vals.push_back(hash_val);
        }
      }
    }
    (*buckets)[bucket_id].vector_idx = vector_idx;
  }
  return Status::OK();
}

Status CuckooTableBuilder::Finish() {
  assert(!closed_);
  closed_ = true;
  std::vector<CuckooBucket> buckets;
  Status s;
  std::string unused_bucket;
  if (num_entries_ > 0) {
    // Calculate the real hash size if module hash is enabled.
    if (use_module_hash_) {
      hash_table_size_ = num_entries_ / max_hash_table_ratio_;
    }
    s = MakeHashTable(&buckets);
    if (!s.ok()) {
      return s;
    }
    // Determine unused_user_key to fill empty buckets.
    std::string unused_user_key = smallest_user_key_;
    int curr_pos = static_cast<int>(unused_user_key.size()) - 1;
    while (curr_pos >= 0) {
      --unused_user_key[curr_pos];
      if (Slice(unused_user_key).compare(smallest_user_key_) < 0) {
        break;
      }
      --curr_pos;
    }
    if (curr_pos < 0) {
      // Try using the largest key to identify an unused key.
      unused_user_key = largest_user_key_;
      curr_pos = static_cast<int>(unused_user_key.size()) - 1;
      while (curr_pos >= 0) {
        ++unused_user_key[curr_pos];
        if (Slice(unused_user_key).compare(largest_user_key_) > 0) {
          break;
        }
        --curr_pos;
      }
    }
    if (curr_pos < 0) {
      return Status::Corruption("Unable to find unused key");
    }
    if (is_last_level_file_) {
      unused_bucket = unused_user_key;
    } else {
      ParsedInternalKey ikey(unused_user_key, 0, kTypeValue);
      AppendInternalKey(&unused_bucket, ikey);
    }
  }
  properties_.num_entries = num_entries_;
  properties_.fixed_key_len = key_size_;
  properties_.user_collected_properties[
        CuckooTablePropertyNames::kValueLength].assign(
        reinterpret_cast<const char*>(&value_size_), sizeof(value_size_));

  uint64_t bucket_size = key_size_ + value_size_;
  unused_bucket.resize(bucket_size, 'a');
  // Write the table.
  uint32_t num_added = 0;
  for (auto& bucket : buckets) {
    if (bucket.vector_idx == kMaxVectorIdx) {
      s = file_->Append(Slice(unused_bucket));
    } else {
      ++num_added;
      s = file_->Append(GetKey(bucket.vector_idx));
      if (s.ok()) {
        if (value_size_ > 0) {
          s = file_->Append(GetValue(bucket.vector_idx));
        }
      }
    }
    if (!s.ok()) {
      return s;
    }
  }
  assert(num_added == NumEntries());
  properties_.raw_key_size = num_added * properties_.fixed_key_len;
  properties_.raw_value_size = num_added * value_size_;

  uint64_t offset = buckets.size() * bucket_size;
  properties_.data_size = offset;
  unused_bucket.resize(properties_.fixed_key_len);
  properties_.user_collected_properties[
    CuckooTablePropertyNames::kEmptyKey] = unused_bucket;
  properties_.user_collected_properties[
    CuckooTablePropertyNames::kNumHashFunc].assign(
        reinterpret_cast<char*>(&num_hash_func_), sizeof(num_hash_func_));

  properties_.user_collected_properties[
    CuckooTablePropertyNames::kHashTableSize].assign(
        reinterpret_cast<const char*>(&hash_table_size_),
        sizeof(hash_table_size_));
  properties_.user_collected_properties[
    CuckooTablePropertyNames::kIsLastLevel].assign(
        reinterpret_cast<const char*>(&is_last_level_file_),
        sizeof(is_last_level_file_));
  properties_.user_collected_properties[
    CuckooTablePropertyNames::kCuckooBlockSize].assign(
        reinterpret_cast<const char*>(&cuckoo_block_size_),
        sizeof(cuckoo_block_size_));
  properties_.user_collected_properties[
    CuckooTablePropertyNames::kIdentityAsFirstHash].assign(
        reinterpret_cast<const char*>(&identity_as_first_hash_),
        sizeof(identity_as_first_hash_));
  properties_.user_collected_properties[
    CuckooTablePropertyNames::kUseModuleHash].assign(
        reinterpret_cast<const char*>(&use_module_hash_),
        sizeof(use_module_hash_));
  uint32_t user_key_len = static_cast<uint32_t>(smallest_user_key_.size());
  properties_.user_collected_properties[
    CuckooTablePropertyNames::kUserKeyLength].assign(
        reinterpret_cast<const char*>(&user_key_len),
        sizeof(user_key_len));

  // Write meta blocks.
  MetaIndexBuilder meta_index_builder;
  PropertyBlockBuilder property_block_builder;

  property_block_builder.AddTableProperty(properties_);
  property_block_builder.Add(properties_.user_collected_properties);
  Slice property_block = property_block_builder.Finish();
  BlockHandle property_block_handle;
  property_block_handle.set_offset(offset);
  property_block_handle.set_size(property_block.size());
  s = file_->Append(property_block);
  offset += property_block.size();
  if (!s.ok()) {
    return s;
  }

  meta_index_builder.Add(kPropertiesBlock, property_block_handle);
  Slice meta_index_block = meta_index_builder.Finish();

  BlockHandle meta_index_block_handle;
  meta_index_block_handle.set_offset(offset);
  meta_index_block_handle.set_size(meta_index_block.size());
  s = file_->Append(meta_index_block);
  if (!s.ok()) {
    return s;
  }

  Footer footer(kCuckooTableMagicNumber, 1);
  footer.set_metaindex_handle(meta_index_block_handle);
  footer.set_index_handle(BlockHandle::NullBlockHandle());
  std::string footer_encoding;
  footer.EncodeTo(&footer_encoding);
  s = file_->Append(footer_encoding);
  return s;
}

void CuckooTableBuilder::Abandon() {
  assert(!closed_);
  closed_ = true;
}

uint64_t CuckooTableBuilder::NumEntries() const {
  return num_entries_;
}

uint64_t CuckooTableBuilder::FileSize() const {
  if (closed_) {
    return file_->GetFileSize();
  } else if (num_entries_ == 0) {
    return 0;
  }

  if (use_module_hash_) {
    return (key_size_ + value_size_) * num_entries_ / max_hash_table_ratio_;
  } else {
    // Account for buckets being a power of two.
    // As elements are added, file size remains constant for a while and
    // doubles its size. Since compaction algorithm stops adding elements
    // only after it exceeds the file limit, we account for the extra element
    // being added here.
    uint64_t expected_hash_table_size = hash_table_size_;
    if (expected_hash_table_size < (num_entries_ + 1) / max_hash_table_ratio_) {
      expected_hash_table_size *= 2;
    }
    return (key_size_ + value_size_) * expected_hash_table_size - 1;
  }
}

// This method is invoked when there is no place to insert the target key.
// It searches for a set of elements that can be moved to accommodate target
// key. The search is a BFS graph traversal with first level (hash_vals)
// being all the buckets target key could go to.
// Then, from each node (curr_node), we find all the buckets that curr_node
// could go to. They form the children of curr_node in the tree.
// We continue the traversal until we find an empty bucket, in which case, we
// move all elements along the path from first level to this empty bucket, to
// make space for target key which is inserted at first level (*bucket_id).
// If tree depth exceedes max depth, we return false indicating failure.
bool CuckooTableBuilder::MakeSpaceForKey(
    const autovector<uint64_t>& hash_vals,
    const uint32_t make_space_for_key_call_id,
    std::vector<CuckooBucket>* buckets, uint64_t* bucket_id) {
  struct CuckooNode {
    uint64_t bucket_id;
    uint32_t depth;
    uint32_t parent_pos;
    CuckooNode(uint64_t _bucket_id, uint32_t _depth, int _parent_pos)
        : bucket_id(_bucket_id), depth(_depth), parent_pos(_parent_pos) {}
  };
  // This is BFS search tree that is stored simply as a vector.
  // Each node stores the index of parent node in the vector.
  std::vector<CuckooNode> tree;
  // We want to identify already visited buckets in the current method call so
  // that we don't add same buckets again for exploration in the tree.
  // We do this by maintaining a count of current method call in
  // make_space_for_key_call_id, which acts as a unique id for this invocation
  // of the method. We store this number into the nodes that we explore in
  // current method call.
  // It is unlikely for the increment operation to overflow because the maximum
  // no. of times this will be called is <= max_num_hash_func_ + num_entries_.
  for (uint32_t hash_cnt = 0; hash_cnt < num_hash_func_; ++hash_cnt) {
    uint64_t bid = hash_vals[hash_cnt];
    (*buckets)[bid].make_space_for_key_call_id = make_space_for_key_call_id;
    tree.push_back(CuckooNode(bid, 0, 0));
  }
  bool null_found = false;
  uint32_t curr_pos = 0;
  while (!null_found && curr_pos < tree.size()) {
    CuckooNode& curr_node = tree[curr_pos];
    uint32_t curr_depth = curr_node.depth;
    if (curr_depth >= max_search_depth_) {
      break;
    }
    CuckooBucket& curr_bucket = (*buckets)[curr_node.bucket_id];
    for (uint32_t hash_cnt = 0;
        hash_cnt < num_hash_func_ && !null_found; ++hash_cnt) {
      uint64_t child_bucket_id = CuckooHash(GetUserKey(curr_bucket.vector_idx),
          hash_cnt, use_module_hash_, hash_table_size_, identity_as_first_hash_,
          get_slice_hash_);
      // Iterate inside Cuckoo Block.
      for (uint32_t block_idx = 0; block_idx < cuckoo_block_size_;
          ++block_idx, ++child_bucket_id) {
        if ((*buckets)[child_bucket_id].make_space_for_key_call_id ==
            make_space_for_key_call_id) {
          continue;
        }
        (*buckets)[child_bucket_id].make_space_for_key_call_id =
          make_space_for_key_call_id;
        tree.push_back(CuckooNode(child_bucket_id, curr_depth + 1,
              curr_pos));
        if ((*buckets)[child_bucket_id].vector_idx == kMaxVectorIdx) {
          null_found = true;
          break;
        }
      }
    }
    ++curr_pos;
  }

  if (null_found) {
    // There is an empty node in tree.back(). Now, traverse the path from this
    // empty node to top of the tree and at every node in the path, replace
    // child with the parent. Stop when first level is reached in the tree
    // (happens when 0 <= bucket_to_replace_pos < num_hash_func_) and return
    // this location in first level for target key to be inserted.
    uint32_t bucket_to_replace_pos = static_cast<uint32_t>(tree.size()) - 1;
    while (bucket_to_replace_pos >= num_hash_func_) {
      CuckooNode& curr_node = tree[bucket_to_replace_pos];
      (*buckets)[curr_node.bucket_id] =
        (*buckets)[tree[curr_node.parent_pos].bucket_id];
      bucket_to_replace_pos = curr_node.parent_pos;
    }
    *bucket_id = tree[bucket_to_replace_pos].bucket_id;
  }
  return null_found;
}

}  // namespace rocksdb
#endif  // ROCKSDB_LITE
