import { Artifact } from "../../../types/artifact";
import { DeploymentLoader } from "../../deployment-loader/types";
import { JsonRpcClient } from "../jsonrpc-client";
import { CallExecutionResult, DeploymentExecutionResult, RevertedTransactionExecutionResult, SendDataExecutionResult, StaticCallExecutionResult } from "./execution-result";
import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState, StaticCallExecutionState } from "./execution-state";
import { RawStaticCallResult, Transaction, TransactionReceiptStatus } from "./jsonrpc";
import { OnchainInteraction, StaticCall } from "./network-interaction";
/**
 * A function that loads an artifact from the deployment's artifacts store.
 */
export type LoadArtifactFunction = (artifactId: string) => Promise<Artifact>;
/**
 * A request to perform an onchain interaction. This leads to the execution engine
 * sending a transaction to the network, and potentially replacing to with others
 * to handle error conditions.
 */
export type OnchainInteractionRequest = Omit<OnchainInteraction, "transactions" | "nonce" | "shouldBeResent">;
/**
 * The different responses that the execution engine can produce when asked to
 * perform an onchain interaction.
 */
export type OnchainInteractionResponse = SuccessfulTransaction | SimulationResult;
/**
 * The different types of response that the execution engine can give when
 * asked to perform an onchain interaction.
 */
export declare enum OnchainInteractionResponseType {
    SUCCESSFUL_TRANSACTION = "SUCCESSFUL_TRANSACTION",
    SIMULATION_RESULT = "SIMULATION_RESULT"
}
/**
 * A response to an onchain interaction request that indicates that a transaction
 * was sent to the network and was successful.
 *
 * This response is not used for reverted transactions. It's only used for
 * transactions that were successful and that reached the desired amount of confirmations.
 */
export interface SuccessfulTransaction {
    type: OnchainInteractionResponseType.SUCCESSFUL_TRANSACTION;
    transaction: Required<Transaction> & {
        receipt: {
            status: TransactionReceiptStatus.SUCCESS;
        };
    };
}
/**
 * A response to an onchain interaction request that includes the results of
 * simulating the onchain interaction by running an `eth_call`.
 */
export interface SimulationResult {
    type: OnchainInteractionResponseType.SIMULATION_RESULT;
    result: RawStaticCallResult;
}
/**
 * The type of a SimulationSuccessSignal
 */
export declare const SIMULATION_SUCCESS_SIGNAL_TYPE = "SIMULATION_SUCCESS_SIGNAL";
/**
 * An type that signals to the execution engine that the simulation of an
 * onchain interaction was successful, so the execution engine can proceed
 * to send a transaction.
 */
export interface SimulationSuccessSignal {
    type: typeof SIMULATION_SUCCESS_SIGNAL_TYPE;
}
/**
 * A request to perform a static call. This leads to the execution engine
 * seding an `eth_call` to the network.
 */
export type StaticCallRequest = Omit<StaticCall, "result" | "from"> & {
    from?: string;
};
/**
 * The response of a static call request.
 */
