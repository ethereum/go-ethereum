export declare class AdjacencyList {
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
    private _list;
    constructor(futureIds: string[]);
    /**
     * Add a dependency from `from` to `to`. If A depends on B
     * then {`from`: A, `to`: B} should be passed.
     */
    addDependency({ from, to }: {
        from: string;
        to: string;
    }): void;
    deleteDependency({ from, to }: {
        from: string;
        to: string;
    }): void;
    /**
     * Get the dependencies, if A depends on B, A's dependencies includes B
     * @param from - the future to get the list of dependencies for
     * @returns - the dependencies
     */
    getDependenciesFor(from: string): Set<string>;
    /**
     * Get the dependents, if A depends on B, B's dependents includes A
     * @param from - the future to get the list of dependents for
     * @returns - the dependents
     */
    getDependentsFor(to: string): string[];
    /**
     * Remove a future, transfering its dependencies to its dependents.
     * @param futureId - The future to eliminate
     */
    eliminate(futureId: string): void;
    static topologicalSort(original: AdjacencyList): string[];
    static clone(original: AdjacencyList): AdjacencyList;
}
//# sourceMappingURL=adjacency-list.d.ts.map