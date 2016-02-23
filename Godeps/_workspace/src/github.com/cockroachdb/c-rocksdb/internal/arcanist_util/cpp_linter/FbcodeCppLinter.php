<?php
// Copyright 2004-present Facebook.  All rights reserved.

class FbcodeCppLinter extends ArcanistLinter {
  const FLINT      = "/home/engshare/tools/flint";
  const LINT_ERROR   = 1;
  const LINT_WARNING = 2;
  const LINT_ADVICE  = 3;
  const C_FLAG = "--c_mode=true";

  private $rawLintOutput = array();

  public function willLintPaths(array $paths) {
    if (!file_exists(self::FLINT)) {
      return;
    }
    $futures = array();
    foreach ($paths as $p) {
      $lpath = $this->getEngine()->getFilePathOnDisk($p);
      $lpath_file = file($lpath);
      if (preg_match('/\.(c)$/', $lpath) ||
          preg_match('/-\*-.*Mode: C[; ].*-\*-/', $lpath_file[0]) ||
          preg_match('/vim(:.*)*:\s*(set\s+)?filetype=c\s*:/', $lpath_file[0])
          ) {
        $futures[$p] = new ExecFuture("%s %s %s 2>&1",
                           self::FLINT, self::C_FLAG,
                           $this->getEngine()->getFilePathOnDisk($p));
      } else {
        $futures[$p] = new ExecFuture("%s %s 2>&1",
          self::FLINT, $this->getEngine()->getFilePathOnDisk($p));
      }
    }

    foreach (Futures($futures)->limit(8) as $p => $f) {
      $this->rawLintOutput[$p] = $f->resolvex();
    }

    return;
  }

  public function getLinterName() {
    return "FBCPP";
  }

  public function lintPath($path) {
    $this->runCppLint($path);
  }

  private function runCppLint($path) {
    $msgs = $this->getCppLintOutput($path);
    foreach ($msgs as $m) {
      $this->raiseLintAtLine($m['line'], 0, $m['severity'], $m['msg']);
    }
  }

  private function adviseOnEachPattern(
    $path,
    $regex,
    $message,
    $lint_type = self::LINT_ADVICE,
    $match_idx = 0) {
      $file_data = $this->getData($path);
      $matches = array();
      if (!preg_match_all($regex, $file_data, $matches, PREG_OFFSET_CAPTURE)) {
        return;
      }

      foreach ($matches[$match_idx] as $match) {
        list($match_str, $offset) = $match;
        $this->raiseLintAtOffset($offset, $lint_type, $message, $match_str);
      }
  }

  public function getLintSeverityMap() {
    return array(
      self::LINT_WARNING => ArcanistLintSeverity::SEVERITY_WARNING,
      self::LINT_ADVICE  => ArcanistLintSeverity::SEVERITY_ADVICE,
      self::LINT_ERROR   => ArcanistLintSeverity::SEVERITY_ERROR
    );
  }

  public function getLintNameMap() {
    return array(
      self::LINT_ADVICE   => "CppLint Advice",
      self::LINT_WARNING  => "CppLint Warning",
      self::LINT_ERROR    => "CppLint Error"
    );
  }

  private function getCppLintOutput($path) {
    list($output) = $this->rawLintOutput[$path];

    $msgs = array();
    $current = null;
    $matches = array();
    foreach (explode("\n", $output) as $line) {
      if (preg_match('/.*?:(\d+):(.*)/', $line, $matches)) {
        if ($current) {
          $msgs[] = $current;
        }
        $line = $matches[1];
        $text = $matches[2];
        if (preg_match('/.*Warning.*/', $text)) {
          $sev = self::LINT_WARNING;
        } else if (preg_match('/.*Advice.*/', $text)) {
          $sev = self::LINT_ADVICE;
        } else {
          $sev = self::LINT_ERROR;
        }
        $current = array('line'     => $line,
                         'msg'      => $text,
                         'severity' => $sev);
      } else if ($current) {
        $current['msg'] .= ' ' . $line;
      }
    }
    if ($current) {
      $msgs[] = $current;
    }

    return $msgs;
  }
}
