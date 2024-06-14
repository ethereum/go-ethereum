import { ArtifactResolver } from "./types/artifact";
import { DeployConfig, DeploymentParameters, DeploymentResult, StrategyConfig } from "./types/deploy";
import { ExecutionEventListener } from "./types/execution-events";
import { IgnitionModule, IgnitionModuleResult } from "./types/module";
import { EIP1193Provider } from "./types/provider";
/**
 * Deploy an IgnitionModule to the chain
 *
 * @beta
 */
export declare function deploy<ModuleIdT extends string, ContractNameT extends string, IgnitionModuleResultsT extends IgnitionModuleResult<ContractNameT>, StrategyT extends keyof StrategyConfig = "basic">({ config, artifactResolver, provider, executionEventListener, deploymentDir, ignitionModule, deploymentParameters, accounts, defaultSender: givenDefaultSender, strategy, strategyConfig, maxFeePerGasLimit, maxPriorityFeePerGas, }: {
    config?: Partial<DeployConfig>;
    artifactResolver: ArtifactResolver;
    provider: EIP1193Provider;
    executionEventListener?: ExecutionEventListener;
    deploymentDir?: string;
    ignitionModule: IgnitionModule<ModuleIdT, ContractNameT, IgnitionModuleResultsT>;
    deploymentParameters: DeploymentParameters;
    accounts: string[];
    defaultSender?: string;
    strategy?: StrategyT;
    strategyConfig?: StrategyConfig[StrategyT];
    maxFeePerGasLimit?: bigint;
    maxPriorityFeePerGas?: bigint;
}): Promise<DeploymentResult>;
//# sourceMappingURL=deploy.d.ts.map