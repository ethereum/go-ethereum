<?php
// Copyright 2004-present Facebook.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

class FacebookFbcodeLintEngine extends ArcanistLintEngine {

  public function buildLinters() {
    $linters = array();
    $paths = $this->getPaths();

    // Remove all deleted files, which are not checked by the
    // following linters.
    foreach ($paths as $key => $path) {
      if (!Filesystem::pathExists($this->getFilePathOnDisk($path))) {
        unset($paths[$key]);
      }
    }

    $generated_linter = new ArcanistGeneratedLinter();
    $linters[] = $generated_linter;

    $nolint_linter = new ArcanistNoLintLinter();
    $linters[] = $nolint_linter;

    $text_linter = new ArcanistTextLinter();
    $text_linter->setCustomSeverityMap(array(
      ArcanistTextLinter::LINT_LINE_WRAP
        => ArcanistLintSeverity::SEVERITY_ADVICE,
    ));
    $linters[] = $text_linter;

    $java_text_linter = new ArcanistTextLinter();
    $java_text_linter->setMaxLineLength(100);
    $java_text_linter->setCustomSeverityMap(array(
      ArcanistTextLinter::LINT_LINE_WRAP
        => ArcanistLintSeverity::SEVERITY_ADVICE,
    ));
    $linters[] = $java_text_linter;

    $python_linter = new ArcanistPEP8Linter();
    $linters[] = $python_linter;

    if (!$this->getCommitHookMode()) {
      $cpp_linters = array();
      $cpp_linters[] = $linters[] = new ArcanistCpplintLinter();
      $cpp_linters[] = $linters[] = new FbcodeCppLinter();

      $clang_format_linter = new FbcodeClangFormatLinter();
      $linters[] = $clang_format_linter;
    }

    $spelling_linter = new ArcanistSpellingLinter();
    $linters[] = $spelling_linter;

    foreach ($paths as $path) {
      $is_text = false;

      $text_extensions = (
        '/\.('.
        'cpp|cxx|c|cc|h|hpp|hxx|tcc|'.
        'py|rb|hs|pl|pm|tw|'.
        'php|phpt|css|js|'.
        'java|'.
        'thrift|'.
        'lua|'.
        'siv|'.
        'txt'.
        ')$/'
      );
      if (preg_match($text_extensions, $path)) {
        $is_text = true;
      }
      if ($is_text) {
        $nolint_linter->addPath($path);

        $generated_linter->addPath($path);
        $generated_linter->addData($path, $this->loadData($path));

        if (preg_match('/\.java$/', $path)) {
          $java_text_linter->addPath($path);
          $java_text_linter->addData($path, $this->loadData($path));
        } else {
          $text_linter->addPath($path);
          $text_linter->addData($path, $this->loadData($path));
        }

        $spelling_linter->addPath($path);
        $spelling_linter->addData($path, $this->loadData($path));
      }
      if (preg_match('/\.(cpp|c|cc|cxx|h|hh|hpp|hxx|tcc)$/', $path)
          && !preg_match('/third-party/', $path)) {
        foreach ($cpp_linters as &$linter) {
          $linter->addPath($path);
          $linter->addData($path, $this->loadData($path));
        }

        $clang_format_linter->addPath($path);
        $clang_format_linter->addData($path, $this->loadData($path));
        $clang_format_linter->setPathChangedLines(
          $path, $this->getPathChangedLines($path));
      }

      // Match *.py and contbuild config files
      if (preg_match('/(\.(py|tw|smcprops)|^contbuild\/configs\/[^\/]*)$/',
                    $path)) {
        $space_count = 4;
        $real_path = $this->getFilePathOnDisk($path);
        $dir = dirname($real_path);
        do {
          if (file_exists($dir.'/.python2space')) {
            $space_count = 2;
            break;
          }
          $dir = dirname($dir);
        } while ($dir != '/' && $dir != '.');

        $cur_path_linter = $python_linter;
        $cur_path_linter->addPath($path);
        $cur_path_linter->addData($path, $this->loadData($path));

        if (preg_match('/\.tw$/', $path)) {
          $cur_path_linter->setCustomSeverityMap(array(
            'E251' => ArcanistLintSeverity::SEVERITY_DISABLED,
          ));
        }
      }
    }

    $name_linter = new ArcanistFilenameLinter();
    $linters[] = $name_linter;
    foreach ($paths as $path) {
      $name_linter->addPath($path);
    }

    return $linters;
  }

}
