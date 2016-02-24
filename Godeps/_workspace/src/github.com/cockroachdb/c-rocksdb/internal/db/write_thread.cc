//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include "db/write_thread.h"

namespace rocksdb {

void WriteThread::Await(Writer* w) {
  std::unique_lock<std::mutex> guard(w->JoinMutex());
  w->JoinCV().wait(guard, [w] { return w->joined; });
}

void WriteThread::MarkJoined(Writer* w) {
  std::lock_guard<std::mutex> guard(w->JoinMutex());
  assert(!w->joined);
  w->joined = true;
  w->JoinCV().notify_one();
}

void WriteThread::LinkOne(Writer* w, bool* wait_needed) {
  assert(!w->joined && !w->done);

  Writer* writers = newest_writer_.load(std::memory_order_relaxed);
  while (true) {
    w->link_older = writers;
    if (writers != nullptr) {
      w->CreateMutex();
    }
    if (newest_writer_.compare_exchange_strong(writers, w)) {
      // Success.
      *wait_needed = (writers != nullptr);
      return;
    }
  }
}

void WriteThread::CreateMissingNewerLinks(Writer* head) {
  while (true) {
    Writer* next = head->link_older;
    if (next == nullptr || next->link_newer != nullptr) {
      assert(next == nullptr || next->link_newer == head);
      break;
    }
    next->link_newer = head;
    head = next;
  }
}

void WriteThread::JoinBatchGroup(Writer* w) {
  assert(w->batch != nullptr);
  bool wait_needed;
  LinkOne(w, &wait_needed);
  if (wait_needed) {
    Await(w);
  }
}

size_t WriteThread::EnterAsBatchGroupLeader(
    Writer* leader, WriteThread::Writer** last_writer,
    autovector<WriteBatch*>* write_batch_group) {
  assert(leader->link_older == nullptr);
  assert(leader->batch != nullptr);

  size_t size = WriteBatchInternal::ByteSize(leader->batch);
  write_batch_group->push_back(leader->batch);

  // Allow the group to grow up to a maximum size, but if the
  // original write is small, limit the growth so we do not slow
  // down the small write too much.
  size_t max_size = 1 << 20;
  if (size <= (128 << 10)) {
    max_size = size + (128 << 10);
  }

  *last_writer = leader;

  if (leader->has_callback) {
    // TODO(agiardullo:) Batching not currently supported as this write may
    // fail if the callback function decides to abort this write.
    return size;
  }

  Writer* newest_writer = newest_writer_.load(std::memory_order_acquire);

  // This is safe regardless of any db mutex status of the caller. Previous
  // calls to ExitAsGroupLeader either didn't call CreateMissingNewerLinks
  // (they emptied the list and then we added ourself as leader) or had to
  // explicitly wake up us (the list was non-empty when we added ourself,
  // so we have already received our MarkJoined).
  CreateMissingNewerLinks(newest_writer);

  // Tricky. Iteration start (leader) is exclusive and finish
  // (newest_writer) is inclusive. Iteration goes from old to new.
  Writer* w = leader;
  while (w != newest_writer) {
    w = w->link_newer;

    if (w->sync && !leader->sync) {
      // Do not include a sync write into a batch handled by a non-sync write.
      break;
    }

    if (!w->disableWAL && leader->disableWAL) {
      // Do not include a write that needs WAL into a batch that has
      // WAL disabled.
      break;
    }

    if (w->has_callback) {
      // Do not include writes which may be aborted if the callback does not
      // succeed.
      break;
    }

    if (w->batch == nullptr) {
      // Do not include those writes with nullptr batch. Those are not writes,
      // those are something else. They want to be alone
      break;
    }

    size += WriteBatchInternal::ByteSize(w->batch);
    if (size > max_size) {
      // Do not make batch too big
      break;
    }

    write_batch_group->push_back(w->batch);
    w->in_batch_group = true;
    *last_writer = w;
  }
  return size;
}

void WriteThread::ExitAsBatchGroupLeader(Writer* leader, Writer* last_writer,
                                         Status status) {
  assert(leader->link_older == nullptr);

  Writer* head = newest_writer_.load(std::memory_order_acquire);
  if (head != last_writer ||
      !newest_writer_.compare_exchange_strong(head, nullptr)) {
    // Either w wasn't the head during the load(), or it was the head
    // during the load() but somebody else pushed onto the list before
    // we did the compare_exchange_strong (causing it to fail).  In the
    // latter case compare_exchange_strong has the effect of re-reading
    // its first param (head).  No need to retry a failing CAS, because
    // only a departing leader (which we are at the moment) can remove
    // nodes from the list.
    assert(head != last_writer);

    // After walking link_older starting from head (if not already done)
    // we will be able to traverse w->link_newer below. This function
    // can only be called from an active leader, only a leader can
    // clear newest_writer_, we didn't, and only a clear newest_writer_
    // could cause the next leader to start their work without a call
    // to MarkJoined, so we can definitely conclude that no other leader
    // work is going on here (with or without db mutex).
    CreateMissingNewerLinks(head);
    assert(last_writer->link_newer->link_older == last_writer);
    last_writer->link_newer->link_older = nullptr;

    // Next leader didn't self-identify, because newest_writer_ wasn't
    // nullptr when they enqueued (we were definitely enqueued before them
    // and are still in the list).  That means leader handoff occurs when
    // we call MarkJoined
    MarkJoined(last_writer->link_newer);
  }
  // else nobody else was waiting, although there might already be a new
  // leader now

  while (last_writer != leader) {
    last_writer->status = status;
    last_writer->done = true;
    // We must read link_older before calling MarkJoined, because as
    // soon as it is marked the other thread's AwaitJoined may return
    // and deallocate the Writer.
    auto next = last_writer->link_older;
    MarkJoined(last_writer);
    last_writer = next;
  }
}

void WriteThread::EnterUnbatched(Writer* w, InstrumentedMutex* mu) {
  assert(w->batch == nullptr);
  bool wait_needed;
  LinkOne(w, &wait_needed);
  if (wait_needed) {
    mu->Unlock();
    Await(w);
    mu->Lock();
  }
}

void WriteThread::ExitUnbatched(Writer* w) {
  Status dummy_status;
  ExitAsBatchGroupLeader(w, w, dummy_status);
}

}  // namespace rocksdb
