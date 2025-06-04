"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.AdjacencyListConverter = void 0;
const type_guards_1 = require("../../type-guards");
const adjacency_list_1 = require("./adjacency-list");
const get_futures_from_module_1 = require("./get-futures-from-module");
class AdjacencyListConverter {
    static buildAdjacencyListFromFutures(futures) {
        const dependencyGraph = new adjacency_list_1.AdjacencyList(futures.map((f) => f.id));
        for (const future of futures) {
            for (const dependency of future.dependencies) {
                if ((0, type_guards_1.isFuture)(dependency)) {
                    // We only add Futures to the dependency graph, modules are handled
                    // in the method call below, by adding their futures to the graph.
                    dependencyGraph.addDependency({ from: future.id, to: dependency.id });
                }
                this._optionallyAddDependenciesFromSubmodules(dependencyGraph, future, dependency);
            }
        }
        return dependencyGraph;
    }
    /**
     * The famed Malaga rule, if a future's dependency is in a submodule,
     * then that future should not be executed until all futures in the
     * submodule and its submodules (recursive) have been run.
     */
    static _optionallyAddDependenciesFromSubmodules(dependencyGraph, future, dependency) {
        // we only need to worry about this case if the dependency is a future
        if ((0, type_guards_1.isFuture)(dependency) && future.module === dependency.module) {
            return;
        }
        const futures = (0, get_futures_from_module_1.getFuturesFromModule)((0, type_guards_1.isFuture)(dependency) ? dependency.module : dependency);
        for (const moduleDep of futures) {
            dependencyGraph.addDependency({
                from: future.id,
                to: moduleDep.id,
            });
        }
    }
}
exports.AdjacencyListConverter = AdjacencyListConverter;
//# sourceMappingURL=adjacency-list-converter.js.map