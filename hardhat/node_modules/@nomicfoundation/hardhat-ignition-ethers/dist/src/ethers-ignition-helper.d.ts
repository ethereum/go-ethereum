import { DeployConfig, DeploymentParameters, EIP1193Provider, IgnitionModule, IgnitionModuleResult, NamedArtifactContractAtFuture, NamedArtifactContractDeploymentFuture, StrategyConfig } from "@nomicfoundation/ignition-core";
import { Contract } from "ethers";
import { HardhatRuntimeEnvironment } from "hardhat/types";
export type IgnitionModuleResultsTToEthersContracts<ContractNameT extends string, IgnitionModuleResultsT extends IgnitionModuleResult<ContractNameT>> = {
    [contract in keyof IgnitionModuleResultsT]: IgnitionModuleResultsT[contract] extends NamedArtifactContractDeploymentFuture<ContractNameT> | NamedArtifactContractAtFuture<ContractNameT> ? TypeChainEthersContractByName<ContractNameT> : Contract;
};
export type TypeChainEthersContractByName<ContractNameT> = Contract;
export declare class EthersIgnitionHelper {
    private _hre;
    private _config?;
    type: string;
    private _provider;
    constructor(_hre: HardhatRuntimeEnvironment, _config?: Partial<DeployConfig> | undefined, provider?: EIP1193Provider);
    /**
     * Deploys the given Ignition module and returns the results of the module as
     * Ethers contract instances.
     *
     * @param ignitionModule - The Ignition module to deploy.
     * @param options - The options to use for the deployment.
     * @returns Ethers contract instances for each contract returned by the
     * module.
     */
    deploy<ModuleIdT extends string, ContractNameT extends string, IgnitionModuleResultsT extends IgnitionModuleResult<ContractNameT>, StrategyT extends keyof StrategyConfig = "basic">(ignitionModule: IgnitionModule<ModuleIdT, ContractNameT, IgnitionModuleResultsT>, { parameters, config: perDeployConfig, defaultSender, strategy, strategyConfig, deploymentId: givenDeploymentId, displayUi, }?: {
        parameters?: DeploymentParameters | string;
        config?: Partial<DeployConfig>;
        defaultSender?: string;
        strategy?: StrategyT;
        strategyConfig?: StrategyConfig[StrategyT];
        deploymentId?: string;
        displayUi?: boolean;
    }): Promise<IgnitionModuleResultsTToEthersContracts<ContractNameT, IgnitionModuleResultsT>>;
    private static _toEthersContracts;
    private static _getContract;
    private static _resolveStrategyConfig;
}
//# sourceMappingURL=ethers-ignition-helper.d.ts.map