import { Artifact, ArtifactResolver, BuildInfo } from "@nomicfoundation/ignition-core";
import { HardhatRuntimeEnvironment } from "hardhat/types";
export declare class HardhatArtifactResolver implements ArtifactResolver {
    private _hre;
    constructor(_hre: HardhatRuntimeEnvironment);
    getBuildInfo(contractName: string): Promise<BuildInfo | undefined>;
    private _resolvePath;
    loadArtifact(contractName: string): Promise<Artifact>;
    /**
     * Returns true if a name is fully qualified, and not just a bare contract name.
     *
     * This is based on Hardhat's own test for fully qualified names, taken
     * from `contract-names.ts` in `hardhat-core` utils.
     */
    private _isFullyQualifiedName;
}
//# sourceMappingURL=hardhat-artifact-resolver.d.ts.map