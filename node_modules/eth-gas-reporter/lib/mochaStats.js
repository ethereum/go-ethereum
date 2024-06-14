/**
 * This file adapted from mocha's stats-collector
 * https://github.com/mochajs/mocha/blob/54475eb4ca35a2c9044a1b8c59a60f09c73e6c01/lib/stats-collector.js#L1-L83
 */
const Date = global.Date;

/**
 * Provides stats such as test duration, number of tests passed / failed etc., by
 * listening for events emitted by `runner`.
 */
function mochaStatsCollector(runner) {
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

module.exports = mochaStatsCollector;
