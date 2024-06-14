"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.resetHardhatContext = void 0;
const context_1 = require("./context");
// This function isn't meant to be used during the Hardhat execution,
// but rather to reset Hardhat in between tests.
function resetHardhatContext() {
    if (context_1.HardhatContext.isCreated()) {
        const ctx = context_1.HardhatContext.getHardhatContext();
        if (ctx.environment !== undefined) {
            const globalAsAny = global;
            for (const key of Object.keys(ctx.environment)) {
                globalAsAny.hre = undefined;
                globalAsAny[key] = undefined;
            }
        }
        const filesLoadedDuringConfig = ctx.getFilesLoadedDuringConfig();
        filesLoadedDuringConfig.forEach(unloadModule);
        context_1.HardhatContext.deleteHardhatContext();
    }
    // Unload all the hardhat's entry-points.
    unloadModule("../register");
    unloadModule("./cli/cli");
    unloadModule("./lib/hardhat-lib");
}
exports.resetHardhatContext = resetHardhatContext;
function unloadModule(path) {
    try {
        delete require.cache[require.resolve(path)];
    }
    catch {
        // module wasn't loaded
    }
}
//# sourceMappingURL=reset.js.map