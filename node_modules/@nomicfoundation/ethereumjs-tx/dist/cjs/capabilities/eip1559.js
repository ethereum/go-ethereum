"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getUpfrontCost = void 0;
function getUpfrontCost(tx, baseFee) {
    const prio = tx.maxPriorityFeePerGas;
    const maxBase = tx.maxFeePerGas - baseFee;
    const inclusionFeePerGas = prio < maxBase ? prio : maxBase;
    const gasPrice = inclusionFeePerGas + baseFee;
    return tx.gasLimit * gasPrice + tx.value;
}
exports.getUpfrontCost = getUpfrontCost;
//# sourceMappingURL=eip1559.js.map