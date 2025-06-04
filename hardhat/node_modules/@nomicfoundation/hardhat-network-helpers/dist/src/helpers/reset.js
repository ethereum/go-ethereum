"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reset = void 0;
const loadFixture_1 = require("../loadFixture");
const utils_1 = require("../utils");
async function reset(url, blockNumber) {
    const provider = await (0, utils_1.getHardhatProvider)();
    await (0, loadFixture_1.clearSnapshots)();
    if (url === undefined) {
        await provider.request({
            method: "hardhat_reset",
            params: [],
        });
    }
    else if (blockNumber === undefined) {
        await provider.request({
            method: "hardhat_reset",
            params: [
                {
                    forking: {
                        jsonRpcUrl: url,
                    },
                },
            ],
        });
    }
    else {
        await provider.request({
            method: "hardhat_reset",
            params: [
                {
                    forking: {
                        jsonRpcUrl: url,
                        blockNumber: (0, utils_1.toNumber)(blockNumber),
                    },
                },
            ],
        });
    }
}
exports.reset = reset;
//# sourceMappingURL=reset.js.map