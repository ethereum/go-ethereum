"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.handleVars = void 0;
const picocolors_1 = __importDefault(require("picocolors"));
const debug_1 = __importDefault(require("debug"));
const errors_1 = require("../core/errors");
const errors_list_1 = require("../core/errors-list");
const context_1 = require("../context");
const vars_manager_setup_1 = require("../core/vars/vars-manager-setup");
const config_loading_1 = require("../core/config/config-loading");
const global_dir_1 = require("../util/global-dir");
const ArgumentsParser_1 = require("./ArgumentsParser");
const emoji_1 = require("./emoji");
const log = (0, debug_1.default)("hardhat:cli:vars");
async function handleVars(allUnparsedCLAs, configPath) {
    const { taskDefinition, taskArguments } = await getTaskDefinitionAndTaskArguments(allUnparsedCLAs);
    switch (taskDefinition.name) {
        case "set":
            return set(taskArguments.var, taskArguments.value);
        case "get":
            return get(taskArguments.var);
        case "list":
            return list();
        case "delete":
            return del(taskArguments.var);
        case "path":
            return path();
        case "setup":
            return setup(configPath);
        default:
            console.error(picocolors_1.default.red(`Invalid task '${taskDefinition.name}'`));
            return 1; // Error code
    }
}
exports.handleVars = handleVars;
async function set(key, value) {
    const varsManager = context_1.HardhatContext.getHardhatContext().varsManager;
    varsManager.validateKey(key);
    varsManager.set(key, value ?? (await getVarValue()));
    if (process.stdout.isTTY) {
        console.warn(`The configuration variable has been stored in ${varsManager.getStoragePath()}`);
    }
    return 0;
}
function get(key) {
    const value = context_1.HardhatContext.getHardhatContext().varsManager.get(key);
    if (value !== undefined) {
        console.log(value);
        return 0;
    }
    console.warn(picocolors_1.default.yellow(`The configuration variable '${key}' is not set in ${context_1.HardhatContext.getHardhatContext().varsManager.getStoragePath()}`));
    return 1;
}
function list() {
    const keys = context_1.HardhatContext.getHardhatContext().varsManager.list();
    const varsStoragePath = context_1.HardhatContext.getHardhatContext().varsManager.getStoragePath();
    if (keys.length > 0) {
        keys.forEach((k) => console.log(k));
        if (process.stdout.isTTY) {
            console.warn(`\nAll configuration variables are stored in ${varsStoragePath}`);
        }
    }
    else {
        if (process.stdout.isTTY) {
            console.warn(picocolors_1.default.yellow(`There are no configuration variables stored in ${varsStoragePath}`));
        }
    }
    return 0;
}
function del(key) {
    const varsStoragePath = context_1.HardhatContext.getHardhatContext().varsManager.getStoragePath();
    if (context_1.HardhatContext.getHardhatContext().varsManager.delete(key)) {
        if (process.stdout.isTTY) {
            console.warn(`The configuration variable was deleted from ${varsStoragePath}`);
        }
        return 0;
    }
    console.warn(picocolors_1.default.yellow(`There is no configuration variable '${key}' to delete from ${varsStoragePath}`));
    return 1;
}
function path() {
    console.log(context_1.HardhatContext.getHardhatContext().varsManager.getStoragePath());
    return 0;
}
function setup(configPath) {
    log("Switching to SetupVarsManager to collect vars");
    const varsManagerSetup = new vars_manager_setup_1.VarsManagerSetup((0, global_dir_1.getVarsFilePath)());
    context_1.HardhatContext.getHardhatContext().varsManager = varsManagerSetup;
    try {
        log("Loading config and tasks to trigger vars collection");
        loadConfigFile(configPath);
    }
    catch (err) {
        console.error(picocolors_1.default.red("There is an error in your Hardhat configuration file. Please double check it.\n"));
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw err;
    }
    listVarsToSetup(varsManagerSetup);
    return 0;
}
// The code below duplicates a section from the 'loadConfigAndTasks' function.
// While we could have refactored the 'config-loading.ts' module to make this logic reusable,
// it would have added complexity and potentially made the code harder to understand.
function loadConfigFile(configPath) {
    const configEnv = require(`../core/config/config-env`);
    // Load all the functions and objects exported by the 'config-env' file in a global scope
    const globalAsAny = global;
    Object.entries(configEnv).forEach(([key, value]) => (globalAsAny[key] = value));
    const resolvedConfigPath = (0, config_loading_1.resolveConfigPath)(configPath);
    (0, config_loading_1.importCsjOrEsModule)(resolvedConfigPath);
}
async function getVarValue() {
    const { default: enquirer } = await Promise.resolve().then(() => __importStar(require("enquirer")));
    const response = await enquirer.prompt({
        type: "password",
        name: "value",
        message: "Enter value:",
    });
    return response.value;
}
function listVarsToSetup(varsManagerSetup) {
    const HH_SET_COMMAND = "npx hardhat vars set";
    const requiredKeysToSet = varsManagerSetup.getRequiredVarsToSet();
    const optionalKeysToSet = varsManagerSetup.getOptionalVarsToSet();
    if (requiredKeysToSet.length === 0 && optionalKeysToSet.length === 0) {
        console.log(picocolors_1.default.green("There are no configuration variables that need to be set for this project"));
        console.log();
        printAlreadySetKeys(varsManagerSetup);
        return;
    }
    if (requiredKeysToSet.length > 0) {
        console.log(picocolors_1.default.bold(`${(0, emoji_1.emoji)("❗ ")}The following configuration variables need to be set:\n`));
        console.log(requiredKeysToSet.map((k) => `  ${HH_SET_COMMAND} ${k}`).join("\n"));
        console.log();
    }
    if (optionalKeysToSet.length > 0) {
        console.log(picocolors_1.default.bold(`${(0, emoji_1.emoji)("💡 ")}The following configuration variables are optional:\n`));
        console.log(optionalKeysToSet.map((k) => `  ${HH_SET_COMMAND} ${k}`).join("\n"));
        console.log();
    }
    printAlreadySetKeys(varsManagerSetup);
}
function printAlreadySetKeys(varsManagerSetup) {
    const requiredKeysAlreadySet = varsManagerSetup.getRequiredVarsAlreadySet();
    const optionalKeysAlreadySet = varsManagerSetup.getOptionalVarsAlreadySet();
    const envVars = varsManagerSetup.getEnvVars();
    if (requiredKeysAlreadySet.length === 0 &&
        optionalKeysAlreadySet.length === 0 &&
        envVars.length === 0) {
        return;
    }
    console.log(`${picocolors_1.default.bold(`${(0, emoji_1.emoji)("✔️  ")}Configuration variables already set:`)}`);
    console.log();
    if (requiredKeysAlreadySet.length > 0) {
        console.log("  Mandatory:");
        console.log(requiredKeysAlreadySet.map((x) => `    ${x}`).join("\n"));
        console.log();
    }
    if (optionalKeysAlreadySet.length > 0) {
        console.log("  Optional:");
        console.log(optionalKeysAlreadySet.map((x) => `    ${x}`).join("\n"));
        console.log();
    }
    if (envVars.length > 0) {
        console.log("  Set via environment variables:");
        console.log(envVars.map((x) => `    ${x}`).join("\n"));
        console.log();
    }
}
async function getTaskDefinitionAndTaskArguments(allUnparsedCLAs) {
    const ctx = context_1.HardhatContext.getHardhatContext();
    ctx.setConfigLoadingAsStarted();
    require("../../builtin-tasks/vars");
    ctx.setConfigLoadingAsFinished();
    const argumentsParser = new ArgumentsParser_1.ArgumentsParser();
    const taskDefinitions = ctx.tasksDSL.getTaskDefinitions();
    const scopesDefinitions = ctx.tasksDSL.getScopesDefinitions();
    const { scopeName, taskName, unparsedCLAs } = argumentsParser.parseScopeAndTaskNames(allUnparsedCLAs, taskDefinitions, scopesDefinitions);
    (0, errors_1.assertHardhatInvariant)(scopeName === "vars", "This function should only be called to handle tasks under the 'vars' scope");
    const taskDefinition = ctx.tasksDSL.getTaskDefinition(scopeName, taskName);
    if (taskDefinition === undefined) {
        throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.UNRECOGNIZED_SCOPED_TASK, {
            scope: scopeName,
            task: taskName,
        });
    }
    const taskArguments = argumentsParser.parseTaskArguments(taskDefinition, unparsedCLAs);
    return { taskDefinition, taskArguments };
}
//# sourceMappingURL=vars.js.map