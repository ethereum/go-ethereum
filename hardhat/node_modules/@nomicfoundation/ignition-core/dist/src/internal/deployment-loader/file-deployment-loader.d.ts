import { Artifact, BuildInfo } from "../../types/artifact";
import { ExecutionEventListener } from "../../types/execution-events";
import { JournalMessage } from "../execution/types/messages";
import { DeploymentLoader } from "./types";
export declare class FileDeploymentLoader implements DeploymentLoader {
    private readonly _deploymentDirPath;
    private readonly _executionEventListener?;
    private _journal;
    private _deploymentDirsEnsured;
    private _paths;
    constructor(_deploymentDirPath: string, _executionEventListener?: ExecutionEventListener | undefined);
    recordToJournal(message: JournalMessage): Promise<void>;
    readFromJournal(): AsyncGenerator<JournalMessage, any, unknown>;
    storeNamedArtifact(futureId: string, _contractName: string, artifact: Artifact): Promise<void>;
    storeUserProvidedArtifact(futureId: string, artifact: Artifact): Promise<void>;
    storeBuildInfo(futureId: string, buildInfo: BuildInfo): Promise<void>;
    readBuildInfo(futureId: string): Promise<BuildInfo>;
    loadArtifact(futureId: string): Promise<Artifact>;
    recordDeployedAddress(futureId: string, contractAddress: string): Promise<void>;
    private _initialize;
    private _resolveArtifactPathFor;
}
//# sourceMappingURL=file-deployment-loader.d.ts.map