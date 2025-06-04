const mocha = require("mocha");
const inherits = require("util").inherits;
const Spec = mocha.reporters.Spec;
const path = require('path');

/**
 * This file adapted from mocha's stats-collector
 * https://github.com/mochajs/mocha/blob/54475eb4ca35a2c9044a1b8c59a60f09c73e6c01/lib/stats-collector.js#L1-L83
 */
const Date = global.Date;

/**
 * Provides stats such as test duration, number of tests passed / failed etc., by
 * listening for events emitted by `runner`.
 */
function mochaStats(runner) {
  const stats = {
    suites: 0,
    tests: 0,
    passes: 0,
    pending: 0,
    failures: 0
  };

  if (!runner) throw new Error("Missing runner argument");

  runner.stats = stats;

  runner.on("pass", () => stats.passes++);
  runner.on("fail", () => stats.failures++);
  runner.on("pending", () => stats.pending++);
  runner.on("test end", () => stats.tests++);

  runner.once("start", () => (stats.start = new Date()));

  runner.once("end", function() {
    stats.end = new Date();
    stats.duration = stats.end - stats.start;
  });
}

/**
 * Based on the Mocha 'Spec' reporter.
 *
 * Watches an Ethereum test suite run and collects data about which tests hit
 * which lines of code. This "test matrix" can be used as an input to fault localization tools
 * like: https://github.com/JoranHonig/tarantula
 *
 * Mocha's JSON reporter output is also generated and saved to a separate file
 *
 * @param {Object} runner  mocha's runner
 * @param {Object} options reporter.options (see README example usage)
 */
function Matrix(runner, options) {
  // Spec reporter
  Spec.call(this, runner, options);

  const self = this;
  const tests = [];
  const failures = [];
  const passes = [];

  // Initialize stats for Mocha 6+ epilogue
  if (!runner.stats) {
    mochaStats(runner);
    this.stats = runner.stats;
  }

  runner.on("test end", (info) => {
    options.reporterOptions.collectTestMatrixData(info);
    tests.push(info);
  });

  runner.on('pass', function(info) {
    passes.push(info)
  })
  runner.on('fail', function(info) {
    failures.push(info)
  });

  runner.once('end', function() {
    delete self.stats.start;
    delete self.stats.end;
    delete self.stats.duration;

    var obj = {
      stats: self.stats,
      tests: tests.map(clean),
      failures: failures.map(clean),
      passes: passes.map(clean)
    };
    runner.testResults = obj;
    options.reporterOptions.saveMochaJsonOutput(obj)
  });

  // >>>>>>>>>>>>>>>>>>>>>>>>>
  // Mocha JSON Reporter Utils
  // Code taken from:
  // https://mochajs.org/api/reporters_json.js.html
  // >>>>>>>>>>>>>>>>>>>>>>>>>
  function clean(info) {
    var err = info.err || {};
    if (err instanceof Error) {
      err = errorJSON(err);
    }
    return {
      title: info.title,
      fullTitle: info.fullTitle(),
      file: path.relative(options.reporterOptions.cwd, info.file),
      currentRetry: info.currentRetry(),
      err: cleanCycles(err)
    };
  }

  function cleanCycles(obj) {
    var cache = [];
    return JSON.parse(
      JSON.stringify(obj, function(key, value) {
        if (typeof value === 'object' && value !== null) {
          if (cache.indexOf(value) !== -1) {
            // Instead of going in a circle, we'll print [object Object]
            return '' + value;
          }
          cache.push(value);
        }
        return value;
      })
    );
  }

  function errorJSON(err) {
    var res = {};
    Object.getOwnPropertyNames(err).forEach(function(key) {
      res[key] = err[key];
    }, err);
    return res;
  }
}

/**
 * Inherit from `Base.prototype`.
 */
inherits(Matrix, Spec);

module.exports = Matrix;