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
        // If a fully qualified name is used, we can can
        // leverage the artifact manager directly to load the build
        // info.
        if (this._isFullyQualifiedName(contractName)) {
            return this._hre.artifacts.getBuildInfo(contractName);
        }
        // Otherwise we have only the contract name, and need to
        // resolve the artifact for the contract ourselves.
        // We can build on the assumption that the contract name
        // is unique based on Module validation.
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
    /**
     * Returns true if a name is fully qualified, and not just a bare contract name.
     *
     * This is based on Hardhat's own test for fully qualified names, taken
     * from `contract-names.ts` in `hardhat-core` utils.
     */
    _isFullyQualifiedName(contractName) {
        return contractName.includes(":");
    }
}
exports.HardhatArtifactResolver = HardhatArtifactResolver;
//# sourceMappingURL=hardhat-artifact-resolver.js.map