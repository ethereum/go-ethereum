"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DependencyGraph = void 0;
const errors_1 = require("../core/errors");
const errors_list_1 = require("../core/errors-list");
class DependencyGraph {
    static async createFromResolvedFiles(resolver, resolvedFiles) {
        const graph = new DependencyGraph();
        // TODO refactor this to make the results deterministic
        await Promise.all(resolvedFiles.map((resolvedFile) => graph._addDependenciesFrom(resolver, resolvedFile)));
        return graph;
    }
    constructor() {
        this._resolvedFiles = new Map();
        this._dependenciesPerFile = new Map();
        // map absolute paths to source names
        this._visitedFiles = new Map();
    }
    getResolvedFiles() {
        return Array.from(this._resolvedFiles.values());
    }
    has(file) {
        return this._resolvedFiles.has(file.sourceName);
    }
    isEmpty() {
        return this._resolvedFiles.size === 0;
    }
    entries() {
        return Array.from(this._dependenciesPerFile.entries()).map(([key, value]) => [this._resolvedFiles.get(key), value]);
    }
    getDependencies(file) {
        const dependencies = this._dependenciesPerFile.get(file.sourceName) ?? new Set();
        return [...dependencies];
    }
    getTransitiveDependencies(file) {
        const visited = new Set();
        const transitiveDependencies = this._getTransitiveDependencies(file, visited, []);
        return [...transitiveDependencies];
    }
    getConnectedComponents() {
        const undirectedGraph = {};
        for (const [sourceName, dependencies,] of this._dependenciesPerFile.entries()) {
            undirectedGraph[sourceName] = undirectedGraph[sourceName] ?? new Set();
            for (const dependency of dependencies) {
                undirectedGraph[dependency.sourceName] =
                    undirectedGraph[dependency.sourceName] ?? new Set();
                undirectedGraph[sourceName].add(dependency.sourceName);
                undirectedGraph[dependency.sourceName].add(sourceName);
            }
        }
        const components = [];
        const visited = new Set();
        for (const node of Object.keys(undirectedGraph)) {
            if (visited.has(node)) {
                continue;
            }
            visited.add(node);
            const component = new Set([node]);
            const stack = [...undirectedGraph[node]];
            while (stack.length > 0) {
                const newNode = stack.pop();
                if (visited.has(newNode)) {
                    continue;
                }
                visited.add(newNode);
                component.add(newNode);
                [...undirectedGraph[newNode]].forEach((adjacent) => {
                    if (!visited.has(adjacent)) {
                        stack.push(adjacent);
                    }
                });
            }
            components.push(component);
        }
        const connectedComponents = [];
        for (const component of components) {
            const dependencyGraph = new DependencyGraph();
            for (const sourceName of component) {
                const file = this._resolvedFiles.get(sourceName);
                const dependencies = this._dependenciesPerFile.get(sourceName);
                dependencyGraph._resolvedFiles.set(sourceName, file);
                dependencyGraph._dependenciesPerFile.set(sourceName, dependencies);
            }
            connectedComponents.push(dependencyGraph);
        }
        return connectedComponents;
    }
    _getTransitiveDependencies(file, visited, path) {
        if (visited.has(file)) {
            return new Set();
        }
        visited.add(file);
        const directDependencies = this.getDependencies(file).map((dependency) => ({
            dependency,
            path,
        }));
        const transitiveDependencies = new Set(directDependencies);
        for (const { dependency } of transitiveDependencies) {
            this._getTransitiveDependencies(dependency, visited, path.concat(dependency)).forEach((x) => transitiveDependencies.add(x));
        }
        return transitiveDependencies;
    }
    async _addDependenciesFrom(resolver, file) {
        const sourceName = this._visitedFiles.get(file.absolutePath);
        if (sourceName !== undefined) {
            if (sourceName !== file.sourceName) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.RESOLVER.AMBIGUOUS_SOURCE_NAMES, {
                    sourcenames: `'${sourceName}' and '${file.sourceName}'`,
                    file: file.absolutePath,
                });
            }
            return;
        }
        this._visitedFiles.set(file.absolutePath, file.sourceName);
        const dependencies = new Set();
        this._resolvedFiles.set(file.sourceName, file);
        this._dependenciesPerFile.set(file.sourceName, dependencies);
        // TODO refactor this to make the results deterministic
        await Promise.all(file.content.imports.map(async (imp) => {
            const dependency = await resolver.resolveImport(file, imp);
            dependencies.add(dependency);
            await this._addDependenciesFrom(resolver, dependency);
        }));
    }
}
exports.DependencyGraph = DependencyGraph;
//# sourceMappingURL=dependencyGraph.js.map