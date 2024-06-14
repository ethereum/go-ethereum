"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.EphemeralDeploymentLoader = void 0;
const memory_journal_1 = require("../journal/memory-journal");
const assertions_1 = require("../utils/assertions");
/**
 * Stores and loads deployment related information without making changes
 * on disk, by either storing in memory or loading already existing files.
 * Used when running in environments like Hardhat tests.
 */
class EphemeralDeploymentLoader {
    _artifactResolver;
    _executionEventListener;
    _journal;
    _deployedAddresses;
    _savedArtifacts;
    constructor(_artifactResolver, _executionEventListener) {
        this._artifactResolver = _artifactResolver;
        this._executionEventListener = _executionEventListener;
        this._journal = new memory_journal_1.MemoryJournal(this._executionEventListener);
        this._deployedAddresses = {};
        this._savedArtifacts = {};
    }
    async recordToJournal(message) {
        this._journal.record(message);
    }
    readFromJournal() {
        return this._journal.read();
    }
    async recordDeployedAddress(futureId, contractAddress) {
        this._deployedAddresses[futureId] = contractAddress;
    }
    async storeBuildInfo(_futureId, _buildInfo) {
        // For ephemeral we are ignoring build info
    }
    async storeNamedArtifact(futureId, contractName, _artifact) {
        this._savedArtifacts[futureId] = { _kind: "contractName", contractName };
    }
    async storeUserProvidedArtifact(futureId, artifact) {
        this._savedArtifacts[futureId] = { _kind: "artifact", artifact };
    }
    async loadArtifact(artifactId) {
        const futureId = artifactId;
        const saved = this._savedArtifacts[futureId];
        (0, assertions_1.assertIgnitionInvariant)(saved !== undefined, `No stored artifact for ${futureId}`);
        switch (saved._kind) {
            case "artifact": {
                return saved.artifact;
            }
            case "contractName": {
                const fileArtifact = this._artifactResolver.loadArtifact(saved.contractName);
                (0, assertions_1.assertIgnitionInvariant)(fileArtifact !== undefined, `Unable to load artifact, underlying resolver returned undefined for ${saved.contractName}`);
                return fileArtifact;
            }
        }
    }
}
exports.EphemeralDeploymentLoader = EphemeralDeploymentLoader;
//# sourceMappingURL=ephemeral-deployment-loader.js.map