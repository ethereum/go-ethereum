<?php
// Copyright 2004-present Facebook. All Rights Reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

final class FbcodeClangFormatLinter extends BaseDirectoryScopedFormatLinter {

  const LINT_FORMATTING = 1;
  const CLANG_FORMAT_BINARY = '/mnt/vol/engshare/admin/scripts/clang-format';

  protected function getPathsToLint() {
    return array('');
  }

  public function getLinterName() {
    return 'CLANG_FORMAT';
  }

  public function getLintSeverityMap() {
    return array(
      self::LINT_FORMATTING => ArcanistLintSeverity::SEVERITY_ADVICE,
    );
  }

  public function getLintNameMap() {
    return array(
      self::LINT_FORMATTING => pht('Changes are not clang-formatted'),
    );
  }

  protected function getFormatFuture($path, array $changed) {
    $args = "";
    foreach ($changed as $key => $value) {
      $args .= " --lines=$key:$key";
    }

    $binary = self::CLANG_FORMAT_BINARY;
    if (!file_exists($binary)) {
      // trust the $PATH
      $binary = "clang-format";
    }

    return new ExecFuture(
      "%s %s $args",
      $binary,
      $this->getEngine()->getFilePathOnDisk($path));
  }

  protected function getLintMessage($diff) {
    $link_to_clang_format =
      "[[ http://fburl.com/clang-format | clang-format ]]";
    return <<<LINT_MSG
Changes in this file were not formatted using $link_to_clang_format.
Please run build_tools/format-diff.sh or `make format`
LINT_MSG;
  }
}
