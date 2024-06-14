"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const task_names_1 = require("hardhat/builtin-tasks/task-names");
const config_1 = require("hardhat/config");
const plugins_1 = require("hardhat/plugins");
require("./type-extensions");
const task_names_2 = require("./task-names");
const merge_reports_1 = require("./merge-reports");
let mochaConfig;
let resolvedQualifiedNames;
let resolvedRemoteContracts = [];
/**
 * Filters out contracts to exclude from report
 * @param  {string}   qualifiedName HRE artifact identifier
 * @param  {string[]} skippable      excludeContracts option values
 * @return {boolean}
 */
function shouldSkipContract(qualifiedName, skippable) {
    for (const item of skippable) {
        if (qualifiedName.includes(item))
            return true;
    }
    return false;
}
/**
 * Method passed to eth-gas-reporter to resolve artifact resources. Loads
 * and processes JSON artifacts
 * @param  {HardhatRuntimeEnvironment} hre.artifacts
 * @param  {String[]}                  skippable    contract *not* to track
 * @return {object[]}                  objects w/ abi and bytecode
 */
function getContracts(artifacts, skippable = []) {
    const contracts = [];
    for (const qualifiedName of resolvedQualifiedNames) {
        if (shouldSkipContract(qualifiedName, skippable)) {
            continue;
        }
        let name;
        let artifact = artifacts.readArtifactSync(qualifiedName);
        // Prefer simple names
        try {
            artifact = artifacts.readArtifactSync(artifact.contractName);
            name = artifact.contractName;
        }
        catch (e) {
            name = qualifiedName;
        }
        contracts.push({
            name: name,
            artifact: {
                abi: artifact.abi,
                bytecode: artifact.bytecode,
                deployedBytecode: artifact.deployedBytecode
            }
        });
    }
    for (const remoteContract of resolvedRemoteContracts) {
        contracts.push({
            name: remoteContract.name,
            artifact: {
                abi: remoteContract.abi,
                bytecode: remoteContract.bytecode,
                bytecodeHash: remoteContract.bytecodeHash,
                deployedBytecode: remoteContract.deployedBytecode
            }
        });
    }
    return contracts;
}
/**
 * Sets reporter options to pass to eth-gas-reporter:
 * > url to connect to client with
 * > artifact format (hardhat)
 * > solc compiler info
 * @param  {HardhatRuntimeEnvironment} hre
 * @return {EthGasReporterConfig}
 */
function getDefaultOptions(hre) {
    const defaultUrl = "http://localhost:8545";
    const defaultCompiler = hre.config.solidity.compilers[0];
    let url;
    // Resolve URL
    if (hre.network.config.url) {
        url = hre.network.config.url;
    }
    else {
        url = defaultUrl;
    }
    return {
        enabled: true,
        url: url,
        metadata: {
            compiler: {
                version: defaultCompiler.version
            },
            settings: {
                optimizer: {
                    enabled: defaultCompiler.settings.optimizer.enabled,
                    runs: defaultCompiler.settings.optimizer.runs
                }
            }
        }
    };
}
/**
 * Merges GasReporter defaults with user's GasReporter config
 * @param  {HardhatRuntimeEnvironment} hre
 * @return {any}
 */
function getOptions(hre) {
    return Object.assign(Object.assign({}, getDefaultOptions(hre)), hre.config.gasReporter);
}
/**
 * Fetches remote bytecode at address and hashes it so these addresses can be
 * added to the tracking at eth-gas-reporter synchronously on init.
 * @param  {EGRAsyncApiProvider}   provider
 * @param  {RemoteContract[] = []} remoteContracts
 * @return {Promise<RemoteContract[]>}
 */
async function getResolvedRemoteContracts(provider, remoteContracts = []) {
    const { default: sha1 } = await Promise.resolve().then(() => __importStar(require("sha1")));
    for (const contract of remoteContracts) {
        let code;
        try {
            contract.bytecode = await provider.getCode(contract.address);
            contract.deployedBytecode = contract.bytecode;
            contract.bytecodeHash = sha1(contract.bytecode);
        }
        catch (error) {
            console.log(`Warning: failed to fetch bytecode for remote contract: ${contract.name}`);
            console.log(`Error was: ${error}\n`);
        }
    }
    return remoteContracts;
}
/**
 * Overrides TASK_TEST_RUN_MOCHA_TEST to (conditionally) use eth-gas-reporter as
 * the mocha test reporter and passes mocha relevant options. These are listed
 * on the `gasReporter` of the user's config.
 */
