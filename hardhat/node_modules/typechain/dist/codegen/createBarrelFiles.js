"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createBarrelFiles = void 0;
const lodash_1 = require("lodash");
const path_1 = require("path");
const normalizeName_1 = require("../parser/normalizeName");
const normalizeDirName_1 = require("./normalizeDirName");
/**
 * returns barrel files with reexports for all given paths
 *
 * @see https://github.com/basarat/typescript-book/blob/master/docs/tips/barrel.md
 */
function createBarrelFiles(paths, { typeOnly, postfix = '', moduleSuffix = '' }) {
    const fileReexports = (0, lodash_1.mapValues)((0, lodash_1.groupBy)(paths.map(path_1.posix.parse), (p) => p.dir), (ps) => ps.map((p) => p.name));
    const directoryPaths = Object.keys(fileReexports).filter((path) => path.includes('/'));
    const directoryReexports = (0, lodash_1.mapValues)((0, lodash_1.groupBy)(directoryPaths.map(path_1.posix.parse), (p) => p.dir), (ps) => ps.map((p) => p.base));
    const barrelPaths = new Set(Object.keys(directoryReexports).concat(Object.keys(fileReexports)));
    const newPaths = [];
    for (const directory of barrelPaths) {
        if (!directory)
            continue;
        const path = directory.split('/');
        while (path.length) {
            const dir = path.slice(0, -1).join('/');
            const name = path[path.length - 1];
            if (!(dir in directoryReexports)) {
                directoryReexports[dir] = [name];
                newPaths.push(dir);
            }
            else if (!directoryReexports[dir].find((x) => x === name)) {
                directoryReexports[dir].push(name);
            }
            path.pop();
        }
    }
    return (0, lodash_1.uniq)([...barrelPaths, ...newPaths]).map((path) => {
        const nestedDirs = (directoryReexports[path] || []).sort();
        const namespacesExports = nestedDirs
            .map((p) => {
            const namespaceIdentifier = (0, normalizeDirName_1.normalizeDirName)(p);
            if (typeOnly)
                return [
                    `import type * as ${namespaceIdentifier} from './${p}';`,
                    `export type { ${namespaceIdentifier} };`,
                ].join('\n');
            if (moduleSuffix) {
                return `export * as ${namespaceIdentifier} from './${p}/index${moduleSuffix}';`;
            }
            return `export * as ${namespaceIdentifier} from './${p}';`;
        })
            .join('\n');
        const contracts = (fileReexports[path] || []).sort();
        const namedExports = contracts
            .map((p) => {
            const exportKeyword = typeOnly ? 'export type' : 'export';
            const name = `${(0, normalizeName_1.normalizeName)(p)}${postfix}`;
            // We can't always `export *` because of possible name conflicts.
            // @todo possibly a config option for user to decide?
            return `${exportKeyword} { ${name} } from './${name}${moduleSuffix}';`;
        })
            .join('\n');
        return {
            path: path_1.posix.join(path, 'index.ts'),
            contents: (namespacesExports + '\n' + namedExports).trim(),
        };
    });
}
exports.createBarrelFiles = createBarrelFiles;
//# sourceMappingURL=createBarrelFiles.js.map