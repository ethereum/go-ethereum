import { ContractCallFuture, EncodeFunctionCallFuture, StaticCallFuture } from "../../../types/module";
import { CallExecutionState, EncodeFunctionCallExecutionState, StaticCallExecutionState } from "../../execution/types/execution-state";
import { ReconciliationContext, ReconciliationFutureResultFailure } from "../types";
export declare function reconcileFunctionName(future: ContractCallFuture<string, string> | StaticCallFuture<string, string> | EncodeFunctionCallFuture<string, string>, exState: CallExecutionState | StaticCallExecutionState | EncodeFunctionCallExecutionState, _context: ReconciliationContext): ReconciliationFutureResultFailure | undefined;
//# sourceMappingURL=reconcile-function-name.d.ts.map