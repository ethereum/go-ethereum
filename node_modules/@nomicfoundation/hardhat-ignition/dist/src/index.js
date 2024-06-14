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
require("@nomicfoundation/hardhat-verify");
const etherscan_1 = require("@nomicfoundation/hardhat-verify/etherscan");
const ignition_core_1 = require("@nomicfoundation/ignition-core");
const fs_extra_1 = require("fs-extra");
const config_1 = require("hardhat/config");
const plugins_1 = require("hardhat/plugins");
const path_1 = __importDefault(require("path"));
require("./type-extensions");
const calculate_deployment_status_display_1 = require("./ui/helpers/calculate-deployment-status-display");
const bigintReviver_1 = require("./utils/bigintReviver");
const getApiKeyAndUrls_1 = require("./utils/getApiKeyAndUrls");
const resolve_deployment_id_1 = require("./utils/resolve-deployment-id");
const shouldBeHardhatPluginError_1 = require("./utils/shouldBeHardhatPluginError");
const verifyEtherscanContract_1 = require("./utils/verifyEtherscanContract");
/* ignition config defaults */
const IGNITION_DIR = "ignition";
const ignitionScope = (0, config_1.scope)("ignition", "Deploy your smart contracts using Hardhat Ignition");
(0, config_1.extendConfig)((config, userConfig) => {
    /* setup path configs */
    const userPathsConfig = userConfig.paths ?? {};
    config.paths = {
        ...config.paths,
        ignition: path_1.default.resolve(config.paths.root, userPathsConfig.ignition ?? IGNITION_DIR),
    };
    Object.keys(config.networks).forEach((networkName) => {
        const userNetworkConfig = userConfig.networks?.[networkName] ?? {};
        config.networks[networkName].ignition = {
            maxFeePerGasLimit: userNetworkConfig.ignition?.maxFeePerGasLimit,
            maxPriorityFeePerGas: userNetworkConfig.ignition?.maxPriorityFeePerGas,
        };
    });
    /* setup core configs */
    const userIgnitionConfig = userConfig.ignition ?? {};
    config.ignition = userIgnitionConfig;
});
/**
 * Add an `ignition` stub to throw
 */
