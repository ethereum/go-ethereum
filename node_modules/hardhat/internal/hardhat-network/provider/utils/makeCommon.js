"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.makeCommon = void 0;
const ethereumjs_common_1 = require("@nomicfoundation/ethereumjs-common");
const hardforks_1 = require("../../../util/hardforks");
function makeCommon({ chainId, networkId, hardfork }) {
    const common = ethereumjs_common_1.Common.custom({
        chainId,
        networkId,
    }, {
        // ethereumjs uses this name for the merge hardfork
        hardfork: hardfork === hardforks_1.HardforkName.MERGE ? "mergeForkIdTransition" : hardfork,
    });
    return common;
}
exports.makeCommon = makeCommon;
//# sourceMappingURL=makeCommon.js.map