"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getRequireCachedFiles = void 0;
function getRequireCachedFiles() {
    return Object.keys(require.cache).filter((p) => !p.startsWith("internal") && (p.endsWith(".js") || p.endsWith(".ts")));
}
exports.getRequireCachedFiles = getRequireCachedFiles;
//# sourceMappingURL=platform.js.map