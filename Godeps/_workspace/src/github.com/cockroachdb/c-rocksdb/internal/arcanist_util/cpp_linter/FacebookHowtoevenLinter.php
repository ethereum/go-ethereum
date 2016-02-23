<?php
// Copyright 2015-present Facebook. All Rights Reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

final class FacebookHowtoevenLinter extends ArcanistLinter {

  const VERSION = 'fd9192f324c36d28136d14380f0b552a1385b59b';

  private $parsedTargets = array();

  public function getLinterName() {
    return 'Howtoeven';
  }

  protected function getSeverity($code) {
    $severities = array(
      ArcanistLintSeverity::SEVERITY_DISABLED,
      ArcanistLintSeverity::SEVERITY_ADVICE,
      ArcanistLintSeverity::SEVERITY_WARNING,
      ArcanistLintSeverity::SEVERITY_ERROR,
    );
    return idx($severities, $code, ArcanistLintSeverity::SEVERITY_WARNING);
  }

  public function willLintPaths(array $paths) {
    // Cleanup previous runs.
    $this->localExecx("rm -rf _build/_lint");

    // Build compilation database.
    $lintable_paths = $this->getLintablePaths($paths);
    $interesting_paths = $this->getInterestingPaths($lintable_paths);

    if (!$lintable_paths) {
      return;
    }

    // Run lint.
    try {
      $this->localExecx(
        "%C %C -p _build/dev/ %Ls",
        $this->getBinaryPath(),
        $this->getFilteredIssues(),
        $lintable_paths);
    } catch (CommandException $exception) {
      PhutilConsole::getConsole()->writeErr($exception->getMessage());
    }

    // Load results.
    $result = id(
      new SQLite3(
        $this->getProjectRoot().'/_build/_lint/lint.db',
        SQLITE3_OPEN_READONLY))
      ->query("SELECT * FROM raised_issues");

    while ($issue = $result->fetchArray(SQLITE3_ASSOC)) {
      // Skip issues not part of the linted file.
      if (in_array($issue['file'], $interesting_paths)) {
        $this->addLintMessage(id(new ArcanistLintMessage())
          ->setPath($issue['file'])
          ->setLine($issue['line'])
          ->setChar($issue['column'])
          ->setCode('Howtoeven')
          ->setSeverity($this->getSeverity($issue['severity']))
          ->setName('Hte-'.$issue['name'])
          ->setDescription(
            sprintf(
              "%s\n\n%s",
              ($issue['message']) ? $issue['message'] : $issue['description'],
              $issue['explanation']))
          ->setOriginalText(idx($issue, 'original', ''))
          ->setReplacementText(idx($issue, 'replacement', '')));
      }
    }
  }

  public function lintPath($path) {
  }

  /**
   * Get the paths that we know how to lint.
   *
   * The strategy is to first look whether there's an existing compilation
   * database and use that if it's exhaustive. We generate our own only if
   * necessary.
   */
  private function getLintablePaths($paths) {
    // Replace headers with existing sources.
    for ($i = 0; $i < count($paths); $i++) {
      if (preg_match("/\.h$/", $paths[$i])) {
        $header = preg_replace("/\.h$/", ".cpp", $paths[$i]);
        if (file_exists($header)) {
          $paths[$i] = $header;
        }
      }
    }

    // Check if database exists and is exhaustive.
    $available_paths = $this->getAvailablePaths();
    $lintable_paths = array_intersect($paths, $available_paths);
    if ($paths === $lintable_paths) {
      return $lintable_paths;
    }

    // Generate our own database.
    $targets = $this->getTargetsFor($paths);
    if (!$targets) {
      PhutilConsole::getConsole()->writeErr(
        "No build targets found for %s\n",
        implode(', ', $paths));
      return array();
    }

    $this->localExecx("./tools/build/bin/fbconfig.par -r %Ls", $targets);
    $this->localExecx("./tools/build/bin/fbmake.par gen_cdb");

    $available_paths = $this->getAvailablePaths();
    $lintable_paths = array_intersect($paths, $available_paths);
    if ($paths != $lintable_paths) {
      PhutilConsole::getConsole()->writeErr(
        "Can't lint %s\n",
        implode(', ', array_diff($paths, $available_paths)));
    }

    // Return what we know how to lint.
    return $lintable_paths;
  }

  /**
   * Get the available paths in the current compilation database.
   */
  private function getAvailablePaths() {
    $database_path = $this->getProjectRoot()
      .'/_build/dev/compile_commands.json';
    if (!file_exists($database_path)) {
      return array();
    }

    $entries = json_decode(file_get_contents($database_path), true);
    $paths = array();
    foreach ($entries as $entry) {
      $paths[] = $entry['file'];
    }
    return $paths;
  }

  /**
   * Search for the targets directories for the given files.
   */
  private static function getTargetsFor($paths) {
    $targets = array();
    foreach ($paths as $path) {
      while (($path = dirname($path)) !== '.') {
        if (in_array('TARGETS', scandir($path))) {
          $contents = file_get_contents($path.'/TARGETS');
          if (strpos($contents, 'cpp_binary') !== false) {
            $targets[] = $path;
            break;
          }
        }
      }
    }
    return array_unique($targets);
  }

  /**
   * The paths that we actually want to report on.
   */
  private function getInterestingPaths($paths) {
    $headers = array();
    foreach ($paths as $path) {
      $headers[] = preg_replace("/\.cpp$/", ".h", $path);
    }
    return array_merge($paths, $headers);
  }

  /**
   * The path where the binary is located. Will return the current dewey binary
   * unless the `HOWTOEVEN_BUILD` environment variable is set.
   */
  private function getBinaryPath() {
    $path = sprintf(
      "/mnt/dewey/fbcode/.commits/%s/builds/howtoeven/client",
      self::VERSION);

    $build = getenv('HOWTOEVEN_BUILD');
    if ($build) {
      $path = sprintf(
        "./_build/%s/tools/howtoeven/client",
        $build);
      if (!file_exists($path)) {
        PhutilConsole::getConsole()->writeErr(">> %s does not exist\n", $path);
        exit(1);
      }
    }

    return $path;
  }

  /**
   * Execute the command in the root directory.
   */
  private function localExecx($command /* , ... */) {
    $arguments = func_get_args();
    return newv('ExecFuture', $arguments)
      ->setCWD($this->getProjectRoot())
      ->resolvex();
  }

  /**
   * The root of the project.
   */
  private function getProjectRoot() {
    return $this->getEngine()->getWorkingCopy()->getProjectRoot();
  }

  private function getFilteredIssues() {
    $issues = getenv('HOWTOEVEN_ISSUES');
    return ($issues) ? csprintf('-issues %s', $issues) : '';
  }

}
