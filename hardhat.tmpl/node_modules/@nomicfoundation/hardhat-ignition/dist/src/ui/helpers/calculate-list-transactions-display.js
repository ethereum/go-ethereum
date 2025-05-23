"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.transactionDisplaySerializeReplacer = exports.calculateListTransactionsDisplay = void 0;
const plugins_1 = require("hardhat/plugins");
const json5_1 = require("json5");
function calculateListTransactionsDisplay(deploymentId, listTransactionsResult, configUrl) {
    let text = `Logging transactions for deployment ${deploymentId}\n\n`;
    for (const [index, transaction] of listTransactionsResult.entries()) {
        const txLink = getTransactionLink(transaction.txHash, configUrl ?? transaction.browserUrl);
        text += `Transaction ${index + 1}${txLink === undefined ? "" : txLink}:\n`;
        text += `  - Type: ${transactionTypeToDisplayType(transaction.type)}\n`;
        text += `  - Status: ${transaction.status}\n`;
        text += `  - TxHash: ${transaction.txHash}\n`;
        text += `  - From: ${transaction.from}\n`;
        if (transaction.to !== undefined) {
            text += `  - To: ${transaction.to}\n`;
        }
        if (transaction.name !== undefined) {
            text += `  - Name: ${transaction.name}\n`;
        }
        if (transaction.address !== undefined) {
            text += `  - Address: ${transaction.address}\n`;
        }
        if (transaction.params !== undefined) {
            text += `  - Params: ${(0, json5_1.stringify)(transaction.params, transactionDisplaySerializeReplacer)}\n`;
        }
        if (transaction.value !== undefined) {
            text += `  - Value: '${transaction.value}n'\n`;
        }
        text += "\n";
    }
    return text;
}
exports.calculateListTransactionsDisplay = calculateListTransactionsDisplay;
function transactionTypeToDisplayType(type) {
    switch (type) {
        case "DEPLOYMENT_EXECUTION_STATE":
            return "Contract Deployment";
        case "CALL_EXECUTION_STATE":
            return "Function Call";
        case "SEND_DATA_EXECUTION_STATE":
            return "Generic Transaction";
        default:
            throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", `Unknown transaction type: ${type}`);
    }
}
function transactionDisplaySerializeReplacer(_key, value) {
    if (typeof value === "bigint") {
        return `${value}n`;
    }
    return value;
}
exports.transactionDisplaySerializeReplacer = transactionDisplaySerializeReplacer;
function getTransactionLink(txHash, browserURL) {
    if (browserURL === undefined) {
        return undefined;
    }
    return `\x1b]8;;${browserURL}/tx/${txHash}\x1b\\ (🔗 view on block explorer)\x1b]8;;\x1b\\`;
}
//# sourceMappingURL=calculate-list-transactions-display.js.map