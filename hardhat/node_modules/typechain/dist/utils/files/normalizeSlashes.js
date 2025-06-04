"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.normalizeSlashes = void 0;
function normalizeSlashes(path) {
    return process.platform === 'win32' ? path.replace(/\\/g, '/') : path;
}
exports.normalizeSlashes = normalizeSlashes;
//# sourceMappingURL=normalizeSlashes.js.map