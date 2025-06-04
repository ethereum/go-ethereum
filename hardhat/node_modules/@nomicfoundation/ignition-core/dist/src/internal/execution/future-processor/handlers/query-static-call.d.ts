import { JsonRpcClient } from "../../jsonrpc-client";
import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState, StaticCallExecutionState } from "../../types/execution-state";
import { StaticCallCompleteMessage } from "../../types/messages";
/**
 * Runs a static call and returns a message indicating its completion.
 *
 * SIDE EFFECTS: This function doesn't have any side effects.
 *
 * @param exState The execution state that requested the static call.
 * @param jsonRpcClient The JSON RPC client to use for the static call.
 * @returns A message indicating the completion of the static call.
 */
export declare function queryStaticCall(exState: DeploymentExecutionState | CallExecutionState | SendDataExecutionState | StaticCallExecutionState, jsonRpcClient: JsonRpcClient): Promise<StaticCallCompleteMessage>;
//# sourceMappingURL=query-static-call.d.ts.map