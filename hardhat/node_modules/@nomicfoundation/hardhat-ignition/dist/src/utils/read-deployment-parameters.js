"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.readDeploymentParameters = void 0;
const fs_extra_1 = require("fs-extra");
const plugins_1 = require("hardhat/plugins");
const json5_1 = require("json5");
const bigintReviver_1 = require("./bigintReviver");
async function readDeploymentParameters(filepath) {
    try {
        const rawFile = await (0, fs_extra_1.readFile)(filepath);
        return await (0, json5_1.parse)(rawFile.toString(), bigintReviver_1.bigintReviver);
    }
    catch (e) {
        if (e instanceof plugins_1.NomicLabsHardhatPluginError) {
            throw e;
        }
        if (e instanceof Error) {
            throw new plugins_1.NomicLabsHardhatPluginError("@nomicfoundation/hardhat-ignition", `Could not parse parameters from ${filepath}`, e);
        }
        throw e;
    }
}
exports.readDeploymentParameters = readDeploymentParameters;
//# sourceMappingURL=read-deployment-parameters.js.map