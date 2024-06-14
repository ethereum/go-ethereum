"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.shortenFullJsonFilePath = void 0;
const path_1 = require("path");
/**
 * Transforms all paths matching `ContractName(\.sol)?/ContractName.ext`
 */
function shortenFullJsonFilePath(path, allPaths) {
    const { name, dir, base } = path_1.posix.parse(path);
    if (allPaths.filter((p) => p.startsWith(dir + '/')).length > 1) {
        return path;
    }
    if (dir.endsWith(`/${name}.sol`) || dir.endsWith(`/${name}`)) {
        return dir.split('/').slice(0, -1).join('/') + '/' + base;
    }
    return path;
}
exports.shortenFullJsonFilePath = shortenFullJsonFilePath;
//# sourceMappingURL=shortenFullJsonFilePath.js.map