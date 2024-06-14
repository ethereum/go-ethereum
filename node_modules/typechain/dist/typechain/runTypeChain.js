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
Object.defineProperty(exports, "__esModule", { value: true });
exports.runTypeChain = exports.DEFAULT_FLAGS = void 0;
const fs = __importStar(require("fs"));
const mkdirp_1 = require("mkdirp");
const path_1 = require("path");
const prettier = __importStar(require("prettier"));
const debug_1 = require("../utils/debug");
const files_1 = require("../utils/files");
const findTarget_1 = require("./findTarget");
const io_1 = require("./io");
exports.DEFAULT_FLAGS = {
    alwaysGenerateOverloads: false,
    discriminateTypes: false,
    tsNocheck: false,
    environment: undefined,
};
async function runTypeChain(publicConfig) {
    const allFiles = (0, io_1.skipEmptyAbis)(publicConfig.allFiles);
    if (allFiles.length === 0) {
        return {
            filesGenerated: 0,
        };
    }
    // skip empty paths
    const config = {
        flags: exports.DEFAULT_FLAGS,
        inputDir: (0, files_1.detectInputsRoot)(allFiles),
        ...publicConfig,
        allFiles,
        filesToProcess: (0, io_1.skipEmptyAbis)(publicConfig.filesToProcess),
    };
    const services = {
        fs,
        prettier,
        mkdirp: mkdirp_1.sync,
    };
    let filesGenerated = 0;
    const target = (0, findTarget_1.findTarget)(config);
    const fileDescriptions = (0, io_1.loadFileDescriptions)(services, config.filesToProcess);
    (0, debug_1.debug)('Executing beforeRun()');
    filesGenerated += (0, io_1.processOutput)(services, config, await target.beforeRun());
    (0, debug_1.debug)('Executing beforeRun()');
    for (const fd of fileDescriptions) {
        (0, debug_1.debug)(`Processing ${(0, path_1.relative)(config.cwd, fd.path)}`);
        filesGenerated += (0, io_1.processOutput)(services, config, await target.transformFile(fd));
    }
    (0, debug_1.debug)('Running afterRun()');
    filesGenerated += (0, io_1.processOutput)(services, config, await target.afterRun());
    return {
        filesGenerated,
    };
}
exports.runTypeChain = runTypeChain;
//# sourceMappingURL=runTypeChain.js.map