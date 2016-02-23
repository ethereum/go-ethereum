<?php
// Copyright 2004-present Facebook. All Rights Reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

class FacebookArcanistConfiguration extends ArcanistConfiguration {

  public function didRunWorkflow($command,
                                 ArcanistBaseWorkflow $workflow,
                                 $error_code) {
    if ($command == 'diff' && !$workflow->isRawDiffSource()) {
      $this->maybePushToJenkins($workflow);
    }
  }

  //////////////////////////////////////////////////////////////////////
  /* Send off builds to jenkins */
  function maybePushToJenkins($workflow) {
    $diffID = $workflow->getDiffID();
    if ($diffID === null) {
      return;
    }

    $results = $workflow->getTestResults();
    if (!$results) {
      return;
    }

    $url = "https://ci-builds.fb.com/view/rocksdb/job/rocksdb_diff_check/"
               ."buildWithParameters?token=AUTH&DIFF_ID=$diffID";
    system("curl --noproxy '*' \"$url\" > /dev/null 2>&1");
  }

}
