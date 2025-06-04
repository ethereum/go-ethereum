import { IgnitionError } from "../../errors";
import { Artifact } from "../../types/artifact";
import { ArgumentType, SolidityParameterType } from "../../types/module";
import { EvmValue, FailedEvmExecutionResult, InvalidResultError, RevertWithCustomError, RevertWithInvalidData, SuccessfulEvmExecutionResult } from "./types/evm-execution";
import { TransactionReceipt } from "./types/jsonrpc";
/**
 * Encodes the constructor arguments for a deployment.
 */
export declare function encodeDeploymentArguments(artifact: Artifact, args: SolidityParameterType[]): string;
/**
 * Links the libraries in the artifact's deployment bytecode, encodes the constructor
 * arguments and returns the result, which can be used as the `data` field of a
 * deployment.
 */
export declare function encodeArtifactDeploymentData(artifact: Artifact, args: SolidityParameterType[], libraries: {
    [libraryName: string]: string;
}): string;
/**
 * Encodes a function call for the given artifact and function name.
 */
export declare function encodeArtifactFunctionCall(artifact: Artifact, functionName: string, args: SolidityParameterType[]): string;
/**
 * Decodes a custom error from the given return data, if it's recognized
 * as one of the artifact's custom errors.
 */
export declare function decodeArtifactCustomError(artifact: Artifact, returnData: string): RevertWithCustomError | RevertWithInvalidData | undefined;
/**
 * Decode the result of a successful function call.
 */
export declare function decodeArtifactFunctionCallResult(artifact: Artifact, functionName: string, returnData: string): InvalidResultError | SuccessfulEvmExecutionResult;
/**
 * Validate that the given args length matches the artifact's abi's args length.
 *
 * @param artifact - the artifact for the contract being validated
 * @param contractName - the name of the contract for error messages
 * @param args - the args to validate against
 */
export declare function validateContractConstructorArgsLength(artifact: Artifact, contractName: string, args: ArgumentType[]): IgnitionError[];
/**
 * Validates that a function is valid for the given artifact. That means:
 *  - It's a valid function name
 *    - The function name exists in the artifact's ABI
 *    - If the function is not overlaoded, its bare name is used.
 *    - If the function is overloaded, the function name is includes the argument types
 *      in parentheses.
 * - The function has the correct number of arguments
 *
 * Optionally checks further static call constraints:
 * - The function is has a pure or view state mutability
 */
export declare function validateArtifactFunction(artifact: Artifact, contractName: string, functionName: string, args: ArgumentType[], isStaticCall: boolean): IgnitionError[];
/**
 * Validates that a function name is valid for the given artifact. That means:
 *  - It's a valid function name
 *  - The function name exists in the artifact's ABI
 *  - If the function is not overlaoded, its bare name is used.
 *  - If the function is overloaded, the function name is includes the argument types
 *    in parentheses.
 */
export declare function validateArtifactFunctionName(artifact: Artifact, functionName: string): IgnitionError[];
/**
 * Validates that the event exists in the artifact, it's name is valid, handles overloads
 * correctly, and that the arugment exists in the event.
 *
 * @param emitterArtifact The artifact of the contract emitting the event.
 * @param eventName The name of the event.
 * @param argument The argument name or index.
 */
export declare function validateArtifactEventArgumentParams(emitterArtifact: Artifact, eventName: string, argument: string | number): IgnitionError[];
/**
 * Returns the value of an argument in an event emitted by the contract
 * at emitterAddress with a certain artifact.
 *
 * @param receipt The receipt of the transaction that emitted the event.
 * @param emitterArtifact The artifact of the contract emitting the event.
 * @param emitterAddress The address of the contract emitting the event.
 * @param eventName The name of the event. It MUST be validated first.
 * @param eventIndex The index of the event, in case there are multiple events emitted with the same name
 * @param argument The name or index of the argument to extract from the event.
 * @returns The EvmValue of the argument.
 */
export declare function getEventArgumentFromReceipt(receipt: TransactionReceipt, emitterArtifact: Artifact, emitterAddress: string, eventName: string, eventIndex: number, nameOrIndex: string | number): EvmValue;
/**
 * Decodes an error from a failed evm execution.
 *
 * @param returnData The data, as returned by the JSON-RPC.
 * @param customErrorReported A value indicating if the JSON-RPC error
 *  reported that it was due to a custom error.
 * @param decodeCustomError A function that decodes custom errors, returning
 *  `RevertWithCustomError` if succesfully decoded, `RevertWithInvalidData`
 *  if a custom error was recognized but couldn't be decoded, and `undefined`
 *  it it wasn't recognized.
 * @returns A `FailedEvmExecutionResult` with the decoded error.
 */
export declare function decodeError(returnData: string, customErrorReported: boolean, decodeCustomError?: (returnData: string) => RevertWithCustomError | RevertWithInvalidData | undefined): FailedEvmExecutionResult;
/**
 * Validates the param type of a static call return value, throwing a validation error if it's not found.
 */
export declare function validateFunctionArgumentParamType(contractName: string, functionName: string, artifact: Artifact, argument: string | number): IgnitionError[];
//# sourceMappingURL=abi.d.ts.map