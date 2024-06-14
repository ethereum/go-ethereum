"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.EthersIgnitionHelper = void 0;
const helpers_1 = require("@nomicfoundation/hardhat-ignition/helpers");
const ignition_core_1 = require("@nomicfoundation/ignition-core");
const plugins_1 = require("hardhat/plugins");
const path_1 = __importDefault(require("path"));
class EthersIgnitionHelper {
    _hre;
    _config;
    type = "ethers";
    _provider;
    constructor(_hre, _config, provider) {
        this._hre = _hre;
        this._config = _config;
        this._provider = provider ?? this._hre.network.provider;
    }
    /**
     * Deploys the given Ignition module and returns the results of the module as
     * Ethers contract instances.
     *
     * @param ignitionModule - The Ignition module to deploy.
     * @param options - The options to use for the deployment.
     * @returns Ethers contract instances for each contract returned by the
     * module.
     */
    async deploy(ignitionModule, { parameters = {}, config: perDeployConfig = {}, defaultSender = undefined, strategy, strategyConfig, deploymentId: givenDeploymentId = undefined, } = {
        parameters: {},
        config: {},
        defaultSender: undefined,
        strategy: undefined,
        strategyConfig: undefined,
        deploymentId: undefined,
    }) {
        const accounts = (await this._hre.network.provider.request({
            method: "eth_accounts",
        }));
        const artifactResolver = new helpers_1.HardhatArtifactResolver(this._hre);
        const resolvedConfig = {
            ...this._config,
            ...perDeployConfig,
        };
        const resolvedStrategyConfig = EthersIgnitionHelper._resolveStrategyConfig(this._hre, strategy, strategyConfig);
        const chainId = Number(await this._hre.network.provider.request({
            method: "eth_chainId",
        }));
        const deploymentId = (0, helpers_1.resolveDeploymentId)(givenDeploymentId, chainId);
        const deploymentDir = this._hre.network.name === "hardhat"
            ? undefined
            : path_1.default.join(this._hre.config.paths.ignition, "deployments", deploymentId);
        const result = await (0, ignition_core_1.deploy)({
            config: resolvedConfig,
            provider: this._provider,
            deploymentDir,
            artifactResolver,
            ignitionModule,
            deploymentParameters: parameters,
            accounts,
            defaultSender,
            strategy,
            strategyConfig: resolvedStrategyConfig,
            maxFeePerGasLimit: this._hre.config.networks[this._hre.network.name]?.ignition
                .maxFeePerGasLimit,
            maxPriorityFeePerGas: this._hre.config.networks[this._hre.network.name]?.ignition
                .maxPriorityFeePerGas,
        });
        if (result.type !== ignition_core_1.DeploymentResultType.SUCCESSFUL_DEPLOYMENT) {
            const message = (0, helpers_1.errorDeploymentResultToExceptionMessage)(result);
            throw new plugins_1.HardhatPluginError("hardhat-ignition-viem", message);
        }
        return EthersIgnitionHelper._toEthersContracts(this._hre, ignitionModule, result);
    }
    static async _toEthersContracts(hre, ignitionModule, result) {
        return Object.fromEntries(await Promise.all(Object.entries(ignitionModule.results).map(async ([name, contractFuture]) => [
            name,
            await this._getContract(hre, contractFuture, result.contracts[contractFuture.id]),
        ])));
    }
    static async _getContract(hre, future, deployedContract) {
        if (!(0, ignition_core_1.isContractFuture)(future)) {
            throw new plugins_1.HardhatPluginError("hardhat-ignition", `Expected contract future but got ${future.id} with type ${future.type} instead`);
        }
        if ("artifact" in future) {
            return hre.ethers.getContractAt(
            // The abi meets the abi spec and we assume we can convert to
            // an acceptable Ethers abi
            future.artifact.abi, deployedContract.address);
        }
        return hre.ethers.getContractAt(future.contractName, deployedContract.address);
    }
    static _resolveStrategyConfig(hre, strategyName, strategyConfig) {
        if (strategyName === undefined) {
            return undefined;
        }
        if (strategyConfig === undefined) {
            const fromHardhatConfig = hre.config.ignition?.strategyConfig?.[strategyName];
            return fromHardhatConfig;
        }
        return strategyConfig;
    }
}
exports.EthersIgnitionHelper = EthersIgnitionHelper;
//# sourceMappingURL=ethers-ignition-helper.js.map