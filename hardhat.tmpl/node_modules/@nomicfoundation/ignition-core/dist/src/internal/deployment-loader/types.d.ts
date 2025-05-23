import { Artifact, BuildInfo } from "../../types/artifact";
import { JournalMessage } from "../execution/types/messages";
/**
 * Read and write to the deployment storage.
 *
 * @beta
 */
export interface DeploymentLoader {
    recordToJournal(message: JournalMessage): Promise<void>;
    readFromJournal(): AsyncGenerator<JournalMessage>;
    loadArtifact(artifactId: string): Promise<Artifact>;
    storeUserProvidedArtifact(futureId: string, artifact: Artifact): Promise<void>;
    storeNamedArtifact(futureId: string, contractName: string, artifact: Artifact): Promise<void>;
    storeBuildInfo(futureId: string, buildInfo: BuildInfo): Promise<void>;
    recordDeployedAddress(futureId: string, contractAddress: string): Promise<void>;
}
//# sourceMappingURL=types.d.ts.map