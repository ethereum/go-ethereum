import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState, StaticCallExecutionState } from "../../execution/types/execution-state";
import { OnchainInteraction } from "../../execution/types/network-interaction";
export declare function findOnchainInteractionBy(executionState: DeploymentExecutionState | CallExecutionState | StaticCallExecutionState | SendDataExecutionState, networkInteractionId: number): OnchainInteraction;
//# sourceMappingURL=find-onchain-interaction-by.d.ts.map