"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.lowestCommonPath = void 0;
function lowestCommonPath(paths) {
    const pathParts = paths.map((path) => path.split(/[\\/]/));
    const commonParts = [];
    const maxParts = Math.min.apply(null, pathParts.map((p) => p.length));
    for (let i = 0; i < maxParts; i++) {
        const part = pathParts[0][i];
        if (pathParts.slice(1).every((otherPath) => otherPath[i] === part)) {
            commonParts.push(part);
        }
        else {
            break;
        }
    }
    return commonParts.join('/');
}
exports.lowestCommonPath = lowestCommonPath;
//# sourceMappingURL=lowestCommonPath.js.map