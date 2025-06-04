#!/usr/bin/env node
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
const prettier = __importStar(require("prettier"));
const runTypeChain_1 = require("../typechain/runTypeChain");
const files_1 = require("../utils/files");
const glob_1 = require("../utils/glob");
const logger_1 = require("../utils/logger");
const parseArgs_1 = require("./parseArgs");
async function main() {
    ;
    global.IS_CLI = true;
    const cliConfig = (0, parseArgs_1.parseArgs)();
    const cwd = process.cwd();
    const files = getFilesToProcess(cwd, cliConfig.files);
    if (files.length === 0) {
        throw new Error('No files passed.' + '\n' + `\`${cliConfig.files}\` didn't match any input files in ${cwd}`);
    }
    const config = {
        cwd,
        target: cliConfig.target,
        outDir: cliConfig.outDir,
        allFiles: files,
        filesToProcess: files,
        inputDir: cliConfig.inputDir || (0, files_1.detectInputsRoot)(files),
        prettier,
        flags: {
            ...cliConfig.flags,
            environment: undefined,
        },
    };
    const result = await (0, runTypeChain_1.runTypeChain)(config);
    // eslint-disable-next-line no-console
    console.log(`Successfully generated ${result.filesGenerated} typings!`);
}
main().catch((e) => {
    logger_1.logger.error('Error occured: ', e.message);
    const stackTracesEnabled = process.argv.includes('--show-stack-traces');
    if (stackTracesEnabled) {
        logger_1.logger.error('Stack trace: ', e.stack);
    }
    else {
        logger_1.logger.error('Run with --show-stack-traces to see the full stacktrace');
    }
    process.exit(1);
});
function getFilesToProcess(cwd, filesOrPattern) {
    var _a;
    let res = (0, glob_1.glob)(cwd, filesOrPattern);
    if (res.length === 0) {
        // If there are no files found, but first parameter is surrounded with single quotes, we try again without quotes
        const match = (_a = filesOrPattern[0].match(/'([\s\S]*)'/)) === null || _a === void 0 ? void 0 : _a[1];
        if (match)
            res = (0, glob_1.glob)(cwd, [match]);
    }
    return res;
}
//# sourceMappingURL=cli.js.map