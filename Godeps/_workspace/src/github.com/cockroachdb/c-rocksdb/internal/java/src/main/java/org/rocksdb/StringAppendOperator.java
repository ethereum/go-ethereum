// Copyright (c) 2014, Vlad Balan (vlad.gm@gmail.com).  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * StringAppendOperator is a merge operator that concatenates
 * two strings.
 */
public class StringAppendOperator implements MergeOperator {
    @Override public long newMergeOperatorHandle() {
        return newMergeOperatorHandleImpl();
    }
    private native long newMergeOperatorHandleImpl();
}