export type StaticCallResponse = RawStaticCallResult;
export type DeploymentStrategyGenerator = AsyncGenerator<OnchainInteractionRequest | SimulationSuccessSignal | StaticCallRequest, Exclude<DeploymentExecutionResult, RevertedTransactionExecutionResult>, OnchainInteractionResponse | StaticCallResponse>;
export type CallStrategyGenerator = AsyncGenerator<OnchainInteractionRequest | SimulationSuccessSignal | StaticCallRequest, Exclude<CallExecutionResult, RevertedTransactionExecutionResult>, OnchainInteractionResponse | StaticCallResponse>;
export type SendDataStrategyGenerator = AsyncGenerator<OnchainInteractionRequest | SimulationSuccessSignal | StaticCallRequest, Exclude<SendDataExecutionResult, RevertedTransactionExecutionResult>, OnchainInteractionResponse | StaticCallResponse>;
export type StaticCallStrategyGenerator = AsyncGenerator<StaticCallRequest, StaticCallExecutionResult, StaticCallResponse>;
/**
 * Each execution strategy defines how each type of execution state is executed.
 *
 * This is performed by returning a generator that can request to the execution
 * engine to perform onchain interactions and static calls.
 *
 * Each request is yielded by the strategy, and the execution engine will perform the
 * necessary actions, and then resume the generator with a response to that request.
 * If a transaction reverts, the execution engine will not resume the generator, but
 * directly create a `RevertedTransactionExecutionResult`.
 *
 * The execution strategy is responsible for interpreting the response and deciding
 * if they are successful or not.
 *
 * If they are not successful, the execution strategy should return an error, interrupting
 * the execution.
 *
 * A successful response can result in a new request being yielded.
 *
 * When the execution strategy is done, it should return a result.
 *
 * If the execution strategy considers a response to have failed for reasons other than
 * a failed contract execution or simulation, it can return a custom error using
 * `StrategySimulationErrorExecutionResult` and `StrategyErrorExecutionResult`.
 *
 * Execution strategies are also use to resume a partial execution, hence, they must be
 * prepared to receive the results of their requests immediately. This affects to the
 * process of requestsing `OnchainInteraction`s, which differs whether if the request
 * has already being resolved or not. See below for more details.
 *
 * There are two types of request, which follow a different protocol:
 *
 *   - `OnchainInteractionRequest`: This request is used to perform an onchain
 *    interaction.
 *
 *    If this `OnchainInteractionRequest` hasn't been executed yet, the execution engine
 *    will simulate the onchain interaction first, and respond with a `SimulationResult`.
 *    The execution strategy can then decide if it wants to proceed with the onchain
 *    interaction or not.
 *
 *    If this `OnchainInteractionRequest` was already executed (e.g. we are resuming an
 *    existing deployment), the strategy will immedately get a `SuccessfulTransaction` as
 *    a response.
 *
 *    If the strategy doesn't want to proceed, it should return a `SimulationErrorExecutionResult`
 *    or a `StrategySimulationErrorExecutionResult`.
 *
 *    If an error is returned, the execution will be considered failed, but no failed result
 *    will be stored in the execution journal.
 *
 *    If the execution strategy wants to proceed, it should yield a `SimulationSuccessSignal`.
 *
 *    The execution engine will then send a transaction to the network and wait for it
 *    to get enough confirmations. The execution engine is responsible for making sure the
 *    transaction gets re-sent if needed.
 *
 *    If the transaction was reverted, the execution strategy generator will not be resumed.
 *
 *    If the transaction was successful, the execution engine will resume the generator
 *    with a `SuccessfulTransaction` response. The execution strategy can then decide if
 *    execution was successful based on the transaction.
 *
 *    If the execution should be considered failed, the execution strategy should return
 *    a `StrategyErrorExecutionResult`.
 *
 *    If this was the latest request, the execution strategy should return a successful result
 *    or a `StrategyErrorExecutionResult`.
 *
 *  - `StaticCallRequest`: This request is used to perform a static call.
 *
 *    The execution engine will perform the static call and respond with a `StaticCallResponse`,
 *    which includes the execution raw result.
 *
 *    The execution strategy can then decode the result and decide if the execution was
 *    successful or not.
 *
 *    If the execution should be considered failed, the execution strategy should return
 *    either a `FailedStaticCallExecutionResult` or a `StrategyErrorExecutionResult`.
 *
 *    If this was the latest request, the execution strategy should return a successful result
 *    or a `StrategyErrorExecutionResult`.
 */
export interface ExecutionStrategy {
    /**
     * The name of the strategy as will be recorded in the journal.
     */
    name: string;
    /**
     * The configuration options for the strategy.
     */
    config: Record<PropertyKey, string | number> | Record<PropertyKey, never>;
    init: (deploymentLoader: DeploymentLoader, jsonRpcClient: JsonRpcClient) => Promise<void>;
    /**
     * Executes a deployment execution state.
     */
    executeDeployment: (executionState: DeploymentExecutionState) => DeploymentStrategyGenerator;
    /**
     * Executes a deployment execution state.
     */
    executeCall: (executionState: CallExecutionState) => CallStrategyGenerator;
    /**
     * Executes a deployment execution state.
     */
    executeSendData: (executionState: SendDataExecutionState) => SendDataStrategyGenerator;
    /**
     * Executes a deployment execution state.
     */
    executeStaticCall: (executionState: StaticCallExecutionState) => AsyncGenerator<StaticCallRequest, StaticCallExecutionResult, StaticCallResponse>;
}
//# sourceMappingURL=execution-strategy.d.ts.map