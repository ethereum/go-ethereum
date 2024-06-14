"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Batcher = void 0;
const execution_state_1 = require("./execution/types/execution-state");
const adjacency_list_converter_1 = require("./utils/adjacency-list-converter");
const get_futures_from_module_1 = require("./utils/get-futures-from-module");
var VisitStatus;
(function (VisitStatus) {
    VisitStatus[VisitStatus["UNVISITED"] = 0] = "UNVISITED";
    VisitStatus[VisitStatus["VISITED"] = 1] = "VISITED";
})(VisitStatus || (VisitStatus = {}));
class Batcher {
    static batch(module, deploymentState) {
        const batchState = this._initializeBatchStateFrom(module, deploymentState);
        const batches = [];
        while (!this._allVisited(batchState)) {
            const nextBatch = this._resolveNextBatch(batchState);
            batches.push(nextBatch);
            this._markAsVisited(batchState, nextBatch);
        }
        return batches;
    }
    static _initializeBatchStateFrom(module, deploymentState) {
        const allFutures = (0, get_futures_from_module_1.getFuturesFromModule)(module);
        const visitState = this._intializeVisitStateFrom(allFutures, deploymentState);
        const adjacencyList = adjacency_list_converter_1.AdjacencyListConverter.buildAdjacencyListFromFutures(allFutures);
        this._eleminateAlreadyVisitedFutures({ adjacencyList, visitState });
        return { adjacencyList, visitState };
    }
    static _intializeVisitStateFrom(futures, deploymentState) {
        return Object.fromEntries(futures.map((f) => {
            const executionState = deploymentState.executionStates[f.id];
            if (executionState === undefined) {
                return [f.id, VisitStatus.UNVISITED];
            }
            switch (executionState.status) {
                case execution_state_1.ExecutionStatus.FAILED:
                case execution_state_1.ExecutionStatus.TIMEOUT:
                case execution_state_1.ExecutionStatus.HELD:
                case execution_state_1.ExecutionStatus.STARTED:
                    return [f.id, VisitStatus.UNVISITED];
                case execution_state_1.ExecutionStatus.SUCCESS:
                    return [f.id, VisitStatus.VISITED];
            }
        }));
    }
    static _eleminateAlreadyVisitedFutures({ adjacencyList, visitState, }) {
        const visitedFutures = Object.entries(visitState)
            .filter(([, vs]) => vs === VisitStatus.VISITED)
            .map(([futureId]) => futureId);
        for (const visitedFuture of visitedFutures) {
            adjacencyList.eliminate(visitedFuture);
        }
    }
    static _allVisited(batchState) {
        return Object.values(batchState.visitState).every((s) => s === VisitStatus.VISITED);
    }
    static _markAsVisited(batchState, nextBatch) {
        for (const futureId of nextBatch) {
            batchState.visitState[futureId] = VisitStatus.VISITED;
        }
    }
    static _resolveNextBatch(batchState) {
        const allUnvisited = Object.entries(batchState.visitState)
            .filter(([, state]) => state === VisitStatus.UNVISITED)
            .map(([id]) => id);
        const allUnvisitedWhereDepsVisited = allUnvisited.filter((futureId) => this._allDependenciesVisited(futureId, batchState));
        return allUnvisitedWhereDepsVisited.sort();
    }
    static _allDependenciesVisited(futureId, batchState) {
        const dependencies = batchState.adjacencyList.getDependenciesFor(futureId);
        return [...dependencies].every((depId) => batchState.visitState[depId] === VisitStatus.VISITED);
    }
}
exports.Batcher = Batcher;
//# sourceMappingURL=batcher.js.map