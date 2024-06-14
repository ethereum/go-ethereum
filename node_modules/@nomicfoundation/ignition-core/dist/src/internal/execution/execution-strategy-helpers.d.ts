/**
 * This is a file containing the different helpers that an execution strategy
 * implementation can use.
 * @file
 */
import { SolidityParameterType } from "../../types/module";
import { InvalidResultError, RevertWithCustomError, RevertWithInvalidData, SuccessfulEvmExecutionResult } from "./types/evm-execution";
import { FailedStaticCallExecutionResult, SimulationErrorExecutionResult } from "./types/execution-result";
import { StaticCallExecutionState } from "./types/execution-state";
import { OnchainInteractionRequest, OnchainInteractionResponse, SimulationSuccessSignal, StaticCallRequest, StaticCallResponse, SuccessfulTransaction } from "./types/execution-strategy";
/**
 * Returns true if the given response is an onchain interaction response.
 */
export declare function isOnchainInteractionResponse(response: StaticCallResponse | OnchainInteractionResponse): response is OnchainInteractionResponse;
/**
 * A function that decodes custom errors.
 *
 * @param returnData The return data of an evm execution, as returned by the JSON-RPC.
 * @returns `RevertWithCustomError` if a custom error was successfully decoded. `RevertWithInvalidData`
 *  if the custom error is recognized but the return data was invalid. `undefined` no custom error was
 *  recognized.
 */
export type DecodeCustomError = (returnData: string) => RevertWithCustomError | RevertWithInvalidData | undefined;
/**
 * A function that decodes the succesful result of an evm execution.
 * @param returnData The return data of an evm execution, as returned by the JSON-RPC.
 * @returns `InvalidResultError` if the result is invalid wrt to the contract's ABI.
 *  `SuccessfulEvmExecutionResult` if the result can be decoded.
 */
export type DecodeSuccessfulExecutionResult = (returnData: string) => InvalidResultError | SuccessfulEvmExecutionResult;
/**
 * Executes an onchain interaction request.
 *
 * @param executionStateId The id of the execution state that lead to the request.
 * @param onchainInteractionRequest  The request to execute.
 * @param decodeSuccessfulSimulationResult A function to decode the results of a
 *  simulation. Can be `undefined` if the request is not related to a contract or
 *  if we want to accept any result (e.g. in a deployment).
 * @param decodeCustomError A function to decode custom errors. Can be `undefined`
 *  if the request is not related to a contract whose custom errors we know how to
 *  decode.
 * @returns The successful transaction response or a simulation error.
 */
export declare function executeOnchainInteractionRequest(executionStateId: string, onchainInteractionRequest: OnchainInteractionRequest, decodeSuccessfulSimulationResult?: DecodeSuccessfulExecutionResult, decodeCustomError?: DecodeCustomError): AsyncGenerator<OnchainInteractionRequest | SimulationSuccessSignal, SuccessfulTransaction | SimulationErrorExecutionResult, OnchainInteractionResponse | StaticCallResponse>;
/**
 * Executes an static call request.
 *
 * @param staticCallRequest  The static call request to execute.
 * @param decodeSuccessfulResult A function to decode the results of a simulation.
 * @param decodeCustomError A function to decode custom errors.
 * @returns The successful evm execution result, or a failed static call result.
 */
export declare function executeStaticCallRequest(staticCallRequest: StaticCallRequest, decodeSuccessfulResult: DecodeSuccessfulExecutionResult, decodeCustomError: DecodeCustomError): AsyncGenerator<StaticCallRequest, SuccessfulEvmExecutionResult | FailedStaticCallExecutionResult, StaticCallResponse>;
/**
 * Returns the right value from the last static call result that should be used
 * as the whole result of the static call execution state.
 *
 * @param _exState The execution state
 * @param lastStaticCallResult The result of the last network interaction.
 * @returns The value that should be used as the result of the static call execution state.
 */
export declare function getStaticCallExecutionStateResultValue(exState: StaticCallExecutionState, lastStaticCallResult: SuccessfulEvmExecutionResult): SolidityParameterType;
export * from "./abi";
//# sourceMappingURL=execution-strategy-helpers.d.ts.map