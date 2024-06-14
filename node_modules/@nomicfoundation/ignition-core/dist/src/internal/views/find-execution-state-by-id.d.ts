import { MapExStateTypeToExState } from "../execution/type-helpers";
import { DeploymentState } from "../execution/types/deployment-state";
import { ExecutionSateType } from "../execution/types/execution-state";
export declare function findExecutionStateById<ExStateT extends ExecutionSateType>(exStateType: ExStateT, deployment: DeploymentState, futureId: string): MapExStateTypeToExState<ExStateT>;
//# sourceMappingURL=find-execution-state-by-id.d.ts.map