"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.checkAutominedNetwork = void 0;
async function checkAutominedNetwork(provider) {
    try {
        const isHardhat = Boolean(await provider.request({ method: "hardhat_getAutomine" }));
        if (isHardhat) {
            return true;
        }
    }
    catch {
        // If this method failed we aren't using Hardhat Network nor Anvil, so we
        // just continue with the next check.
    }
    try {
        const isGanache = /ganache/i.test((await provider.request({ method: "web3_clientVersion" })));
        if (isGanache) {
            return true;
        }
    }
    catch {
        // If this method failed we aren't using Ganache
    }
    return false;
}
exports.checkAutominedNetwork = checkAutominedNetwork;
//# sourceMappingURL=check-automined-network.js.map