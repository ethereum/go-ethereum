import { Artifact, ArtifactResolver, BuildInfo } from "../../types/artifact";
import { ExecutionEventListener } from "../../types/execution-events";
import { JournalMessage } from "../execution/types/messages";
import { DeploymentLoader } from "./types";
/**
 * Stores and loads deployment related information without making changes
 * on disk, by either storing in memory or loading already existing files.
 * Used when running in environments like Hardhat tests.
 */
export declare class EphemeralDeploymentLoader implements DeploymentLoader {
    private _artifactResolver;
    private _executionEventListener?;
    private _journal;
    private _deployedAddresses;
    private _savedArtifacts;
    constructor(_artifactResolver: ArtifactResolver, _executionEventListener?: ExecutionEventListener | undefined);
    recordToJournal(message: JournalMessage): Promise<void>;
    readFromJournal(): AsyncGenerator<JournalMessage, any, unknown>;
    recordDeployedAddress(futureId: string, contractAddress: string): Promise<void>;
    storeBuildInfo(_futureId: string, _buildInfo: BuildInfo): Promise<void>;
    storeNamedArtifact(futureId: string, contractName: string, _artifact: Artifact): Promise<void>;
    storeUserProvidedArtifact(futureId: string, artifact: Artifact): Promise<void>;
    loadArtifact(artifactId: string): Promise<Artifact>;
}
//# sourceMappingURL=ephemeral-deployment-loader.d.ts.map