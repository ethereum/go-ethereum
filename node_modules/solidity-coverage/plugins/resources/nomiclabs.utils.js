const shell = require('shelljs');
const globby = require('globby');
const pluginUtils = require("./plugin.utils");
const path = require('path');
const DataCollector = require("./../../lib/collector")
const semver = require("semver")
const util = require('util')

// =============================
// Nomiclabs Plugin Utils
// =============================

/**
 * Returns a list of test files to pass to mocha.
 * @param  {String}   files   file or glob
 * @return {String[]}         list of files to pass to mocha
 */
function getTestFilePaths(files){
  const target = globby.sync([files])

  // Hardhat supports js & ts
  const testregex = /.*\.(js|ts)$/;
  return target.filter(f => f.match(testregex) != null);
}

/**
 * Normalizes Hardhat paths / logging for use by the plugin utilities and
 * attaches them to the config
 * @param  {HardhatConfig} config
 * @return {HardhatConfig}        updated config
 */
function normalizeConfig(config, args={}){
  let sources;

  (args.sources)
    ? sources = path.join(config.paths.sources, args.sources)
    : sources = config.paths.sources;

  if (!path.isAbsolute(sources)) {
    sources = path.join(config.paths.root, sources);
  }

  if (config.solidity && config.solidity.compilers.length) {
    config.viaIR = isUsingViaIR(config.solidity);
    config.usingSolcV4 = isUsingSolcV4(config.solidity);
  }

  config.workingDir = config.paths.root;
  config.contractsDir = sources;
  config.testDir = config.paths.tests;
  config.artifactsDir = config.paths.artifacts;
  config.logger = config.logger ? config.logger : {log: null};
  config.solcoverjs = args.solcoverjs
  config.gasReporter = { enabled: false }
  config.matrix = args.matrix;

  try {
    const hardhatPackage = require('hardhat/package.json');
    if (semver.gt(hardhatPackage.version, '2.0.3')){
      config.useHardhatDefaultPaths = true;
    }
  } catch(e){ /* ignore */ }

  return config;
}

function isUsingSolcV4(solidity) {
  for (compiler of solidity.compilers) {
    if (compiler.version && semver.lt(compiler.version, '0.5.0')) {
      return true;
    }
  }
  if (solidity.overrides) {
    for (key of Object.keys(solidity.overrides)){
      if (solidity.overrides[key].version && semver.lt(solidity.overrides[key].version, '0.5.0')) {
        return true;
      }
    }
  }
  return false;
}

function isUsingViaIR(solidity) {
  for (compiler of solidity.compilers) {
    if (compiler.settings && compiler.settings.viaIR) {
      return true;
    }
  }
  if (solidity.overrides) {
    for (key of Object.keys(solidity.overrides)){
      if (solidity.overrides[key].settings && solidity.overrides[key].settings.viaIR) {
        return true;
      }
    }
  }
  return false;
}

async function setupHardhatNetwork(env, api, ui){
  const hardhatPackage = require('hardhat/package.json');
  const { createProvider } = require("hardhat/internal/core/providers/construction");
  const { HARDHAT_NETWORK_NAME } = require("hardhat/plugins")

  // after 2.15.0, the internal createProvider function has a different signature
  const newCreateProviderSignature = semver.satisfies(hardhatPackage.version, "^2.15.0");

  let provider, networkConfig;

  // HardhatEVM
  networkConfig = env.network.config;
  configureHardhatEVMGas(networkConfig, api);

  if (newCreateProviderSignature) {
    provider = await createProvider(
      env.config,
      HARDHAT_NETWORK_NAME,
      env.artifacts,
    )
  } else {
    provider = createProvider(
      HARDHAT_NETWORK_NAME,
      networkConfig,
      env.config.paths,
      env.artifacts,
    )
  }

  return configureNetworkEnv(
    env,
    HARDHAT_NETWORK_NAME,
    networkConfig,
    provider
  )
}

