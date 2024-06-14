"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.tryRequire = void 0;
const debug_1 = require("./debug");
function tryRequire(name) {
    try {
        let path;
        try {
            path = require.resolve(name, { paths: [process.cwd()] });
        }
        catch (_a) {
            path = require.resolve(name);
        }
        const module = { module: require(path), name, path };
        (0, debug_1.debug)('Load successfully: ', name);
        return module;
    }
    catch (err) {
        if (err instanceof Error && err.message.startsWith(`Cannot find module '${name}'`)) {
            // this error is expected
        }
        else {
            throw err;
        }
        (0, debug_1.debug)("Couldn't load: ", name);
    }
}
exports.tryRequire = tryRequire;
//# sourceMappingURL=modules.js.map