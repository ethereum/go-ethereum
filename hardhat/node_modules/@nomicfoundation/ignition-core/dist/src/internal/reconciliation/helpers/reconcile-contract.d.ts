import { ContractCallFuture, StaticCallFuture } from "../../../types/module";
import { CallExecutionState, StaticCallExecutionState } from "../../execution/types/execution-state";
import { ReconciliationContext, ReconciliationFutureResultFailure } from "../types";
export declare function reconcileContract(future: ContractCallFuture<string, string> | StaticCallFuture<string, string>, exState: CallExecutionState | StaticCallExecutionState, context: ReconciliationContext): ReconciliationFutureResultFailure | undefined;
//# sourceMappingURL=reconcile-contract.d.ts.map