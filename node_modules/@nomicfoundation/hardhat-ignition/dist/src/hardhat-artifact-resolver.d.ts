import { Artifact, ArtifactResolver, BuildInfo } from "@nomicfoundation/ignition-core";
import { HardhatRuntimeEnvironment } from "hardhat/types";
export declare class HardhatArtifactResolver implements ArtifactResolver {
    private _hre;
    constructor(_hre: HardhatRuntimeEnvironment);
    getBuildInfo(contractName: string): Promise<BuildInfo | undefined>;
    private _resolvePath;
    loadArtifact(contractName: string): Promise<Artifact>;
}
//# sourceMappingURL=hardhat-artifact-resolver.d.ts.map