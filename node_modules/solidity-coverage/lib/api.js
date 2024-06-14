const shell = require('shelljs');
const pify = require('pify');
const fs = require('fs');
const path = require('path');
const istanbul = require('sc-istanbul');
const assert = require('assert');
const _ = require('lodash/lang');

const ConfigValidator = require('./validator');
const Instrumenter = require('./instrumenter');
const Coverage = require('./coverage');
const DataCollector = require('./collector');
const AppUI = require('./ui').AppUI;
const AbiUtils = require('./abi');

/**
 * Coverage Runner
 */
class API {
  constructor(config={}) {
    this.validator = new ConfigValidator();
    this.abiUtils = new AbiUtils();
    this.config = config || {};
    this.testMatrix = {};

    // Validate
    this.validator.validate(this.config);
    this.coverage = new Coverage();
    this.instrumenter = new Instrumenter(this.config);

    // Options
    this.testsErrored = false;

    this.cwd = config.cwd || process.cwd();
    this.abiOutputPath = config.abiOutputPath || "humanReadableAbis.json";
    this.matrixOutputPath = config.matrixOutputPath || "testMatrix.json";
    this.mochaJsonOutputPath = config.mochaJsonOutputPath || "mochaOutput.json";
    this.matrixReporterPath = config.matrixReporterPath || "solidity-coverage/plugins/resources/matrix.js"

    this.defaultHook = () => {};
    this.onServerReady = config.onServerReady           || this.defaultHook;
    this.onTestsComplete = config.onTestsComplete       || this.defaultHook;
    this.onCompileComplete = config.onCompileComplete   || this.defaultHook;
    this.onIstanbulComplete = config.onIstanbulComplete || this.defaultHook;
    this.onPreCompile = config.onPreCompile             || this.defaultHook;

    this.skipFiles = config.skipFiles || [];

    this.log = config.log || console.log;
    this.gasLimit = 0xffffffffff                // default "gas sent" with transactions
    this.gasLimitNumber = 0x1fffffffffffff;     // block gas limit for Hardhat
    this.gasPrice = 0x01;

    this.istanbulFolder = config.istanbulFolder || false;
    this.istanbulReporter = config.istanbulReporter || ['html', 'lcov', 'text', 'json'];

    this.viaIR = config.viaIR;
    this.usingSolcV4 = config.usingSolcV4;
    this.solcOptimizerDetails = config.solcOptimizerDetails;

    this.setLoggingLevel(config.silent);
    this.ui = new AppUI(this.log);
  }

  /**
   * Instruments a set of sources to prepare them for running under coverage
   * @param  {Object[]}  targets (see below)
   * @return {Object[]}          (see below)
   * @example of input/output array:
   * [{
   *   source:         (required) <solidity-source>,
   *   canonicalPath:  (required) <absolute path to source file>
   *   relativePath:   (optional) <rel path to source file for logging>
   * }]
   */
  instrument(targets=[]) {
    let currentFile;      // Keep track of filename in case we crash...
    let started = false;
    let outputs = [];

    try {
      for (let target of targets) {
        currentFile = target.relativePath || target.canonicalPath;

        if(!started){
          started = true;
          this.ui.report('instr-start');
        }

        this.ui.report('instr-item', [currentFile]);

        const instrumented = this.instrumenter.instrument(
          target.source,
          target.canonicalPath
        );

        this.coverage.addContract(instrumented, target.canonicalPath);

        outputs.push({
          canonicalPath: target.canonicalPath,
          relativePath: target.relativePath,
          source: instrumented.contract
        })
      }

    } catch (err) {
      err.message = this.ui.generate('instr-fail', [currentFile]) + err.message;
      throw err;
    }

    return outputs;
  }

  /**
   * Returns a copy of the hit map created during instrumentation.
   * Useful if you'd like to delegate coverage collection to multiple processes.
   * @return {Object} instrumentationData
   */
  getInstrumentationData(){
    return _.cloneDeep(this.instrumenter.instrumentationData)
  }

  /**
   * Sets the hit map object generated during instrumentation. Useful if you'd like
   * to collect data for a pre-existing instrumentation.
   * @param {Object} data
   */
  setInstrumentationData(data={}){
    this.instrumenter.instrumentationData = _.cloneDeep(data);
  }

  /**
   * Generate coverage / write coverage report / run istanbul
   */
  async report(_folder) {
    const folder = _folder || this.istanbulFolder;

    const collector = new istanbul.Collector();
    const reporter = new istanbul.Reporter(false, folder);

    return new Promise((resolve, reject) => {
      try {
        this.coverage.generate(this.instrumenter.instrumentationData);

        const mapping = this.makeKeysRelative(this.coverage.data, this.cwd);
        this.saveCoverage(mapping);

        collector.add(mapping);

        this.istanbulReporter.forEach(report => reporter.add(report));

        // Pify doesn't like this one...
        reporter.write(collector, true, (err) => {
          if (err) return reject(err);

          this.ui.report('istanbul');
          resolve();
        });

      } catch (error) {
        error.message = this.ui.generate('istanbul-fail') + error.message;
        throw error;
      }
    })
  }

