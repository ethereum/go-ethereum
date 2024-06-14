import { Abi } from "./artifact";
import { DeployedContract, ExecutionErrorDeploymentResult } from "./deploy";
/**
 * The information of a deployed contract.
 *
 * @beta
 */
export interface GenericContractInfo extends DeployedContract {
    sourceName: string;
    abi: Abi;
}
/**
 * The result of requesting the status of a deployment. It lists the futures
 * broken down by their status, and includes the deployed contracts.
 *
 * @beta
 */
export interface StatusResult extends Omit<ExecutionErrorDeploymentResult, "type"> {
    chainId: number;
    contracts: {
        [key: string]: GenericContractInfo;
    };
}
//# sourceMappingURL=status.d.ts.map