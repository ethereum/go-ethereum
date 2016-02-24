<?php
// Copyright 2015-present Facebook. All Rights Reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

final class FacebookHowtoevenLintEngine extends ArcanistLintEngine {

  public function buildLinters() {
    $paths = array();

    foreach ($this->getPaths() as $path) {
      // Don't try to lint deleted files or changed directories.
      if (!Filesystem::pathExists($path) || is_dir($path)) {
        continue;
      }

      if (preg_match('/\.(cpp|c|cc|cxx|h|hh|hpp|hxx|tcc)$/', $path)) {
        $paths[] = $path;
      }
    }

    $howtoeven = new FacebookHowtoevenLinter();
    $howtoeven->setPaths($paths);
    return array($howtoeven);
  }
}