  /**
   * Removes coverage build artifacts, kills testrpc.
   */
  async finish() { /* Just a stub now - used to shutdown ganache */}

  // ------------------------------------------ Utils ----------------------------------------------

  // ========
  // Provider
  // ========

  // Hardhat
  async attachToHardhatVM(provider){
    const self = this;
    const useExpandedOpcodeDictionary = this.viaIR || this.usingSolcV4;
    this.collector = new DataCollector(this.instrumenter.instrumentationData, useExpandedOpcodeDictionary);

    if ('init' in provider) {
      // Newer versions of Hardhat initialize the provider lazily, so we need to
      // call provider.init() explicitly. This is a no-op if the provider is
      // already initialized.
      await provider.init();
    }

    let cur = provider;

    // Go down to core HardhatNetworkProvider
    while (cur._wrapped) {
      cur = Object.assign({}, cur._wrapped)
    }
    cur._node._vm.evm.events.on('step', self.collector.step.bind(self.collector))
  }

  // Temporarily disabled because some relevant traces aren't available
  // (maybe bytecode cannot be found)
  /*hardhatTraceHandler(trace, isTraceFromCall){
    for (const step of trace.steps){
      if (trace.bytecode && trace.bytecode._pcToInstruction){
        const instruction = trace.bytecode._pcToInstruction.get(step.pc)
        this.collector.trackHardhatEVMInstruction(instruction)
      }
    }
  }*/

  // ==========================
  // Test Matrix Data Collector
  // ==========================
  /**
   * @param  {Object} testInfo Mocha object passed to reporter 'test end' event
   */
  collectTestMatrixData(testInfo){
    const hashes = Object.keys(this.instrumenter.instrumentationData);
    const title = testInfo.title;
    const file = path.relative(this.cwd, testInfo.file);

    for (const hash of hashes){
      const {
        contractPath,
        hits,
        type,
        id
      } = this.instrumenter.instrumentationData[hash];

      if (type === 'line'){
        if (!this.testMatrix[contractPath]){
          this.testMatrix[contractPath] = {};
        }
        if (!this.testMatrix[contractPath][id]){
          this.testMatrix[contractPath][id] = [];
        }

        if (hits > 0){
          // Search for and exclude duplicate entries
          let duplicate = false;
          for (const item of this.testMatrix[contractPath][id]){
            if (item.title === title && item.file === file){
              duplicate = true;
              break;
            }
          }

          if (!duplicate) {
            this.testMatrix[contractPath][id].push({title, file});
          }

          // Reset line data
          this.instrumenter.instrumentationData[hash].hits = 0;
        }
      }
    }
  }

  // ========
  // File I/O
  // ========

  saveCoverage(data){
    const covPath = path.join(this.cwd, "coverage.json");
    fs.writeFileSync(covPath, JSON.stringify(data));
  }

  saveTestMatrix(){
    const matrixPath = path.join(this.cwd, this.matrixOutputPath);
    const mapping = this.makeKeysRelative(this.testMatrix, this.cwd);
    fs.writeFileSync(matrixPath, JSON.stringify(mapping, null, ' '));
  }

  saveMochaJsonOutput(data){
    const outputPath = path.join(this.cwd, this.mochaJsonOutputPath);
    fs.writeFileSync(outputPath, JSON.stringify(data, null, ' '));
  }

  saveHumanReadableAbis(data){
    const abiPath = path.join(this.cwd, this.abiOutputPath);
    fs.writeFileSync(abiPath, JSON.stringify(data, null, ' '));
  }

  // =====
  // Paths
  // =====
  //
  /**
   * Relativizes path keys so that istanbul report can be read on Windows
   * @param  {Object} map  coverage map generated by coverageMap
   * @param  {String} wd   working directory
   * @return {Object}      map with relativized keys
   */
  makeKeysRelative(map, wd) {
    const newCoverage = {};

    Object
     .keys(map)
     .forEach(pathKey => newCoverage[path.relative(wd, pathKey)] = map[pathKey]);

    return newCoverage;
  }

  // =======
  // Logging
  // =======

  /**
   * Turn logging off (for CI)
   * @param {Boolean} isSilent
   */
  setLoggingLevel(isSilent) {
    if (isSilent) this.log = () => {};
  }

}

module.exports = API;
