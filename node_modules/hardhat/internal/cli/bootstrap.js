#!/usr/bin/env node
"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const semver_1 = __importDefault(require("semver"));
const chalk_1 = __importDefault(require("chalk"));
const constants_1 = require("./constants");
if (!semver_1.default.satisfies(process.version, constants_1.SUPPORTED_NODE_VERSIONS.join(" || "))) {
    console.warn(chalk_1.default.yellow.bold(`WARNING:`), `You are currently using Node.js ${process.version}, which is not supported by Hardhat. This can lead to unexpected behavior. See https://hardhat.org/nodejs-versions`);
    console.log();
    console.log();
}
require("./cli");
//# sourceMappingURL=bootstrap.js.map