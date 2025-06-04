import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState, StaticCallExecutionState } from "../../execution/types/execution-state";
import { Transaction } from "../../execution/types/jsonrpc";
export declare function findTransactionBy(executionState: DeploymentExecutionState | CallExecutionState | StaticCallExecutionState | SendDataExecutionState, networkInteractionId: number, hash: string): Transaction;
//# sourceMappingURL=find-transaction-by.d.ts.map