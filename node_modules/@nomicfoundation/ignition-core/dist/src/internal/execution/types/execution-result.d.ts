import { SolidityParameterType } from "../../../types/module";
import { FailedEvmExecutionResult } from "./evm-execution";
/**
 * The differnt types of result that executing a future can produce.
 */
export declare enum ExecutionResultType {
    SUCCESS = "SUCCESS",
    SIMULATION_ERROR = "SIMULATION_ERROR",
    STRATEGY_SIMULATION_ERROR = "STRATEGY_SIMULATION_ERROR",
    REVERTED_TRANSACTION = "REVERTED_TRANSACTION",
    STATIC_CALL_ERROR = "STATIC_CALL_ERROR",
    STRATEGY_ERROR = "STRATEGY_ERROR",
    STRATEGY_HELD = "STRATEGY_HELD"
}
/**
 * A simulation of an onchain interaction failed, making the execution fail.
 *
 * Note: We don't journal this result.
 */
export interface SimulationErrorExecutionResult {
    type: ExecutionResultType.SIMULATION_ERROR;
    error: FailedEvmExecutionResult;
}
/**
 * A simulation of an onchain interaction seemingly succeded, but the strategy
 * decided that it should be considered a failure.
 *
 * Note: We don't journal this result.
 */
export interface StrategySimulationErrorExecutionResult {
    type: ExecutionResultType.STRATEGY_SIMULATION_ERROR;
    error: string;
}
/**
 * A transaction reverted, making the execution fail.
 */
export interface RevertedTransactionExecutionResult {
    type: ExecutionResultType.REVERTED_TRANSACTION;
    txHash: string;
}
/**
 * A static call failed, making the execution fail.
 */
export interface FailedStaticCallExecutionResult {
    type: ExecutionResultType.STATIC_CALL_ERROR;
    error: FailedEvmExecutionResult;
}
/**
 * The execution strategy returned a strategy-specific error.
 */
export interface StrategyErrorExecutionResult {
    type: ExecutionResultType.STRATEGY_ERROR;
    error: string;
}
/**
 * The execution strategy returned a strategy-specific hold e.g.
 * waiting for off-chain multi-sig confirmations.
 */
export interface StrategyHeldExecutionResult {
    type: ExecutionResultType.STRATEGY_HELD;
    heldId: number;
    reason: string;
}
/**
 * A deployment was successfully executed.
 */
export interface SuccessfulDeploymentExecutionResult {
    type: ExecutionResultType.SUCCESS;
    address: string;
}
/**
 * The different results that executing a future that deploys
 * a contract can produce.
 */
export type DeploymentExecutionResult = SuccessfulDeploymentExecutionResult | SimulationErrorExecutionResult | StrategySimulationErrorExecutionResult | RevertedTransactionExecutionResult | FailedStaticCallExecutionResult | StrategyErrorExecutionResult | StrategyHeldExecutionResult;
/**
 * A call future was successfully executed.
 */
export interface SuccessfulCallExecutionResult {
    type: ExecutionResultType.SUCCESS;
}
/**
 * The different results that executing a call future can produce.
 */
export type CallExecutionResult = SuccessfulCallExecutionResult | SimulationErrorExecutionResult | StrategySimulationErrorExecutionResult | RevertedTransactionExecutionResult | FailedStaticCallExecutionResult | StrategyErrorExecutionResult | StrategyHeldExecutionResult;
/**
 * A send data future was successfully executed.
 */
export interface SuccessfulSendDataExecutionResult {
    type: ExecutionResultType.SUCCESS;
}
/**
 * The different results that executing a send data future can produce.
 */
export type SendDataExecutionResult = SuccessfulSendDataExecutionResult | SimulationErrorExecutionResult | StrategySimulationErrorExecutionResult | RevertedTransactionExecutionResult | FailedStaticCallExecutionResult | StrategyErrorExecutionResult | StrategyHeldExecutionResult;
/**
 * A static call future was successfully executed.
 */
export interface SuccessfulStaticCallExecutionResult {
    type: ExecutionResultType.SUCCESS;
    value: SolidityParameterType;
}
/**
 * The different results that executing a static call future can produce.
 */
export type StaticCallExecutionResult = SuccessfulStaticCallExecutionResult | FailedStaticCallExecutionResult | StrategyErrorExecutionResult | StrategyHeldExecutionResult;
//# sourceMappingURL=execution-result.d.ts.map