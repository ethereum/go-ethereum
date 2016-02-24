<?php

/**
 * Uses google's cpplint.py to check code. RocksDB team forked this file from
 * phabricator's /src/lint/linter/ArcanistCpplintLinter.php, and customized it
 * for its own use.
 *
 * You can get it here:
 * http://google-styleguide.googlecode.com/svn/trunk/cpplint/cpplint.py
 * @group linter
 */
final class ArcanistCpplintLinter extends ArcanistLinter {

  public function willLintPaths(array $paths) {
    return;
  }

  public function getLinterName() {
    return 'cpplint.py';
  }

  public function getLintPath() {
    $bin = 'cpplint.py';
    // Search under current dir
    list($err) = exec_manual('which %s/%s', $this->linterDir(), $bin);
    if (!$err) {
      return $this->linterDir().'/'.$bin;
    }

    // Look for globally installed cpplint.py
    list($err) = exec_manual('which %s', $bin);
    if ($err) {
      throw new ArcanistUsageException(
        "cpplint.py does not appear to be installed on this system. Install ".
        "it (e.g., with 'wget \"http://google-styleguide.googlecode.com/".
        "svn/trunk/cpplint/cpplint.py\"') ".
        "in your .arcconfig to point to the directory where it resides. ".
        "Also don't forget to chmod a+x cpplint.py!");
    }

    return $bin;
  }

  public function lintPath($path) {
    $bin = $this->getLintPath();
    $path = $this->rocksdbDir().'/'.$path;

    $f = new ExecFuture("%C $path", $bin);

    list($err, $stdout, $stderr) = $f->resolve();

    if ($err === 2) {
      throw new Exception("cpplint failed to run correctly:\n".$stderr);
    }

    $lines = explode("\n", $stderr);
    $messages = array();
    foreach ($lines as $line) {
      $line = trim($line);
      $matches = null;
      $regex = '/^[^:]+:(\d+):\s*(.*)\s*\[(.*)\] \[(\d+)\]$/';
      if (!preg_match($regex, $line, $matches)) {
        continue;
      }
      foreach ($matches as $key => $match) {
        $matches[$key] = trim($match);
      }
      $message = new ArcanistLintMessage();
      $message->setPath($path);
      $message->setLine($matches[1]);
      $message->setCode($matches[3]);
      $message->setName($matches[3]);
      $message->setDescription($matches[2]);
      $message->setSeverity(ArcanistLintSeverity::SEVERITY_WARNING);
      $this->addLintMessage($message);
    }
  }

  // The path of this linter
  private function linterDir() {
    return dirname(__FILE__);
  }

  // TODO(kaili) a quick and dirty way to figure out rocksdb's root dir.
  private function rocksdbDir() {
    return $this->linterDir()."/../..";
  }
}
