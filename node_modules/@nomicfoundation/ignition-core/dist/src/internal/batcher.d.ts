import { IgnitionModule } from "../types/module";
import { DeploymentState } from "./execution/types/deployment-state";
import { AdjacencyList } from "./utils/adjacency-list";
declare enum VisitStatus {
    UNVISITED = 0,
    VISITED = 1
}
interface VisitStatusMap {
    [key: string]: VisitStatus;
}
export declare class Batcher {
    static batch(module: IgnitionModule, deploymentState: DeploymentState): string[][];
    private static _initializeBatchStateFrom;
    private static _intializeVisitStateFrom;
    static _eleminateAlreadyVisitedFutures({ adjacencyList, visitState, }: {
        adjacencyList: AdjacencyList;
        visitState: VisitStatusMap;
    }): void;
    private static _allVisited;
    private static _markAsVisited;
    private static _resolveNextBatch;
    private static _allDependenciesVisited;
}
export {};
//# sourceMappingURL=batcher.d.ts.map