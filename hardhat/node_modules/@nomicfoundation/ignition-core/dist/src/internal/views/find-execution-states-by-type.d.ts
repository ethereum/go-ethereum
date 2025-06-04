import { MapExStateTypeToExState } from "../execution/type-helpers";
import { DeploymentState } from "../execution/types/deployment-state";
import { ExecutionStateType } from "../execution/types/execution-state";
export declare function findExecutionStatesByType<ExStateT extends ExecutionStateType>(exStateType: ExStateT, deployment: DeploymentState): Array<MapExStateTypeToExState<ExStateT>>;
//# sourceMappingURL=find-execution-states-by-type.d.ts.map