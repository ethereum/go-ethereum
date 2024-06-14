import fs from "fs"
import path from "path"
import { TASK_TEST_RUN_MOCHA_TESTS } from "hardhat/builtin-tasks/task-names";
import { task, subtask } from "hardhat/config";
import { HARDHAT_NETWORK_NAME, HardhatPluginError } from "hardhat/plugins";

import type { EGRAsyncApiProvider as EGRAsyncApiProviderT } from "./providers";

import {
  HardhatArguments,
  HttpNetworkConfig,
  NetworkConfig,
  EthereumProvider,
  HardhatRuntimeEnvironment,
  Artifact,
  Artifacts
} from "hardhat/types";

import "./type-extensions"
import { EthGasReporterConfig, EthGasReporterOutput, RemoteContract } from "./types";
import { TASK_GAS_REPORTER_MERGE, TASK_GAS_REPORTER_MERGE_REPORTS } from "./task-names";
import { mergeReports } from "./merge-reports";

let mochaConfig;
let resolvedQualifiedNames: string[]
let resolvedRemoteContracts: RemoteContract[] = [];

/**
 * Filters out contracts to exclude from report
 * @param  {string}   qualifiedName HRE artifact identifier
 * @param  {string[]} skippable      excludeContracts option values
 * @return {boolean}
 */