(0, config_1.extendEnvironment)((hre) => {
    if (hre.ignition === undefined) {
        hre.ignition = {
            type: "stub",
            deploy: () => {
                throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", "Please install either `@nomicfoundation/hardhat-ignition-viem` or `@nomicfoundation/hardhat-ignition-ethers` to use Ignition in your Hardhat tests");
            },
        };
    }
});
ignitionScope
    .task("deploy")
    .addPositionalParam("modulePath", "The path to the module file to deploy")
    .addOptionalParam("parameters", "A relative path to a JSON file to use for the module parameters")
    .addOptionalParam("deploymentId", "Set the id of the deployment")
    .addOptionalParam("defaultSender", "Set the default sender for the deployment")
    .addOptionalParam("strategy", "Set the deployment strategy to use", "basic")
    .addFlag("reset", "Wipes the existing deployment state before deploying")
    .addFlag("verify", "Verify the deployment on Etherscan")
    .setDescription("Deploy a module to the specified network")
    .setAction(async ({ modulePath, parameters: parametersInput, deploymentId: givenDeploymentId, defaultSender, reset, verify, strategy: strategyName, }, hre) => {
    const { default: chalk } = await Promise.resolve().then(() => __importStar(require("chalk")));
    const { default: Prompt } = await Promise.resolve().then(() => __importStar(require("prompts")));
    const { deploy } = await Promise.resolve().then(() => __importStar(require("@nomicfoundation/ignition-core")));
    const { HardhatArtifactResolver } = await Promise.resolve().then(() => __importStar(require("./hardhat-artifact-resolver")));
    const { loadModule } = await Promise.resolve().then(() => __importStar(require("./utils/load-module")));
    const { PrettyEventHandler } = await Promise.resolve().then(() => __importStar(require("./ui/pretty-event-handler")));
    if (verify) {
        if (hre.config.etherscan === undefined ||
            hre.config.etherscan.apiKey === undefined ||
            hre.config.etherscan.apiKey === "") {
            throw new plugins_1.NomicLabsHardhatPluginError("@nomicfoundation/hardhat-ignition", "No etherscan API key configured");
        }
    }
    const chainId = Number(await hre.network.provider.request({
        method: "eth_chainId",
    }));
    const deploymentId = (0, resolve_deployment_id_1.resolveDeploymentId)(givenDeploymentId, chainId);
    const deploymentDir = hre.network.name === "hardhat"
        ? undefined
        : path_1.default.join(hre.config.paths.ignition, "deployments", deploymentId);
    if (chainId !== 31337) {
        if (process.env.HARDHAT_IGNITION_CONFIRM_DEPLOYMENT === undefined) {
            const prompt = await Prompt({
                type: "confirm",
                name: "networkConfirmation",
                message: `Confirm deploy to network ${hre.network.name} (${chainId})?`,
                initial: false,
            });
            if (prompt.networkConfirmation !== true) {
                console.log("Deploy cancelled");
                return;
            }
        }
        if (reset && process.env.HARDHAT_IGNITION_CONFIRM_RESET === undefined) {
            const resetPrompt = await Prompt({
                type: "confirm",
                name: "resetConfirmation",
                message: `Confirm reset of deployment "${deploymentId}" on chain ${chainId}?`,
                initial: false,
            });
            if (resetPrompt.resetConfirmation !== true) {
                console.log("Deploy cancelled");
                return;
            }
        }
    }
    else if (deploymentDir !== undefined) {
        // since we're on hardhat-network
        // check for a previous run of this deploymentId and compare instanceIds
        // if they're different, wipe deployment state
        const instanceFilePath = path_1.default.join(hre.config.paths.cache, ".hardhat-network-instances.json");
        const instanceFileExists = await (0, fs_extra_1.pathExists)(instanceFilePath);
        const instanceFile = instanceFileExists ? require(instanceFilePath) : {};
        const metadata = (await hre.network.provider.request({
            method: "hardhat_metadata",
        }));
        if (instanceFile[deploymentId] !== metadata.instanceId) {
            await (0, fs_extra_1.rm)(deploymentDir, { recursive: true, force: true });
        }
        // save current instanceId to instanceFile for future runs
        instanceFile[deploymentId] = metadata.instanceId;
        await (0, fs_extra_1.ensureDir)(path_1.default.dirname(instanceFilePath));
        await (0, fs_extra_1.writeJSON)(instanceFilePath, instanceFile, { spaces: 2 });
    }
    if (reset) {
        if (deploymentDir === undefined) {
            throw new plugins_1.NomicLabsHardhatPluginError("@nomicfoundation/hardhat-ignition", "Deploy cancelled: Cannot reset deployment on ephemeral Hardhat network");
        }
        else {
            await (0, fs_extra_1.rm)(deploymentDir, { recursive: true, force: true });
        }
    }
    if (strategyName !== "basic" && strategyName !== "create2") {
        throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", "Invalid strategy name, must be either 'basic' or 'create2'");
    }
    await hre.run("compile", { quiet: true });
    const userModule = loadModule(hre.config.paths.ignition, modulePath);
    if (userModule === undefined) {
        throw new plugins_1.NomicLabsHardhatPluginError("@nomicfoundation/hardhat-ignition", "No Ignition modules found");
    }
    let parameters;
    if (parametersInput === undefined) {
        parameters = await resolveParametersFromModuleName(userModule.id, hre.config.paths.ignition);
    }
    else if (parametersInput.endsWith(".json")) {
        parameters = await resolveParametersFromFileName(parametersInput);
    }
    else {
        parameters = resolveParametersString(parametersInput);
    }
    const accounts = (await hre.network.provider.request({
        method: "eth_accounts",
    }));
    const artifactResolver = new HardhatArtifactResolver(hre);
    const executionEventListener = new PrettyEventHandler();
    const strategyConfig = hre.config.ignition.strategyConfig?.[strategyName];
    try {
        const ledgerConnectionStart = () => executionEventListener.ledgerConnectionStart();
        const ledgerConnectionSuccess = () => executionEventListener.ledgerConnectionSuccess();
        const ledgerConnectionFailure = () => executionEventListener.ledgerConnectionFailure();
        const ledgerConfirmationStart = () => executionEventListener.ledgerConfirmationStart();
        const ledgerConfirmationSuccess = () => executionEventListener.ledgerConfirmationSuccess();
        const ledgerConfirmationFailure = () => executionEventListener.ledgerConfirmationFailure();
        try {
            await hre.network.provider.send("hardhat_setLedgerOutputEnabled", [
                false,
            ]);
            hre.network.provider.once("connection_start", ledgerConnectionStart);
            hre.network.provider.once("connection_success", ledgerConnectionSuccess);
            hre.network.provider.once("connection_failure", ledgerConnectionFailure);
            hre.network.provider.on("confirmation_start", ledgerConfirmationStart);
            hre.network.provider.on("confirmation_success", ledgerConfirmationSuccess);
            hre.network.provider.on("confirmation_failure", ledgerConfirmationFailure);
        }
        catch { }
        const result = await deploy({
            config: hre.config.ignition,
            provider: hre.network.provider,
            executionEventListener,
            artifactResolver,
            deploymentDir,
            ignitionModule: userModule,
            deploymentParameters: parameters ?? {},
            accounts,
            defaultSender,
            strategy: strategyName,
            strategyConfig,
            maxFeePerGasLimit: hre.config.networks[hre.network.name]?.ignition.maxFeePerGasLimit,
            maxPriorityFeePerGas: hre.config.networks[hre.network.name]?.ignition
                .maxPriorityFeePerGas,
        });
        try {
            await hre.network.provider.send("hardhat_setLedgerOutputEnabled", [
                true,
            ]);
            hre.network.provider.off("connection_start", ledgerConnectionStart);
            hre.network.provider.off("connection_success", ledgerConnectionSuccess);
            hre.network.provider.off("connection_failure", ledgerConnectionFailure);
            hre.network.provider.off("confirmation_start", ledgerConfirmationStart);
            hre.network.provider.off("confirmation_success", ledgerConfirmationSuccess);
            hre.network.provider.off("confirmation_failure", ledgerConfirmationFailure);
        }
        catch { }
        if (result.type === "SUCCESSFUL_DEPLOYMENT" && verify) {
            console.log("");
            console.log(chalk.bold("Verifying deployed contracts"));
            console.log("");
            await hre.run({ scope: "ignition", task: "verify" }, { deploymentId });
        }
        if (result.type !== "SUCCESSFUL_DEPLOYMENT") {
            process.exitCode = 1;
        }
    }
    catch (e) {
        if (e instanceof ignition_core_1.IgnitionError && (0, shouldBeHardhatPluginError_1.shouldBeHardhatPluginError)(e)) {
            throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", e.message, e);
        }
        throw e;
    }
});
ignitionScope
    .task("visualize")
    .addFlag("noOpen", "Disables opening report in browser")
    .addPositionalParam("modulePath", "The path to the module file to visualize")
    .setDescription("Visualize a module as an HTML report")
    .setAction(async ({ noOpen = false, modulePath }, hre) => {
    const { IgnitionModuleSerializer, batches } = await Promise.resolve().then(() => __importStar(require("@nomicfoundation/ignition-core")));
    const { loadModule } = await Promise.resolve().then(() => __importStar(require("./utils/load-module")));
    const { open } = await Promise.resolve().then(() => __importStar(require("./utils/open")));
    const { writeVisualization } = await Promise.resolve().then(() => __importStar(require("./visualization/write-visualization")));
    await hre.run("compile", { quiet: true });
    const userModule = loadModule(hre.config.paths.ignition, modulePath);
    if (userModule === undefined) {
        throw new plugins_1.NomicLabsHardhatPluginError("@nomicfoundation/hardhat-ignition", "No Ignition modules found");
    }
    else {
        try {
            const serializedIgnitionModule = IgnitionModuleSerializer.serialize(userModule);
            const batchInfo = batches(userModule);
            await writeVisualization({ module: serializedIgnitionModule, batches: batchInfo }, {
                cacheDir: hre.config.paths.cache,
            });
        }
        catch (e) {
            if (e instanceof ignition_core_1.IgnitionError && (0, shouldBeHardhatPluginError_1.shouldBeHardhatPluginError)(e)) {
                throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", e.message, e);
            }
            throw e;
        }
    }
    if (!noOpen) {
        const indexFile = path_1.default.join(hre.config.paths.cache, "visualization", "index.html");
        console.log(`Deployment visualization written to ${indexFile}`);
        open(indexFile);
    }
});
ignitionScope
    .task("status")
    .addPositionalParam("deploymentId", "The id of the deployment to show")
    .setDescription("Show the current status of a deployment")
    .setAction(async ({ deploymentId }, hre) => {
    const { status } = await Promise.resolve().then(() => __importStar(require("@nomicfoundation/ignition-core")));
    const { HardhatArtifactResolver } = await Promise.resolve().then(() => __importStar(require("./hardhat-artifact-resolver")));
    const deploymentDir = path_1.default.join(hre.config.paths.ignition, "deployments", deploymentId);
    const artifactResolver = new HardhatArtifactResolver(hre);
    let statusResult;
    try {
        statusResult = await status(deploymentDir, artifactResolver);
    }
    catch (e) {
        if (e instanceof ignition_core_1.IgnitionError && (0, shouldBeHardhatPluginError_1.shouldBeHardhatPluginError)(e)) {
            throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", e.message, e);
        }
        throw e;
    }
    console.log((0, calculate_deployment_status_display_1.calculateDeploymentStatusDisplay)(deploymentId, statusResult));
});
ignitionScope
    .task("deployments")
    .setDescription("List all deployment IDs")
    .setAction(async (_, hre) => {
    const { listDeployments } = await Promise.resolve().then(() => __importStar(require("@nomicfoundation/ignition-core")));
    const deploymentDir = path_1.default.join(hre.config.paths.ignition, "deployments");
    try {
        const deployments = await listDeployments(deploymentDir);
        for (const deploymentId of deployments) {
            console.log(deploymentId);
        }
    }
    catch (e) {
        if (e instanceof ignition_core_1.IgnitionError && (0, shouldBeHardhatPluginError_1.shouldBeHardhatPluginError)(e)) {
            throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", e.message, e);
        }
        throw e;
    }
});
ignitionScope
    .task("wipe")
    .addPositionalParam("deploymentId", "The id of the deployment with the future to wipe")
    .addPositionalParam("futureId", "The id of the future to wipe")
    .setDescription("Reset a deployment's future to allow rerunning")
    .setAction(async ({ deploymentId, futureId }, hre) => {
    const { wipe } = await Promise.resolve().then(() => __importStar(require("@nomicfoundation/ignition-core")));
    const { HardhatArtifactResolver } = await Promise.resolve().then(() => __importStar(require("./hardhat-artifact-resolver")));
    const deploymentDir = path_1.default.join(hre.config.paths.ignition, "deployments", deploymentId);
    try {
        await wipe(deploymentDir, new HardhatArtifactResolver(hre), futureId);
    }
    catch (e) {
        if (e instanceof ignition_core_1.IgnitionError && (0, shouldBeHardhatPluginError_1.shouldBeHardhatPluginError)(e)) {
            throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", e.message, e);
        }
        throw e;
    }
    console.log(`${futureId} state has been cleared`);
});
ignitionScope
    .task("verify")
    .addFlag("includeUnrelatedContracts", "Include all compiled contracts in the verification")
    .addPositionalParam("deploymentId", "The id of the deployment to verify")
    .setDescription("Verify contracts from a deployment against the configured block explorers")
    .setAction(async ({ deploymentId, includeUnrelatedContracts = false, }, hre) => {
    const { getVerificationInformation } = await Promise.resolve().then(() => __importStar(require("@nomicfoundation/ignition-core")));
    const deploymentDir = path_1.default.join(hre.config.paths.ignition, "deployments", deploymentId);
    if (hre.config.etherscan === undefined ||
        hre.config.etherscan.apiKey === undefined ||
        hre.config.etherscan.apiKey === "") {
        throw new plugins_1.NomicLabsHardhatPluginError("@nomicfoundation/hardhat-ignition", "No etherscan API key configured");
    }
    try {
        for await (const [chainConfig, contractInfo,] of getVerificationInformation(deploymentDir, hre.config.etherscan.customChains, includeUnrelatedContracts)) {
            const apiKeyAndUrls = (0, getApiKeyAndUrls_1.getApiKeyAndUrls)(hre.config.etherscan.apiKey, chainConfig);
            const instance = new etherscan_1.Etherscan(...apiKeyAndUrls);
            console.log(`Verifying contract "${contractInfo.name}" for network ${chainConfig.network}...`);
            const result = await (0, verifyEtherscanContract_1.verifyEtherscanContract)(instance, contractInfo);
            if (result.type === "success") {
                console.log(`Successfully verified contract "${contractInfo.name}" for network ${chainConfig.network}:\n  - ${result.contractURL}`);
                console.log("");
            }
            else {
                if (/already verified/gi.test(result.reason.message)) {
                    const contractURL = instance.getContractUrl(contractInfo.address);
                    console.log(`Contract ${contractInfo.name} already verified on network ${chainConfig.network}:\n  - ${contractURL}`);
                    console.log("");
                    continue;
                }
                else {
                    if (!includeUnrelatedContracts) {
                        throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", `Verification failed. Please run \`hardhat ignition verify ${deploymentId} --include-unrelated-contracts\` to attempt verifying all contracts.`);
                    }
                    else {
                        throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", result.reason.message);
                    }
                }
            }
        }
    }
    catch (e) {
        if (e instanceof ignition_core_1.IgnitionError && (0, shouldBeHardhatPluginError_1.shouldBeHardhatPluginError)(e)) {
            throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", e.message, e);
        }
        throw e;
    }
});
async function resolveParametersFromModuleName(moduleName, ignitionPath) {
    const files = (0, fs_extra_1.readdirSync)(ignitionPath);
    const configFilename = `${moduleName}.config.json`;
    return files.includes(configFilename)
        ? resolveConfigPath(path_1.default.resolve(ignitionPath, configFilename))
        : undefined;
}
async function resolveParametersFromFileName(fileName) {
    const filepath = path_1.default.resolve(process.cwd(), fileName);
    return resolveConfigPath(filepath);
}
async function resolveConfigPath(filepath) {
    try {
        const rawFile = await (0, fs_extra_1.readFile)(filepath);
        return JSON.parse(rawFile.toString(), bigintReviver_1.bigintReviver);
    }
    catch (e) {
        if (e instanceof plugins_1.NomicLabsHardhatPluginError) {
            throw e;
        }
        if (e instanceof Error) {
            throw new plugins_1.NomicLabsHardhatPluginError("@nomicfoundation/hardhat-ignition", `Could not parse parameters from ${filepath}`, e);
        }
        throw e;
    }
}
function resolveParametersString(paramString) {
    try {
        return JSON.parse(paramString, bigintReviver_1.bigintReviver);
    }
    catch (e) {
        if (e instanceof plugins_1.NomicLabsHardhatPluginError) {
            throw e;
        }
        if (e instanceof Error) {
            throw new plugins_1.NomicLabsHardhatPluginError("@nomicfoundation/hardhat-ignition", "Could not parse JSON parameters", e);
        }
        throw e;
    }
}
//# sourceMappingURL=index.js.map