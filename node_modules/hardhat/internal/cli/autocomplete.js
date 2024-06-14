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
exports.complete = exports.REQUIRED_HH_VERSION_RANGE = exports.HARDHAT_COMPLETE_FILES = void 0;
const find_up_1 = __importDefault(require("find-up"));
const fs = __importStar(require("fs-extra"));
const path = __importStar(require("path"));
const hardhat_params_1 = require("../core/params/hardhat-params");
const global_dir_1 = require("../util/global-dir");
const hash_1 = require("../util/hash");
const lang_1 = require("../util/lang");
const ArgumentsParser_1 = require("./ArgumentsParser");
exports.HARDHAT_COMPLETE_FILES = "__hardhat_complete_files__";
exports.REQUIRED_HH_VERSION_RANGE = "^1.0.0";
async function complete({ line, point, }) {
    const completionData = await getCompletionData();
    if (completionData === undefined) {
        return [];
    }
    const { networks, tasks, scopes } = completionData;
    const words = line.split(/\s+/).filter((x) => x.length > 0);
    const wordsBeforeCursor = line.slice(0, point).split(/\s+/);
    // 'prev' and 'last' variables examples:
    // `hh compile --network|` => prev: "compile" last: "--network"
    // `hh compile --network |` => prev: "--network" last: ""
    // `hh compile --network ha|` => prev: "--network" last: "ha"
    const [prev, last] = wordsBeforeCursor.slice(-2);
    const startsWithLast = (completion) => completion.startsWith(last);
    const coreParams = Object.values(hardhat_params_1.HARDHAT_PARAM_DEFINITIONS)
        .map((param) => ({
        name: ArgumentsParser_1.ArgumentsParser.paramNameToCLA(param.name),
        description: param.description ?? "",
    }))
        .filter((x) => !words.includes(x.name));
    // Get the task or scope if the user has entered one
    let taskName;
    let scopeName;
    let index = 1;
    while (index < words.length) {
        const word = words[index];
        if (isGlobalFlag(word)) {
            index += 1;
        }
        else if (isGlobalParam(word)) {
            index += 2;
        }
        else if (word.startsWith("--")) {
            index += 1;
        }
        else {
            // Possible scenarios:
            // - no task or scope: `hh `
            // - only a task: `hh task `
            // - only a scope: `hh scope `
            // - both a scope and a task (the task always follow the scope): `hh scope task `
            // Between a scope and a task there could be other words, e.g.: `hh scope --flag task `
            if (scopeName === undefined) {
                if (tasks[word] !== undefined) {
                    taskName = word;
                    break;
                }
                else if (scopes[word] !== undefined) {
                    scopeName = word;
                }
            }
            else {
                taskName = word;
                break;
            }
            index += 1;
        }
    }
    // If a task or a scope is found and it is equal to the last word,
    // this indicates that the cursor is positioned after the task or scope.
    // In this case, we ignore the task or scope. For instance, if you have a task or a scope named 'foo' and 'foobar',
    // and the line is 'hh foo|', we want to suggest the value for 'foo' and 'foobar'.
    // Possible scenarios:
    // - no task or scope: `hh ` -> task and scope already undefined
    // - only a task: `hh task ` -> task set to undefined, scope already undefined
    // - only a scope: `hh scope ` -> scope set to undefined, task already undefined
    // - both a scope and a task (the task always follow the scope): `hh scope task ` -> task set to undefined, scope stays defined
    if (taskName === last || scopeName === last) {
        if (taskName !== undefined && scopeName !== undefined) {
            [taskName, scopeName] = [undefined, scopeName];
        }
        else {
            [taskName, scopeName] = [undefined, undefined];
        }
    }
    if (prev === "--network") {
        return networks.filter(startsWithLast).map((network) => ({
            name: network,
            description: "",
        }));
    }
    const scopeDefinition = scopeName === undefined ? undefined : scopes[scopeName];
    const taskDefinition = taskName === undefined
        ? undefined
        : scopeDefinition === undefined
            ? tasks[taskName]
            : scopeDefinition.tasks[taskName];
    // if the previous word is a param, then a value is expected
    // we don't complete anything here
    if (prev.startsWith("-")) {
        const paramName = ArgumentsParser_1.ArgumentsParser.cLAToParamName(prev);
        const globalParam = hardhat_params_1.HARDHAT_PARAM_DEFINITIONS[paramName];
        if (globalParam !== undefined && !globalParam.isFlag) {
            return exports.HARDHAT_COMPLETE_FILES;
        }
        const isTaskParam = taskDefinition?.paramDefinitions[paramName]?.isFlag === false;
        if (isTaskParam) {
            return exports.HARDHAT_COMPLETE_FILES;
        }
    }
    // If there's no task or scope, we complete either tasks and scopes or params
    if (taskDefinition === undefined && scopeDefinition === undefined) {
        if (last.startsWith("-")) {
            return coreParams.filter((param) => startsWithLast(param.name));
        }
        const taskSuggestions = Object.values(tasks)
            .filter((x) => !x.isSubtask)
            .map((x) => ({
            name: x.name,
            description: x.description,
        }));
        const scopeSuggestions = Object.values(scopes).map((x) => ({
            name: x.name,
            description: x.description,
        }));
        return taskSuggestions
            .concat(scopeSuggestions)
            .filter((x) => startsWithLast(x.name));
    }
    // If there's a scope but not a task, we complete with the scopes'tasks
    if (taskDefinition === undefined && scopeDefinition !== undefined) {
        return Object.values(scopes[scopeName].tasks)
            .filter((x) => !x.isSubtask)
            .map((x) => ({
            name: x.name,
            description: x.description,
        }))
            .filter((x) => startsWithLast(x.name));
    }
    if (!last.startsWith("-")) {
        return exports.HARDHAT_COMPLETE_FILES;
    }
    const taskParams = taskDefinition === undefined
        ? []
        : Object.values(taskDefinition.paramDefinitions)
            .map((param) => ({
            name: ArgumentsParser_1.ArgumentsParser.paramNameToCLA(param.name),
            description: param.description,
        }))
            .filter((x) => !words.includes(x.name));
    return [...taskParams, ...coreParams].filter((suggestion) => startsWithLast(suggestion.name));
}
exports.complete = complete;
async function getCompletionData() {
    const projectId = getProjectId();
    if (projectId === undefined) {
        return undefined;
    }
    const cachedCompletionData = await getCachedCompletionData(projectId);
    if (cachedCompletionData !== undefined) {
        if (arePreviousMtimesCorrect(cachedCompletionData.mtimes)) {
            return cachedCompletionData.completionData;
        }
    }
    const filesBeforeRequire = Object.keys(require.cache);
    let hre;
    try {
        process.env.TS_NODE_TRANSPILE_ONLY = "1";
        require("../../register");
        hre = global.hre;
    }
    catch {
        return undefined;
    }
    const filesAfterRequire = Object.keys(require.cache);
    const mtimes = getMtimes(filesBeforeRequire, filesAfterRequire);
    const networks = Object.keys(hre.config.networks);
    // we extract the tasks data explicitly to make sure everything
    // is serializable and to avoid saving unnecessary things from the HRE
    const tasks = (0, lang_1.mapValues)(hre.tasks, (task) => getTaskFromTaskDefinition(task));
    const scopes = (0, lang_1.mapValues)(hre.scopes, (scope) => ({
        name: scope.name,
        description: scope.description ?? "",
        tasks: (0, lang_1.mapValues)(scope.tasks, (task) => getTaskFromTaskDefinition(task)),
    }));
    const completionData = {
        networks,
        tasks,
        scopes,
    };
    await saveCachedCompletionData(projectId, completionData, mtimes);
    return completionData;
}
function getTaskFromTaskDefinition(taskDef) {
    return {
        name: taskDef.name,
        description: taskDef.description ?? "",
        isSubtask: taskDef.isSubtask,
        paramDefinitions: (0, lang_1.mapValues)(taskDef.paramDefinitions, (paramDefinition) => ({
            name: paramDefinition.name,
            description: paramDefinition.description ?? "",
            isFlag: paramDefinition.isFlag,
        })),
    };
}
function getProjectId() {
    const packageJsonPath = find_up_1.default.sync("package.json");
    if (packageJsonPath === null) {
        return undefined;
    }
    return (0, hash_1.createNonCryptographicHashBasedIdentifier)(Buffer.from(packageJsonPath)).toString("hex");
}
function arePreviousMtimesCorrect(mtimes) {
    try {
        return Object.entries(mtimes).every(([file, mtime]) => fs.statSync(file).mtime.valueOf() === mtime);
    }
    catch {
        return false;
    }
}
function getMtimes(filesLoadedBefore, filesLoadedAfter) {
    const loadedByHardhat = filesLoadedAfter.filter((f) => !filesLoadedBefore.includes(f));
    const stats = loadedByHardhat.map((f) => fs.statSync(f));
    const mtimes = loadedByHardhat.map((f, i) => ({
        [f]: stats[i].mtime.valueOf(),
    }));
    if (mtimes.length === 0) {
        return {};
    }
    return Object.assign(mtimes[0], ...mtimes.slice(1));
}
async function getCachedCompletionData(projectId) {
    const cachedCompletionDataPath = await getCachedCompletionDataPath(projectId);
    if (fs.existsSync(cachedCompletionDataPath)) {
        try {
            const cachedCompletionData = fs.readJsonSync(cachedCompletionDataPath);
            return cachedCompletionData;
        }
        catch {
            // remove the file if it seems invalid
            fs.unlinkSync(cachedCompletionDataPath);
            return undefined;
        }
    }
}
async function saveCachedCompletionData(projectId, completionData, mtimes) {
    const cachedCompletionDataPath = await getCachedCompletionDataPath(projectId);
    await fs.outputJson(cachedCompletionDataPath, { completionData, mtimes });
}
async function getCachedCompletionDataPath(projectId) {
    const cacheDir = await (0, global_dir_1.getCacheDir)();
    return path.join(cacheDir, "autocomplete", `${projectId}.json`);
}
function isGlobalFlag(param) {
    const paramName = ArgumentsParser_1.ArgumentsParser.cLAToParamName(param);
    return hardhat_params_1.HARDHAT_PARAM_DEFINITIONS[paramName]?.isFlag === true;
}
function isGlobalParam(param) {
    const paramName = ArgumentsParser_1.ArgumentsParser.cLAToParamName(param);
    return hardhat_params_1.HARDHAT_PARAM_DEFINITIONS[paramName]?.isFlag === false;
}
//# sourceMappingURL=autocomplete.js.map