function requiresEVMConfiguration(networkConfig, api) {
  return (
    networkConfig.allowUnlimitedContractSize !== true ||
    networkConfig.blockGasLimit !== api.gasLimitNumber ||
    networkConfig.gas !==  api.gasLimit ||
    networkConfig.gasPrice !== api.gasPrice ||
    networkConfig.initialBaseFeePerGas !== 0
  )
}

function configureHardhatEVMGas(networkConfig, api){
  networkConfig.allowUnlimitedContractSize = true;
  networkConfig.blockGasLimit = api.gasLimitNumber;
  networkConfig.gas =  api.gasLimit;
  networkConfig.gasPrice = api.gasPrice;
  networkConfig.initialBaseFeePerGas = 0;
}

function configureNetworkEnv(env, networkName, networkConfig, provider){
  env.config.networks[networkName] = networkConfig;
  env.config.defaultNetwork = networkName;

  env.network = Object.assign(env.network, {
    name: networkName,
    config: networkConfig,
    provider: provider,
    isHardhatEVM: true
  });

  env.ethereum = provider;

  // Return a reference so we can set the from account
  return env.network;
}

/**
 * Configures mocha to generate a json object which maps which tests
 * hit which lines of code.
 */
function collectTestMatrixData(args, env, api){
  if (args.matrix){
    mochaConfig = env.config.mocha || {};
    mochaConfig.reporter = api.matrixReporterPath;
    mochaConfig.reporterOptions = {
      collectTestMatrixData: api.collectTestMatrixData.bind(api),
      saveMochaJsonOutput: api.saveMochaJsonOutput.bind(api),
      cwd: api.cwd
    }
    env.config.mocha = mochaConfig;
  }
}

/**
 * Returns all Hardhat artifacts.
 * @param  {HRE} env
 * @return {Artifact[]}
 */
async function getAllArtifacts(env){
  const all = [];
  const qualifiedNames = await env.artifacts.getArtifactPaths();
  for (const name of qualifiedNames){
    all.push(require(name));
  }
  return all;
}

/**
 * Compiles project
 * Collects all artifacts from Hardhat project,
 * Converts them to a format that can be consumed by api.abiUtils.diff
 * Saves them to `api.abiOutputPath`
 * @param  {HRE}    env
 * @param  {SolidityCoverageAPI} api
 */
async function generateHumanReadableAbiList(env, api, TASK_COMPILE){
  await env.run(TASK_COMPILE);
  const _artifacts = await getAllArtifacts(env);
  const list = api.abiUtils.generateHumanReadableAbiList(_artifacts)
  api.saveHumanReadableAbis(list);
}

/**
 * Sets the default `from` account field in the network that will be used.
 * This needs to be done after accounts are fetched from the launched client.
 * @param {env} config
 * @param {Array}         accounts
 */
function setNetworkFrom(networkConfig, accounts){
  if (!networkConfig.from){
    networkConfig.from = accounts[0];
  }
}

// TODO: Hardhat cacheing??
/**
 * Generates a path to a temporary compilation cache directory
 * @param  {HardhatConfig} config
 * @return {String}        .../.coverage_cache
 */
function tempCacheDir(config){
  return path.join(config.paths.root, '.coverage_cache');
}

/**
 * Silently removes temporary folders and calls api.finish to shut server down
 * @param  {HardhatConfig}     config
 * @param  {SolidityCoverage}  api
 * @return {Promise}
 */
async function finish(config, api, shouldKill){
  const {
    tempContractsDir,
    tempArtifactsDir
  } = pluginUtils.getTempLocations(config);

  shell.config.silent = true;
  shell.rm('-Rf', tempContractsDir);
  shell.rm('-Rf', tempArtifactsDir);
  shell.rm('-Rf', path.join(config.paths.root, '.coverage_cache'));
  shell.config.silent = false;

  if (api) await api.finish();
  if (shouldKill) process.exit(1)
}

module.exports = {
  configureHardhatEVMGas,
  requiresEVMConfiguration,
  normalizeConfig,
  finish,
  tempCacheDir,
  setupHardhatNetwork,
  getTestFilePaths,
  setNetworkFrom,
  collectTestMatrixData,
  getAllArtifacts,
  generateHumanReadableAbiList
}

