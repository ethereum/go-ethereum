import { Future } from "../../types/module";
import { AdjacencyList } from "./adjacency-list";
export declare class AdjacencyListConverter {
    static buildAdjacencyListFromFutures(futures: Future[]): AdjacencyList;
    /**
     * The famed Malaga rule, if a future's dependency is in a submodule,
     * then that future should not be executed until all futures in the
     * submodule and its submodules (recursive) have been run.
     */
    private static _optionallyAddDependenciesFromSubmodules;
}
//# sourceMappingURL=adjacency-list-converter.d.ts.map