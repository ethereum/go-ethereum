"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.AdjacencyList = void 0;
const assertions_1 = require("./assertions");
class AdjacencyList {
    /**
     * A mapping from futures to each futures dependencies.
     *
     * Example:
     *     A
     *    ^ ^
     *    | |
     *    B C
     * Gives a mapping of {A: [], B: [A], C:[A]}
     *
     */
    _list = new Map();
    constructor(futureIds) {
        for (const futureId of futureIds) {
            this._list.set(futureId, new Set());
        }
    }
    /**
     * Add a dependency from `from` to `to`. If A depends on B
     * then {`from`: A, `to`: B} should be passed.
     */
    addDependency({ from, to }) {
        const toSet = this._list.get(from) ?? new Set();
        toSet.add(to);
        this._list.set(from, toSet);
    }
    deleteDependency({ from, to }) {
        const toSet = this._list.get(from) ?? new Set();
        toSet.delete(to);
        this._list.set(from, toSet);
    }
    /**
     * Get the dependencies, if A depends on B, A's dependencies includes B
     * @param from - the future to get the list of dependencies for
     * @returns - the dependencies
     */
    getDependenciesFor(from) {
        return this._list.get(from) ?? new Set();
    }
    /**
     * Get the dependents, if A depends on B, B's dependents includes A
     * @param from - the future to get the list of dependents for
     * @returns - the dependents
     */
    getDependentsFor(to) {
        return [...this._list.entries()]
            .filter(([_from, toSet]) => toSet.has(to))
            .map(([from]) => from);
    }
    /**
     * Remove a future, transfering its dependencies to its dependents.
     * @param futureId - The future to eliminate
     */
    eliminate(futureId) {
        const dependents = this.getDependentsFor(futureId);
        const dependencies = this.getDependenciesFor(futureId);
        this._list.delete(futureId);
        for (const dependent of dependents) {
            const toSet = this._list.get(dependent);
            (0, assertions_1.assertIgnitionInvariant)(toSet !== undefined, "Dependency sets should be defined");
            const setWithoutFuture = new Set([...toSet].filter((n) => n !== futureId));
            const updatedSet = new Set([
                ...setWithoutFuture,
                ...dependencies,
            ]);
            this._list.set(dependent, updatedSet);
        }
    }
    static topologicalSort(original) {
        const newList = this.clone(original);
        if (newList._list.size === 0) {
            return [];
        }
        // Empty list that will contain the sorted elements
        let l = [];
        // set of all nodes with no dependents
        const s = new Set([...newList._list.keys()].filter((fid) => newList.getDependentsFor(fid).length === 0));
        while (s.size !== 0) {
            const n = [...s].pop();
            s.delete(n);
            l = [...l, n];
            for (const m of newList.getDependenciesFor(n)) {
                newList.deleteDependency({ from: n, to: m });
                if (newList.getDependentsFor(m).length === 0) {
                    s.add(m);
                }
            }
        }
        return l;
    }
    static clone(original) {
        const newList = new AdjacencyList([
            ...original._list.keys(),
        ]);
        for (const [from, toSet] of original._list.entries()) {
            newList._list.set(from, new Set(toSet));
        }
        return newList;
    }
}
exports.AdjacencyList = AdjacencyList;
//# sourceMappingURL=adjacency-list.js.map