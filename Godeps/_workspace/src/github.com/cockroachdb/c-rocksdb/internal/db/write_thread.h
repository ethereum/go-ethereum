//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#include <assert.h>
#include <stdint.h>
#include <atomic>
#include <condition_variable>
#include <mutex>
#include <type_traits>
#include "rocksdb/status.h"
#include "db/write_batch_internal.h"
#include "util/autovector.h"
#include "util/instrumented_mutex.h"

namespace rocksdb {

class WriteThread {
 public:
  // Information kept for every waiting writer.
  struct Writer {
    WriteBatch* batch;
    bool sync;
    bool disableWAL;
    bool in_batch_group;
    bool done;
    bool has_callback;
    Status status;
    bool made_waitable;  // records lazy construction of mutex and cv
    bool joined;         // read/write only under JoinMutex() (or pre-link)
    std::aligned_storage<sizeof(std::mutex)>::type join_mutex_bytes;
    std::aligned_storage<sizeof(std::condition_variable)>::type join_cv_bytes;
    Writer* link_older;  // read/write only before linking, or as leader
    Writer* link_newer;  // lazy, read/write only before linking, or as leader

    Writer()
        : batch(nullptr),
          sync(false),
          disableWAL(false),
          in_batch_group(false),
          done(false),
          has_callback(false),
          made_waitable(false),
          joined(false),
          link_older(nullptr),
          link_newer(nullptr) {}

    ~Writer() {
      if (made_waitable) {
        JoinMutex().~mutex();
        JoinCV().~condition_variable();
      }
    }

    void CreateMutex() {
      assert(!joined);
      if (!made_waitable) {
        made_waitable = true;
        new (&join_mutex_bytes) std::mutex;
        new (&join_cv_bytes) std::condition_variable;
      }
    }

    // No other mutexes may be acquired while holding JoinMutex(), it is
    // always last in the order
    std::mutex& JoinMutex() {
      assert(made_waitable);
      return *static_cast<std::mutex*>(static_cast<void*>(&join_mutex_bytes));
    }

    std::condition_variable& JoinCV() {
      assert(made_waitable);
      return *static_cast<std::condition_variable*>(
          static_cast<void*>(&join_cv_bytes));
    }
  };

  WriteThread() : newest_writer_(nullptr) {}

  // IMPORTANT: None of the methods in this class rely on the db mutex
  // for correctness. All of the methods except JoinBatchGroup and
  // EnterUnbatched may be called either with or without the db mutex held.
  // Correctness is maintained by ensuring that only a single thread is
  // a leader at a time.

  // Registers w as ready to become part of a batch group, and blocks
  // until some other thread has completed the write (in which case
  // w->done will be set to true) or this write has become the leader
  // of a batch group (w->done will remain unset).  The db mutex SHOULD
  // NOT be held when calling this function, because it will block.
  // If !w->done then JoinBatchGroup should be followed by a call to
  // EnterAsBatchGroupLeader and ExitAsBatchGroupLeader.
  //
  // Writer* w:        Writer to be executed as part of a batch group
  void JoinBatchGroup(Writer* w);

  // Constructs a write batch group led by leader, which should be a
  // Writer passed to JoinBatchGroup on the current thread.
  //
  // Writer* leader:         Writer passed to JoinBatchGroup, but !done
  // Writer** last_writer:   Out-param for use by ExitAsBatchGroupLeader
  // autovector<WriteBatch*>* write_batch_group: Out-param of group members
  // returns:                Total batch group size
  size_t EnterAsBatchGroupLeader(Writer* leader, Writer** last_writer,
                                 autovector<WriteBatch*>* write_batch_group);

  // Unlinks the Writer-s in a batch group, wakes up the non-leaders, and
  // wakes up the next leader (if any).
  //
  // Writer* leader:         From EnterAsBatchGroupLeader
  // Writer* last_writer:    Value of out-param of EnterAsBatchGroupLeader
  // Status status:          Status of write operation
  void ExitAsBatchGroupLeader(Writer* leader, Writer* last_writer,
                              Status status);

  // Waits for all preceding writers (unlocking mu while waiting), then
  // registers w as the currently proceeding writer.
  //
  // Writer* w:              A Writer not eligible for batching
  // InstrumentedMutex* mu:  The db mutex, to unlock while waiting
  // REQUIRES: db mutex held
  void EnterUnbatched(Writer* w, InstrumentedMutex* mu);

  // Completes a Writer begun with EnterUnbatched, unblocking subsequent
  // writers.
  void ExitUnbatched(Writer* w);

 private:
  // Points to the newest pending Writer.  Only leader can remove
  // elements, adding can be done lock-free by anybody
  std::atomic<Writer*> newest_writer_;

  void Await(Writer* w);
  void MarkJoined(Writer* w);

  // Links w into the newest_writer_ list. Sets *wait_needed to false
  // if w was linked directly into the leader position, true otherwise.
  // Safe to call from multiple threads without external locking.
  void LinkOne(Writer* w, bool* wait_needed);

  // Computes any missing link_newer links.  Should not be called
  // concurrently with itself.
  void CreateMissingNewerLinks(Writer* head);
};

}  // namespace rocksdb
