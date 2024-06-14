import { ModuleParameters } from "./module";
/**
 * Configuration options for the deployment.
 *
 * @beta
 */
export interface DeployConfig {
    /**
     * The interval, in milliseconds, between checks to see if a new block
     * has been created
     */
    blockPollingInterval: number;
    /**
     * The amount of time, in milliseconds, to wait on a transaction before
     * bumping its fees.
     */
    timeBeforeBumpingFees: number;
    /**
     * The maximum amount of times a transaction is bumped.
     */
    maxFeeBumps: number;
    /**
     * The number of block confirmations to wait before considering
     * a transaction to be confirmed during Ignition execution.
     */
    requiredConfirmations: number;
}
/**
 * The result of running a deployment.
 *
 * @beta
 */
export type DeploymentResult = ValidationErrorDeploymentResult | ReconciliationErrorDeploymentResult | ExecutionErrorDeploymentResult | PreviousRunErrorDeploymentResult | SuccessfulDeploymentResult;
/**
 * The different kinds of results that a deployment can produce.
 *
 * @beta
 */
export declare enum DeploymentResultType {
    /**
     * One or more futures failed validation.
     */
    VALIDATION_ERROR = "VALIDATION_ERROR",
    /**
     * One or more futures failed the reconciliation process with
     * the previous state of the deployment.
     */
    RECONCILIATION_ERROR = "RECONCILIATION_ERROR",
    /**
     * One or more future's execution failed or timed out.
     */
    EXECUTION_ERROR = "EXECUTION_ERROR",
    /**
     * One or more futures from a previous run failed or timed out.
     */
    PREVIOUS_RUN_ERROR = "PREVIOUS_RUN_ERROR",
    /**
     * The entire deployment was successful.
     */
    SUCCESSFUL_DEPLOYMENT = "SUCCESSFUL_DEPLOYMENT"
}
/**
 * A deployment result where one or more futures failed validation.
 *
 * @beta
 */
export interface ValidationErrorDeploymentResult {
    type: DeploymentResultType.VALIDATION_ERROR;
    /**
     * A map form future id to a list of all of its validation errors.
     */
    errors: {
        [futureId: string]: string[];
    };
}
/**
 * A deployment result where one or more futures failed reconciliation.
 *
 * @beta
 */
export interface ReconciliationErrorDeploymentResult {
    type: DeploymentResultType.RECONCILIATION_ERROR;
    /**
     * A map form future id to a list of all of its reconciliation errors.
     */
    errors: {
        [futureId: string]: string[];
    };
}
/**
 * A deployment result where one or more futures errored during execution or
 * timed out.
 *
 * @beta
 */
export interface ExecutionErrorDeploymentResult {
    type: DeploymentResultType.EXECUTION_ERROR;
    /**
     * A list of all the futures that have started executing but have not
     * finished, neither successfully nor unsuccessfully.
     */
    started: string[];
    /**
     * A list of all the futures that have timed out, including details of the
     * network interaction that timed out.
     */
    timedOut: Array<{
        futureId: string;
        networkInteractionId: number;
    }>;
    /**
     * A list of all the futures that are being Held as determined by the execution
     * strategy, i.e. an off-chain process is not yet complete.
     */
    held: Array<{
        futureId: string;
        heldId: number;
        reason: string;
    }>;
    /**
     * A list of all the future that have failed, including the details of
     * the network interaction that errored.
     */
    failed: Array<{
        futureId: string;
        networkInteractionId: number;
        error: string;
    }>;
    /**
     * A list with the id of all the future that have successfully executed.
     */
    successful: string[];
}
/**
 * A deployment result where one or more futures from a previous run failed or timed out
 * and need their state wiped.
 *
 * @beta
 */
export interface PreviousRunErrorDeploymentResult {
    type: DeploymentResultType.PREVIOUS_RUN_ERROR;
    /**
     * A map from future id to a list of all of its previous run errors.
     */
    errors: {
        [futureId: string]: string[];
    };
}
/**
 * The details of a deployed contract.
 *
 * @beta
 */
export interface DeployedContract {
    id: string;
    contractName: string;
    address: string;
}
/**
 * A deployment result where all of the futures of the module have completed
 * successfully.
 *
 * The deployed contracts returned include the deployed contracts from previous
 * runs.
 *
 * @beta
 */
export interface SuccessfulDeploymentResult {
    type: DeploymentResultType.SUCCESSFUL_DEPLOYMENT;
    /**
     * A map with the contracts returned by the deployed module.
     *
     * The contracts can be the result of a deployment or a contractAt call,
     * in the current run and the previous runs
     */
    contracts: {
        [key: string]: DeployedContract;
    };
}
/**
 * An object containing a map of module ID's to their respective ModuleParameters.
 *
 * @beta
 */
export interface DeploymentParameters {
    [moduleId: string]: ModuleParameters;
}
/**
 * The config options for the deployment strategies.
 *
 * @beta
 */
export interface StrategyConfig {
    basic: Record<PropertyKey, never>;
    create2: {
        salt: string;
    };
}
//# sourceMappingURL=deploy.d.ts.map