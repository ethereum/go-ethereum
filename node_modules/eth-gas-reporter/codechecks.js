const { join } = require("path");
const fs = require("fs");
const { codechecks } = require("@codechecks/client");
const CodeChecksReport = require("eth-gas-reporter/lib/codechecksReport");

/**
 * Consumed by codecheck command when user's .yml lists
 * `eth-gas-reporter/codechecks`. The reporter dumps collected
 * data to the project root whenever `process.env.CI` is true. This
 * file processes it and runs the relevant codechecks routines.
 * >
 * > Source: krzkaczor/truffle-codechecks.
 * >
 */
module.exports.default = async function gasReporter(options = {}) {
  let output;
  let file = "gasReporterOutput.json";

  // Load gas reporter output
  try {
    output = JSON.parse(fs.readFileSync(file, "utf-8"));
  } catch (error) {
    const message =
      `Error: Couldn't load data from "${file}".\n` +
      `If you're using codechecks locally make sure you set ` +
      `the environment variable "CI" to "true" before running ` +
      `your tests. ( ex: CI=true npm test )`;

    console.log(message);
    return;
  }

  // Lets monorepo subcomponents individuate themselves
  output.namespace = options.name
    ? `${output.namespace}:${options.name}`
    : output.namespace;

  let report = new CodeChecksReport(output.config);
  report.generate(output.info);

  try {
    await codechecks.saveValue(output.namespace, report.newData);
    console.log(`Successful save: output.namespace was: ${output.namespace}`);
  } catch (err) {
    console.log(
      `If you have a chance, report this incident to the eth-gas-reporter github issues.`
    );
    console.log(`Codechecks errored running 'saveValue'...\n${err}\n`);
    console.log(`output.namespace was: ${output.namespace}`);
    console.log(`Saved gas-reporter data was: ${report.newData}`);
  }

  // Exit early on merge commit / push build
  if (!codechecks.isPr()) {
    return;
  }

  // Get historical data for each pr commit
  try {
    output.config.previousData = await codechecks.getValue(output.namespace);
  } catch (err) {
    console.log(
      `If you have a chance, report this incident to the eth-gas-reporter github issues.`
    );
    console.log(`Codechecks errored running 'getValue'...\n${err}\n`);
    return;
  }

  report = new CodeChecksReport(output.config);
  const table = report.generate(output.info);
  const shortDescription = report.getShortDescription();

  // Support multiple reports
  const checkName = options.name ? `Gas Usage: ${options.name}` : `Gas Usage`;

  // Submit report
  try {
    const payload = {
      name: checkName,
      shortDescription: shortDescription,
      longDescription: table
    };

    report.success
      ? await codechecks.success(payload)
      : await codechecks.failure(payload);
  } catch (err) {
    console.log(
      `If you have a chance, report this incident to the eth-gas-reporter github issues.`
    );
    console.log(`Codechecks errored running .success or .failure\n${err}\n`);
    console.log(`Short description was: ${shortDescription}`);
    console.log(`Table was: ${table}`);
  }
};
