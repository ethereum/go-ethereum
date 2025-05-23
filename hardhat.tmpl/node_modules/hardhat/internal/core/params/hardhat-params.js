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
exports.HARDHAT_PARAM_DEFINITIONS = void 0;
const types = __importStar(require("./argumentTypes"));
exports.HARDHAT_PARAM_DEFINITIONS = {
    network: {
        name: "network",
        defaultValue: undefined,
        description: "The network to connect to.",
        type: types.string,
        isOptional: true,
        isFlag: false,
        isVariadic: false,
    },
    showStackTraces: {
        name: "showStackTraces",
        defaultValue: false,
        description: "Show stack traces (always enabled on CI servers).",
        type: types.boolean,
        isFlag: true,
        isOptional: true,
        isVariadic: false,
    },
    version: {
        name: "version",
        defaultValue: false,
        description: "Shows hardhat's version.",
        type: types.boolean,
        isFlag: true,
        isOptional: true,
        isVariadic: false,
    },
    help: {
        name: "help",
        defaultValue: false,
        description: "Shows this message, or a task's help if its name is provided",
        type: types.boolean,
        isFlag: true,
        isOptional: true,
        isVariadic: false,
    },
    emoji: {
        name: "emoji",
        defaultValue: process.platform === "darwin",
        description: "Use emoji in messages.",
        type: types.boolean,
        isFlag: true,
        isOptional: true,
        isVariadic: false,
    },
    config: {
        name: "config",
        defaultValue: undefined,
        description: "A Hardhat config file.",
        type: types.inputFile,
        isFlag: false,
        isOptional: true,
        isVariadic: false,
    },
    verbose: {
        name: "verbose",
        defaultValue: false,
        description: "Enables Hardhat verbose logging",
        type: types.boolean,
        isFlag: true,
        isOptional: true,
        isVariadic: false,
    },
    maxMemory: {
        name: "maxMemory",
        defaultValue: undefined,
        description: "The maximum amount of memory that Hardhat can use.",
        type: types.int,
        isOptional: true,
        isFlag: false,
        isVariadic: false,
    },
    tsconfig: {
        name: "tsconfig",
        defaultValue: undefined,
        description: "A TypeScript config file.",
        type: types.inputFile,
        isOptional: true,
        isFlag: false,
        isVariadic: false,
    },
    flamegraph: {
        name: "flamegraph",
        defaultValue: undefined,
        description: "Generate a flamegraph of your Hardhat tasks",
        type: types.boolean,
        isOptional: true,
        isFlag: true,
        isVariadic: false,
    },
    typecheck: {
        name: "typecheck",
        defaultValue: false,
        description: "Enable TypeScript type-checking of your scripts/tests",
        type: types.boolean,
        isFlag: true,
        isOptional: true,
        isVariadic: false,
    },
};
//# sourceMappingURL=hardhat-params.js.map