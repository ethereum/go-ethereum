"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.HardhatArtifactResolver = void 0;
const fs_1 = __importDefault(require("fs"));
const plugins_1 = require("hardhat/plugins");
const path_1 = __importDefault(require("path"));
class HardhatArtifactResolver {
    _hre;
    constructor(_hre) {
        this._hre = _hre;
    }
    async getBuildInfo(contractName) {
        const artifactPath = await this._resolvePath(contractName);
        if (artifactPath === undefined) {
            throw new plugins_1.HardhatPluginError("hardhat-ignition", `Artifact path not found for ${contractName}`);
        }
        const debugPath = artifactPath.replace(".json", ".dbg.json");
        const debugJson = await fs_1.default.promises.readFile(debugPath);
        const buildInfoPath = path_1.default.join(path_1.default.parse(debugPath).dir, JSON.parse(debugJson.toString()).buildInfo);
        const buildInfoJson = await fs_1.default.promises.readFile(buildInfoPath);
        return JSON.parse(buildInfoJson.toString());
    }
    async _resolvePath(contractName) {
        const artifactPaths = await this._hre.artifacts.getArtifactPaths();
        const artifactPath = artifactPaths.find((p) => path_1.default.parse(p).name === contractName);
        return artifactPath;
    }
    loadArtifact(contractName) {
        return this._hre.artifacts.readArtifact(contractName);
    }
}
exports.HardhatArtifactResolver = HardhatArtifactResolver;
//# sourceMappingURL=hardhat-artifact-resolver.js.map