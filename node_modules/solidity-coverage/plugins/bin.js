#!/usr/bin/env node

/*
  Logs a warning / informational message when user tries to
  invoke 'solidity-coverage' as a shell command. This file
  is listed as the package.json "bin".
*/
const AppUI = require('../lib/ui').AppUI;

(new AppUI()).report('command');
