"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.deploy = void 0;
const errors_1 = require("./errors");
const defaultConfig_1 = require("./internal/defaultConfig");
const deployer_1 = require("./internal/deployer");
const ephemeral_deployment_loader_1 = require("./internal/deployment-loader/ephemeral-deployment-loader");
const file_deployment_loader_1 = require("./internal/deployment-loader/file-deployment-loader");
const errors_list_1 = require("./internal/errors-list");
const jsonrpc_client_1 = require("./internal/execution/jsonrpc-client");
const address_1 = require("./internal/execution/utils/address");
const get_default_sender_1 = require("./internal/execution/utils/get-default-sender");
const check_automined_network_1 = require("./internal/utils/check-automined-network");
const validate_1 = require("./internal/validation/validate");
const resolve_strategy_1 = require("./strategies/resolve-strategy");
const execution_events_1 = require("./types/execution-events");
/**
 * Deploy an IgnitionModule to the chain
 *
 * @beta
 */
async function deploy({ config = {}, artifactResolver, provider, executionEventListener, deploymentDir, ignitionModule, deploymentParameters, accounts, defaultSender: givenDefaultSender, strategy, strategyConfig, maxFeePerGasLimit, maxPriorityFeePerGas, }) {
    const executionStrategy = (0, resolve_strategy_1.resolveStrategy)(strategy, strategyConfig);
    if (executionEventListener !== undefined) {
        executionEventListener.setModuleId({
            type: execution_events_1.ExecutionEventType.SET_MODULE_ID,
            moduleName: ignitionModule.id,
        });
        executionEventListener.setStrategy({
            type: execution_events_1.ExecutionEventType.SET_STRATEGY,
            strategy: executionStrategy.name,
        });
    }
    const validationResult = await (0, validate_1.validate)(ignitionModule, artifactResolver, deploymentParameters, accounts);
    if (validationResult !== null) {
        if (executionEventListener !== undefined) {
            executionEventListener.deploymentComplete({
                type: execution_events_1.ExecutionEventType.DEPLOYMENT_COMPLETE,
                result: validationResult,
            });
        }
        return validationResult;
    }
    const defaultSender = _resolveDefaultSender(givenDefaultSender, accounts);
    const deploymentLoader = deploymentDir === undefined
        ? new ephemeral_deployment_loader_1.EphemeralDeploymentLoader(artifactResolver, executionEventListener)
        : new file_deployment_loader_1.FileDeploymentLoader(deploymentDir, executionEventListener);
    const jsonRpcClient = new jsonrpc_client_1.EIP1193JsonRpcClient(provider, {
        maxFeePerGasLimit,
        maxPriorityFeePerGas,
    });
    const isAutominedNetwork = await (0, check_automined_network_1.checkAutominedNetwork)(provider);
    const resolvedConfig = {
        ...defaultConfig_1.defaultConfig,
        requiredConfirmations: isAutominedNetwork
            ? defaultConfig_1.DEFAULT_AUTOMINE_REQUIRED_CONFIRMATIONS
            : config.requiredConfirmations ?? defaultConfig_1.defaultConfig.requiredConfirmations,
        ...config,
    };
    const deployer = new deployer_1.Deployer(resolvedConfig, deploymentDir, executionStrategy, jsonRpcClient, artifactResolver, deploymentLoader, executionEventListener);
    return deployer.deploy(ignitionModule, deploymentParameters, accounts, defaultSender);
}
exports.deploy = deploy;
function _resolveDefaultSender(givenDefaultSender, accounts) {
    let defaultSender;
    if (givenDefaultSender !== undefined) {
        const isDefaultSenderInAccounts = accounts.some((account) => (0, address_1.equalAddresses)(account, givenDefaultSender));
        if (!isDefaultSenderInAccounts) {
            throw new errors_1.IgnitionError(errors_list_1.ERRORS.VALIDATION.INVALID_DEFAULT_SENDER, {
                defaultSender: givenDefaultSender,
            });
        }
        defaultSender = givenDefaultSender;
    }
    else {
        defaultSender = (0, get_default_sender_1.getDefaultSender)(accounts);
    }
    return defaultSender;
}
//# sourceMappingURL=deploy.js.map