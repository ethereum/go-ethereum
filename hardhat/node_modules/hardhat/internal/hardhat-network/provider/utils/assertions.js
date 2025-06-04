"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.assertHardhatNetworkInvariant = void 0;
const errors_1 = require("../../../core/providers/errors");
function assertHardhatNetworkInvariant(invariant, description) {
    if (!invariant) {
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw new errors_1.InternalError(`Internal Hardhat Network invariant was violated: ${description}`);
    }
}
exports.assertHardhatNetworkInvariant = assertHardhatNetworkInvariant;
//# sourceMappingURL=assertions.js.map