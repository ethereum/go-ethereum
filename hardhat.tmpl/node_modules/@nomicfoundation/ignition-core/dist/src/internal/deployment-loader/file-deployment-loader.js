"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.FileDeploymentLoader = void 0;
const fs_extra_1 = require("fs-extra");
const path_1 = __importDefault(require("path"));
const file_journal_1 = require("../journal/file-journal");
class FileDeploymentLoader {
    _deploymentDirPath;
    _executionEventListener;
    _journal;
    _deploymentDirsEnsured;
    _paths;
    constructor(_deploymentDirPath, _executionEventListener) {
        this._deploymentDirPath = _deploymentDirPath;
        this._executionEventListener = _executionEventListener;
        const artifactsDir = path_1.default.join(this._deploymentDirPath, "artifacts");
        const buildInfoDir = path_1.default.join(this._deploymentDirPath, "build-info");
        const journalPath = path_1.default.join(this._deploymentDirPath, "journal.jsonl");
        const deployedAddressesPath = path_1.default.join(this._deploymentDirPath, "deployed_addresses.json");
        this._journal = new file_journal_1.FileJournal(journalPath, this._executionEventListener);
        this._paths = {
            deploymentDir: this._deploymentDirPath,
            artifactsDir,
            buildInfoDir,
            journalPath,
            deployedAddressesPath,
        };
        this._deploymentDirsEnsured = false;
    }
    async recordToJournal(message) {
        await this._initialize();
        // NOTE: the journal record is sync, even though this call is async
        this._journal.record(message);
    }
    readFromJournal() {
        return this._journal.read();
    }
    storeNamedArtifact(futureId, _contractName, artifact) {
        // For a file deployment we don't differentiate between
        // named contracts (from HH) and anonymous contracts passed in by the user
        return this.storeUserProvidedArtifact(futureId, artifact);
    }
    async storeUserProvidedArtifact(futureId, artifact) {
        await this._initialize();
        const artifactFilePath = path_1.default.join(this._paths.artifactsDir, `${futureId}.json`);
        await (0, fs_extra_1.writeFile)(artifactFilePath, JSON.stringify(artifact, undefined, 2));
    }
    async storeBuildInfo(futureId, buildInfo) {
        await this._initialize();
        const buildInfoFilePath = path_1.default.join(this._paths.buildInfoDir, `${buildInfo.id}.json`);
        await (0, fs_extra_1.writeFile)(buildInfoFilePath, JSON.stringify(buildInfo, undefined, 2));
        const debugInfoFilePath = path_1.default.join(this._paths.artifactsDir, `${futureId}.dbg.json`);
        const relativeBuildInfoPath = path_1.default.relative(this._paths.artifactsDir, buildInfoFilePath);
        await (0, fs_extra_1.writeFile)(debugInfoFilePath, JSON.stringify({
            _format: "hh-sol-dbg-1",
            buildInfo: relativeBuildInfoPath,
        }, undefined, 2));
    }
    async readBuildInfo(futureId) {
        await this._initialize();
        const debugInfoFilePath = path_1.default.join(this._paths.artifactsDir, `${futureId}.dbg.json`);
        const json = JSON.parse((await (0, fs_extra_1.readFile)(debugInfoFilePath)).toString());
        const buildInfoPath = path_1.default.resolve(this._paths.artifactsDir, json.buildInfo);
        const buildInfo = JSON.parse((await (0, fs_extra_1.readFile)(buildInfoPath)).toString());
        return buildInfo;
    }
    async loadArtifact(futureId) {
        await this._initialize();
        const artifactFilePath = this._resolveArtifactPathFor(futureId);
        const json = await (0, fs_extra_1.readFile)(artifactFilePath);
        const artifact = JSON.parse(json.toString());
        return artifact;
    }
    async recordDeployedAddress(futureId, contractAddress) {
        await this._initialize();
        let deployedAddresses;
        if (await (0, fs_extra_1.pathExists)(this._paths.deployedAddressesPath)) {
            const json = (await (0, fs_extra_1.readFile)(this._paths.deployedAddressesPath)).toString();
            deployedAddresses = JSON.parse(json);
        }
        else {
            deployedAddresses = {};
        }
        deployedAddresses[futureId] = contractAddress;
        await (0, fs_extra_1.writeFile)(this._paths.deployedAddressesPath, `${JSON.stringify(deployedAddresses, undefined, 2)}\n`);
    }
    async _initialize() {
        if (this._deploymentDirsEnsured) {
            return;
        }
        await (0, fs_extra_1.ensureDir)(this._paths.deploymentDir);
        await (0, fs_extra_1.ensureDir)(this._paths.artifactsDir);
        await (0, fs_extra_1.ensureDir)(this._paths.buildInfoDir);
        this._deploymentDirsEnsured = true;
    }
    _resolveArtifactPathFor(futureId) {
        const artifactFilePath = path_1.default.join(this._paths.artifactsDir, `${futureId}.json`);
        return artifactFilePath;
    }
}
exports.FileDeploymentLoader = FileDeploymentLoader;
//# sourceMappingURL=file-deployment-loader.js.map