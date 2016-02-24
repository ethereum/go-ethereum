// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * Just a Java wrapper around EmptyValueCompactionFilter implemented in C++
 */
public class RemoveEmptyValueCompactionFilter extends AbstractCompactionFilter<Slice> {
  public RemoveEmptyValueCompactionFilter() {
    super();
    createNewRemoveEmptyValueCompactionFilter0();
  }

  private native void createNewRemoveEmptyValueCompactionFilter0();
}
