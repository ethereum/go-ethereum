import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState, StaticCallExecutionState } from "../../execution/types/execution-state";
import { StaticCall } from "../../execution/types/network-interaction";
export declare function findStaticCallBy(executionState: DeploymentExecutionState | CallExecutionState | StaticCallExecutionState | SendDataExecutionState, networkInteractionId: number): StaticCall;
//# sourceMappingURL=find-static-call-by.d.ts.map