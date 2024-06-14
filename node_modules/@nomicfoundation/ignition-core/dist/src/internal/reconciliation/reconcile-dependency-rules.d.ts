import { Future } from "../../types/module";
import { DeploymentState } from "../execution/types/deployment-state";
import { ExecutionState } from "../execution/types/execution-state";
import { ReconciliationFutureResult } from "./types";
export declare function reconcileDependencyRules(future: Future, executionState: ExecutionState, context: {
    deploymentState: DeploymentState;
}): ReconciliationFutureResult;
//# sourceMappingURL=reconcile-dependency-rules.d.ts.map