"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.listTransactions = void 0;
const findLastIndex_1 = __importDefault(require("lodash/findLastIndex"));
const errors_1 = require("./errors");
const chain_config_1 = require("./internal/chain-config");
const file_deployment_loader_1 = require("./internal/deployment-loader/file-deployment-loader");
const errors_list_1 = require("./internal/errors-list");
const deployment_state_helpers_1 = require("./internal/execution/deployment-state-helpers");
const execution_result_1 = require("./internal/execution/types/execution-result");
const execution_state_1 = require("./internal/execution/types/execution-state");
const jsonrpc_1 = require("./internal/execution/types/jsonrpc");
const messages_1 = require("./internal/execution/types/messages");
const assertions_1 = require("./internal/utils/assertions");
const list_transactions_1 = require("./types/list-transactions");
/**
 * Returns the transactions associated with a deployment.
 *
 * @param deploymentDir - the directory of the deployment to get the transactions of
 * @param artifactResolver - the artifact resolver to use when loading artifacts
 * for a future
 *
 * @beta
 */
async function listTransactions(deploymentDir, _artifactResolver) {
    const deploymentLoader = new file_deployment_loader_1.FileDeploymentLoader(deploymentDir);
    const deploymentState = await (0, deployment_state_helpers_1.loadDeploymentState)(deploymentLoader);
    if (deploymentState === undefined) {
        throw new errors_1.IgnitionError(errors_list_1.ERRORS.LIST_TRANSACTIONS.UNINITIALIZED_DEPLOYMENT, {
            deploymentDir,
        });
    }
    const transactions = [];
    const browserUrl = chain_config_1.builtinChains.find(({ chainId }) => chainId === deploymentState.chainId);
    for await (const message of deploymentLoader.readFromJournal()) {
        if (message.type !== messages_1.JournalMessageType.TRANSACTION_SEND) {
            continue;
        }
        const exState = deploymentState.executionStates[message.futureId];
        (0, assertions_1.assertIgnitionInvariant)(doesSendTransactions(exState), "Expected execution state to be a type that sends transactions");
        const networkInteraction = exState.networkInteractions[message.networkInteractionId - 1];
        (0, assertions_1.assertIgnitionInvariant)(networkInteraction.type === "ONCHAIN_INTERACTION", "Expected network interaction to be an onchain interaction");
        // this seems redundant, but we use it later to determine pending vs dropped status
        const lastTxIndex = (0, findLastIndex_1.default)(networkInteraction.transactions, (tx) => tx.hash === message.transaction.hash);
        const transaction = networkInteraction.transactions[lastTxIndex];
        switch (exState.type) {
            case execution_state_1.ExecutionStateType.DEPLOYMENT_EXECUTION_STATE: {
                transactions.push({
                    type: exState.type,
                    from: exState.from,
                    txHash: transaction.hash,
                    status: getTransactionStatus(transaction, lastTxIndex === networkInteraction.transactions.length - 1),
                    name: exState.contractName,
                    address: transaction.receipt?.status === jsonrpc_1.TransactionReceiptStatus.SUCCESS
                        ? exState.result?.type === execution_result_1.ExecutionResultType.SUCCESS
                            ? exState.result.address
                            : undefined
                        : undefined,
                    params: exState.constructorArgs,
                    value: networkInteraction.value,
                    browserUrl: browserUrl?.urls.browserURL,
                });
                break;
            }
            case execution_state_1.ExecutionStateType.CALL_EXECUTION_STATE: {
                const artifact = await deploymentLoader.loadArtifact(exState.artifactId);
                transactions.push({
                    type: exState.type,
                    from: exState.from,
                    txHash: transaction.hash,
                    status: getTransactionStatus(transaction, lastTxIndex === networkInteraction.transactions.length - 1),
                    name: `${artifact.contractName}#${exState.functionName}`,
                    to: networkInteraction.to,
                    params: exState.args,
                    value: networkInteraction.value,
                    browserUrl: browserUrl?.urls.browserURL,
                });
                break;
            }
            case execution_state_1.ExecutionStateType.SEND_DATA_EXECUTION_STATE: {
                transactions.push({
                    type: exState.type,
                    from: exState.from,
                    txHash: transaction.hash,
                    status: getTransactionStatus(transaction, lastTxIndex === networkInteraction.transactions.length - 1),
                    to: networkInteraction.to,
                    value: networkInteraction.value,
                    browserUrl: browserUrl?.urls.browserURL,
                });
                break;
            }
        }
    }
    return transactions;
}
exports.listTransactions = listTransactions;
function doesSendTransactions(exState) {
    return (exState.type === execution_state_1.ExecutionStateType.DEPLOYMENT_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.CALL_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.SEND_DATA_EXECUTION_STATE);
}
function getTransactionStatus(transaction, isFinalTransaction) {
    if (transaction.receipt === undefined) {
        if (isFinalTransaction) {
            return list_transactions_1.TransactionStatus.PENDING;
        }
        return list_transactions_1.TransactionStatus.DROPPED;
    }
    if (transaction.receipt.status === jsonrpc_1.TransactionReceiptStatus.SUCCESS) {
        return list_transactions_1.TransactionStatus.SUCCESS;
    }
    return list_transactions_1.TransactionStatus.FAILURE;
}
//# sourceMappingURL=list-transactions.js.map