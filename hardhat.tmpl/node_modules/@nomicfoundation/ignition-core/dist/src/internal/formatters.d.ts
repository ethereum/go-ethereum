import { SolidityParameterType } from "../types/module";
import { EvmTuple, FailedEvmExecutionResult } from "./execution/types/evm-execution";
import { FailedStaticCallExecutionResult, RevertedTransactionExecutionResult, SimulationErrorExecutionResult, StrategyErrorExecutionResult, StrategySimulationErrorExecutionResult } from "./execution/types/execution-result";
/**
 * Formats an execution error result into a human-readable string.
 */
export declare function formatExecutionError(result: SimulationErrorExecutionResult | StrategySimulationErrorExecutionResult | RevertedTransactionExecutionResult | FailedStaticCallExecutionResult | StrategyErrorExecutionResult): string;
/**
 * Formats a failed EVM execution result into a human-readable string.
 */
export declare function formatFailedEvmExecutionResult(result: FailedEvmExecutionResult): string;
/**
 * Formats a custom error into a human-readable string.
 */
export declare function formatCustomError(errorName: string, args: EvmTuple): string;
/**
 * Formats a Solidity parameter into a human-readable string.
 *
 * @beta
 */
export declare function formatSolidityParameter(param: SolidityParameterType): string;
//# sourceMappingURL=formatters.d.ts.map