(0, config_1.subtask)(task_names_1.TASK_TEST_RUN_MOCHA_TESTS).setAction(async (args, hre, runSuper) => {
    let options = getOptions(hre);
    options.getContracts = getContracts.bind(null, hre.artifacts, options.excludeContracts);
    if (options.enabled) {
        // Temporarily skipping when in parallel mode because it crashes and unsure how to resolve...
        if (args.parallel === true) {
            const result = await runSuper();
            console.log("Note: Gas reporting has been skipped because plugin `hardhat-gas-reporter` does not support " +
                "the --parallel flag.");
            return result;
        }
        const { parseSoliditySources, setGasAndPriceRates } = require('eth-gas-reporter/lib/utils');
        const InternalReporterConfig = require('eth-gas-reporter/lib/config');
        // Fetch data from gas and coin price providers
        const originalOptions = options;
        options = new InternalReporterConfig(originalOptions);
        await setGasAndPriceRates(options);
        mochaConfig = hre.config.mocha || {};
        mochaConfig.reporter = "eth-gas-reporter";
        mochaConfig.reporterOptions = options;
        if (hre.network.name === plugins_1.HARDHAT_NETWORK_NAME || options.fast) {
            const { BackwardsCompatibilityProviderAdapter } = await Promise.resolve().then(() => __importStar(require("hardhat/internal/core/providers/backwards-compatibility")));
            const { EGRDataCollectionProvider, EGRAsyncApiProvider } = await Promise.resolve().then(() => __importStar(require("./providers")));
            const wrappedDataProvider = new EGRDataCollectionProvider(hre.network.provider, mochaConfig);
            hre.network.provider = new BackwardsCompatibilityProviderAdapter(wrappedDataProvider);
            const asyncProvider = new EGRAsyncApiProvider(hre.network.provider);
            resolvedRemoteContracts = await getResolvedRemoteContracts(asyncProvider, originalOptions.remoteContracts);
            mochaConfig.reporterOptions.provider = asyncProvider;
            mochaConfig.reporterOptions.blockLimit = hre.network.config.blockGasLimit;
            mochaConfig.attachments = {};
        }
        hre.config.mocha = mochaConfig;
        resolvedQualifiedNames = await hre.artifacts.getAllFullyQualifiedNames();
    }
    return runSuper();
});
(0, config_1.subtask)(task_names_2.TASK_GAS_REPORTER_MERGE_REPORTS)
    .addOptionalVariadicPositionalParam("inputFiles", "Path of several gasReporterOutput.json files to merge", [])
    .setAction(async ({ inputFiles }) => {
    const reports = inputFiles.map((input) => JSON.parse(fs_1.default.readFileSync(input, "utf-8")));
    return (0, merge_reports_1.mergeReports)(reports);
});
/**
 * Task for merging multiple gasReporterOutput.json files generated by eth-gas-reporter
 * This task is necessary when we want to generate different parts of the reports
 * parallelized on different jobs, then merge the results and upload it to codechecks.
 * Gas Report JSON file schema: https://github.com/cgewecke/eth-gas-reporter/blob/master/docs/gasReporterOutput.md
 */
(0, config_1.task)(task_names_2.TASK_GAS_REPORTER_MERGE)
    .addOptionalParam("output", "Target file to save the merged report", "gasReporterOutput.json")
    .addVariadicPositionalParam("input", "A list of gasReporterOutput.json files generated by eth-gas-reporter. Files can be defined using glob patterns")
    .setAction(async (taskArguments, { run }) => {
    const output = path_1.default.resolve(process.cwd(), taskArguments.output);
    // Parse input files and calculate glob patterns
    const { globSync } = await Promise.resolve().then(() => __importStar(require("hardhat/internal/util/glob")));
    const arrayUniq = require("array-uniq");
    const inputFiles = arrayUniq(taskArguments.input.map(globSync).flat())
        .map(inputFile => path_1.default.resolve(inputFile));
    if (inputFiles.length === 0) {
        throw new Error(`No files found for the given input: ${taskArguments.input.join(" ")}`);
    }
    console.log(`Merging ${inputFiles.length} input files:`);
    inputFiles.forEach(inputFile => {
        console.log("  - ", inputFile);
    });
    console.log("\nOutput: ", output);
    const result = await run(task_names_2.TASK_GAS_REPORTER_MERGE_REPORTS, { inputFiles });
    fs_1.default.writeFileSync(output, JSON.stringify(result), "utf-8");
});
//# sourceMappingURL=index.js.map