function shouldSkipContract(qualifiedName: string, skippable: string[]): boolean {
  for (const item of skippable){
    if (qualifiedName.includes(item)) return true;
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
function getContracts(artifacts: Artifacts, skippable: string[] = []) : any[] {
  const contracts = [];

  for (const qualifiedName of resolvedQualifiedNames) {
    if (shouldSkipContract(qualifiedName, skippable)){
      continue;
    }

    let name: string;
    let artifact = artifacts.readArtifactSync(qualifiedName)

    // Prefer simple names
    try {
      artifact = artifacts.readArtifactSync(artifact.contractName);
      name = artifact.contractName;
    } catch (e) {
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

  for (const remoteContract of resolvedRemoteContracts){
    contracts.push({
      name: remoteContract.name,
      artifact: {
        abi: remoteContract.abi,
        bytecode: remoteContract.bytecode,
        bytecodeHash: remoteContract.bytecodeHash,
        deployedBytecode: remoteContract.deployedBytecode
      }
    })
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
function getDefaultOptions(hre: HardhatRuntimeEnvironment): EthGasReporterConfig {
  const defaultUrl = "http://localhost:8545";
  const defaultCompiler = hre.config.solidity.compilers[0]

  let url: any;
  // Resolve URL
  if ((<HttpNetworkConfig>hre.network.config).url) {
    url = (<HttpNetworkConfig>hre.network.config).url;
  } else {
    url = defaultUrl;
  }

  return {
    enabled: true,
    url: <string>url,
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
  }
}

/**
 * Merges GasReporter defaults with user's GasReporter config
 * @param  {HardhatRuntimeEnvironment} hre
 * @return {any}
 */
function getOptions(hre: HardhatRuntimeEnvironment): any {
  return { ...getDefaultOptions(hre), ...(hre.config as any).gasReporter };
}

/**
 * Fetches remote bytecode at address and hashes it so these addresses can be
 * added to the tracking at eth-gas-reporter synchronously on init.
 * @param  {EGRAsyncApiProvider}   provider
 * @param  {RemoteContract[] = []} remoteContracts
 * @return {Promise<RemoteContract[]>}
 */
async function getResolvedRemoteContracts(
  provider: EGRAsyncApiProviderT,
  remoteContracts: RemoteContract[] = []
) : Promise <RemoteContract[]> {
  const { default : sha1 } = await import("sha1");
  for (const contract of remoteContracts){
    let code;
    try {
      contract.bytecode = await provider.getCode(contract.address);
      contract.deployedBytecode = contract.bytecode;
      contract.bytecodeHash = sha1(contract.bytecode);
    } catch (error){
      console.log(`Warning: failed to fetch bytecode for remote contract: ${contract.name}`)
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
subtask(TASK_TEST_RUN_MOCHA_TESTS).setAction(
  async (args: any, hre, runSuper) => {

    let options = getOptions(hre);
    options.getContracts = getContracts.bind(null, hre.artifacts, options.excludeContracts);

    if (options.enabled) {
      // Temporarily skipping when in parallel mode because it crashes and unsure how to resolve...
      if (args.parallel === true) {
        const result = await runSuper();
        console.log(
          "Note: Gas reporting has been skipped because plugin `hardhat-gas-reporter` does not support " +
          "the --parallel flag."
        );
        return result;
      }


      const { parseSoliditySources, setGasAndPriceRates } = require('eth-gas-reporter/lib/utils');
      const InternalReporterConfig  = require('eth-gas-reporter/lib/config');

      // Fetch data from gas and coin price providers
      const originalOptions = options
      options = new InternalReporterConfig(originalOptions);
      await setGasAndPriceRates(options);

      mochaConfig = hre.config.mocha || {};
      mochaConfig.reporter = "eth-gas-reporter";
      mochaConfig.reporterOptions = options;

      if (hre.network.name === HARDHAT_NETWORK_NAME || options.fast){

        const {
          BackwardsCompatibilityProviderAdapter
        } = await import("hardhat/internal/core/providers/backwards-compatibility")

        const {
          EGRDataCollectionProvider,
          EGRAsyncApiProvider
        } = await import("./providers");

        const wrappedDataProvider= new EGRDataCollectionProvider(hre.network.provider,mochaConfig);
        hre.network.provider = new BackwardsCompatibilityProviderAdapter(wrappedDataProvider);

        const asyncProvider = new EGRAsyncApiProvider(hre.network.provider);
        resolvedRemoteContracts = await getResolvedRemoteContracts(
          asyncProvider,
          originalOptions.remoteContracts
        );

        mochaConfig.reporterOptions.provider = asyncProvider;
        mochaConfig.reporterOptions.blockLimit = (<any>hre.network.config).blockGasLimit as number;
        mochaConfig.attachments = {};
      }

      hre.config.mocha = mochaConfig;
      resolvedQualifiedNames = await hre.artifacts.getAllFullyQualifiedNames();
    }

    return runSuper();
  }
);

subtask(TASK_GAS_REPORTER_MERGE_REPORTS)
  .addOptionalVariadicPositionalParam(
    "inputFiles",
    "Path of several gasReporterOutput.json files to merge",
    []
  )
  .setAction(async ({ inputFiles }: { inputFiles: string[] }) => {
    const reports = inputFiles.map((input) => JSON.parse(fs.readFileSync(input, "utf-8")));
    return mergeReports(reports);
  })

/**
 * Task for merging multiple gasReporterOutput.json files generated by eth-gas-reporter
 * This task is necessary when we want to generate different parts of the reports
 * parallelized on different jobs, then merge the results and upload it to codechecks.
 * Gas Report JSON file schema: https://github.com/cgewecke/eth-gas-reporter/blob/master/docs/gasReporterOutput.md
 */
task(TASK_GAS_REPORTER_MERGE)
  .addOptionalParam(
    "output",
    "Target file to save the merged report",
    "gasReporterOutput.json"
  )
  .addVariadicPositionalParam(
		"input",
		"A list of gasReporterOutput.json files generated by eth-gas-reporter. Files can be defined using glob patterns"
	)
  .setAction(async (taskArguments, { run }) => {
		const output = path.resolve(process.cwd(), taskArguments.output);

		// Parse input files and calculate glob patterns
    const { globSync } = await import("hardhat/internal/util/glob");
    const arrayUniq = require("array-uniq");
		const inputFiles = arrayUniq(taskArguments.input.map(globSync).flat())
      .map(inputFile => path.resolve(inputFile));

		if (inputFiles.length === 0) {
			throw new Error(`No files found for the given input: ${taskArguments.input.join(" ")}`);
		}

		console.log(`Merging ${inputFiles.length} input files:`);
		inputFiles.forEach(inputFile => {
			console.log("  - ", inputFile);
		});

		console.log("\nOutput: ", output);

		const result = await run(TASK_GAS_REPORTER_MERGE_REPORTS, { inputFiles });

		fs.writeFileSync(output, JSON.stringify(result), "utf-8");
	});
