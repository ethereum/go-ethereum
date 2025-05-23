"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.BroadcastedTxDifferentHash = exports.AccountIndexOutOfRange = exports.UnsupportedEventError = exports.NotImplementedError = exports.HardhatEthersError = void 0;
const plugins_1 = require("hardhat/plugins");
class HardhatEthersError extends plugins_1.NomicLabsHardhatPluginError {
    constructor(message, parent) {
        super("@nomicfoundation/hardhat-ethers", message, parent);
    }
}
exports.HardhatEthersError = HardhatEthersError;
class NotImplementedError extends HardhatEthersError {
    constructor(method) {
        super(`Method '${method}' is not implemented`);
    }
}
exports.NotImplementedError = NotImplementedError;
class UnsupportedEventError extends HardhatEthersError {
    constructor(event) {
        super(`Event '${event}' is not supported`);
    }
}
exports.UnsupportedEventError = UnsupportedEventError;
class AccountIndexOutOfRange extends HardhatEthersError {
    constructor(accountIndex, accountsLength) {
        super(`Tried to get account with index ${accountIndex} but there are ${accountsLength} accounts`);
    }
}
exports.AccountIndexOutOfRange = AccountIndexOutOfRange;
class BroadcastedTxDifferentHash extends HardhatEthersError {
    constructor(txHash, broadcastedTxHash) {
        super(`Expected broadcasted transaction to have hash '${txHash}', but got '${broadcastedTxHash}'`);
    }
}
exports.BroadcastedTxDifferentHash = BroadcastedTxDifferentHash;
//# sourceMappingURL=errors.js.map