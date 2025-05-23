"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ensureAbsPath = void 0;
const path_1 = require("path");
function ensureAbsPath(path) {
    if ((0, path_1.isAbsolute)(path)) {
        return path;
    }
    return (0, path_1.join)(process.cwd(), path);
}
exports.ensureAbsPath = ensureAbsPath;
//# sourceMappingURL=ensureAbsPath.js.map