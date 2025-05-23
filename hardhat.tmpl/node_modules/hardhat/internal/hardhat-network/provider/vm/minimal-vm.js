"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getMinimalEthereumJsVm = exports.MinimalEthereumJsEvmEventEmitter = exports.MinimalEthereumJsVmEventEmitter = void 0;
const util_1 = require("@ethereumjs/util");
class MinimalEthereumJsVmEventEmitter extends util_1.AsyncEventEmitter {
}
exports.MinimalEthereumJsVmEventEmitter = MinimalEthereumJsVmEventEmitter;
class MinimalEthereumJsEvmEventEmitter extends util_1.AsyncEventEmitter {
}
exports.MinimalEthereumJsEvmEventEmitter = MinimalEthereumJsEvmEventEmitter;
function getMinimalEthereumJsVm(provider) {
    const minimalEthereumJsVm = {
        events: new MinimalEthereumJsVmEventEmitter(),
        evm: {
            events: new MinimalEthereumJsEvmEventEmitter(),
        },
        stateManager: {
            putContractCode: async (address, code) => {
                await provider.handleRequest(JSON.stringify({
                    method: "hardhat_setCode",
                    params: [address.toString(), `0x${code.toString("hex")}`],
                }));
            },
            getContractStorage: async (address, slotHash) => {
                const responseObject = await provider.handleRequest(JSON.stringify({
                    method: "eth_getStorageAt",
                    params: [address.toString(), `0x${slotHash.toString("hex")}`],
                }));
                let response;
                if (typeof responseObject.data === "string") {
                    response = JSON.parse(responseObject.data);
                }
                else {
                    response = responseObject.data;
                }
                return Buffer.from(response.result.slice(2), "hex");
            },
            putContractStorage: async (address, slotHash, slotValue) => {
                await provider.handleRequest(JSON.stringify({
                    method: "hardhat_setStorageAt",
                    params: [
                        address.toString(),
                        `0x${slotHash.toString("hex")}`,
                        `0x${slotValue.toString("hex")}`,
                    ],
                }));
            },
        },
    };
    return minimalEthereumJsVm;
}
exports.getMinimalEthereumJsVm = getMinimalEthereumJsVm;
//# sourceMappingURL=minimal-vm.js.map