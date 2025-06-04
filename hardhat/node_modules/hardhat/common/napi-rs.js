"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.requireNapiRsModule = void 0;
const errors_1 = require("../internal/core/errors");
const errors_list_1 = require("../internal/core/errors-list");
function requireNapiRsModule(id) {
    try {
        return require(id);
    }
    catch (e) {
        if (e.code === "MODULE_NOT_FOUND") {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.GENERAL.CORRUPTED_LOCKFILE);
        }
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw e;
    }
}
exports.requireNapiRsModule = requireNapiRsModule;
//# sourceMappingURL=napi-rs.js.map