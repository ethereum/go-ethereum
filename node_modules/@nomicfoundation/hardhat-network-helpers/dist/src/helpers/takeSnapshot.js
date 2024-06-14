"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.takeSnapshot = void 0;
const errors_1 = require("../errors");
const utils_1 = require("../utils");
/**
 * Takes a snapshot of the state of the blockchain at the current block.
 *
 * Returns an object with a `restore` method that can be used to reset the
 * network to this state.
 */
async function takeSnapshot() {
    const provider = await (0, utils_1.getHardhatProvider)();
    let snapshotId = await provider.request({
        method: "evm_snapshot",
    });
    if (typeof snapshotId !== "string") {
        throw new errors_1.HardhatNetworkHelpersError("Assertion error: the value returned by evm_snapshot should be a string");
    }
    return {
        restore: async () => {
            const reverted = await provider.request({
                method: "evm_revert",
                params: [snapshotId],
            });
            if (typeof reverted !== "boolean") {
                throw new errors_1.HardhatNetworkHelpersError("Assertion error: the value returned by evm_revert should be a boolean");
            }
            if (!reverted) {
                throw new errors_1.InvalidSnapshotError();
            }
            // re-take the snapshot so that `restore` can be called again
            snapshotId = await provider.request({
                method: "evm_snapshot",
            });
        },
        snapshotId,
    };
}
exports.takeSnapshot = takeSnapshot;
//# sourceMappingURL=takeSnapshot.js.map