import { Artifact, Artifacts as IArtifacts, BuildInfo, CompilerInput, CompilerOutput } from "../types";
export declare class Artifacts implements IArtifacts {
    private _artifactsPath;
    private _validArtifacts;
    private _cache?;
    constructor(_artifactsPath: string);
    addValidArtifacts(validArtifacts: Array<{
        sourceName: string;
        artifacts: string[];
    }>): void;
    readArtifact(name: string): Promise<Artifact>;
    readArtifactSync(name: string): Artifact;
    artifactExists(name: string): Promise<boolean>;
    getAllFullyQualifiedNames(): Promise<string[]>;
    getBuildInfo(fullyQualifiedName: string): Promise<BuildInfo | undefined>;
    getBuildInfoSync(fullyQualifiedName: string): BuildInfo | undefined;
    getArtifactPaths(): Promise<string[]>;
    getBuildInfoPaths(): Promise<string[]>;
    getDebugFilePaths(): Promise<string[]>;
    saveArtifactAndDebugFile(artifact: Artifact, pathToBuildInfo?: string): Promise<void>;
    saveBuildInfo(solcVersion: string, solcLongVersion: string, input: CompilerInput, output: CompilerOutput): Promise<string>;
    /**
     * Remove all artifacts that don't correspond to the current solidity files
     */
    removeObsoleteArtifacts(): Promise<void>;
    /**
     * Returns the absolute path to the given artifact
     * @throws {HardhatError} If the name is not fully qualified.
     */
    formArtifactPathFromFullyQualifiedName(fullyQualifiedName: string): string;
    clearCache(): void;
    disableCache(): void;
    /**
     * Remove all build infos that aren't used by any debug file
     */
    private _removeObsoleteBuildInfos;
    private _getBuildInfoName;
    /**
     * Returns the absolute path to the artifact that corresponds to the given
     * name.
     *
     * If the name is fully qualified, the path is computed from it.  If not, an
     * artifact that matches the given name is searched in the existing artifacts.
     * If there is an ambiguity, an error is thrown.
     *
     * @throws {HardhatError} with descriptor:
     * - {@link ERRORS.ARTIFACTS.WRONG_CASING} if the path case doesn't match the one in the filesystem.
     * - {@link ERRORS.ARTIFACTS.MULTIPLE_FOUND} if there are multiple artifacts matching the given contract name.
     * - {@link ERRORS.ARTIFACTS.NOT_FOUND} if the artifact is not found.
     */
    private _getArtifactPath;
    private _createBuildInfo;
    private _createDebugFile;
    private _getArtifactPathsSync;
    /**
     * Sync version of _getArtifactPath
     */
    private _getArtifactPathSync;
    /**
     * DO NOT DELETE OR CHANGE
     *
     * use this.formArtifactPathFromFullyQualifiedName instead
     * @deprecated until typechain migrates to public version
     * @see https://github.com/dethcrypto/TypeChain/issues/544
     */
    private _getArtifactPathFromFullyQualifiedName;
    /**
     * Returns the absolute path to the artifact that corresponds to the given
     * fully qualified name.
     * @param fullyQualifiedName The fully qualified name of the contract.
     * @returns The absolute path to the artifact.
     * @throws {HardhatError} with descriptor:
     * - {@link ERRORS.CONTRACT_NAMES.INVALID_FULLY_QUALIFIED_NAME} If the name is not fully qualified.
     * - {@link ERRORS.ARTIFACTS.WRONG_CASING} If the path case doesn't match the one in the filesystem.
     * - {@link ERRORS.ARTIFACTS.NOT_FOUND} If the artifact is not found.
     */
    private _getValidArtifactPathFromFullyQualifiedName;
    private _getAllContractNamesFromFiles;
    private _getAllFullyQualifiedNamesSync;
    private _formatSuggestions;
    /**
     * @throws {HardhatError} with a list of similar contract names.
     */
    private _handleWrongArtifactForFullyQualifiedName;
    /**
     * @throws {HardhatError} with a list of similar contract names.
     */
    private _handleWrongArtifactForContractName;
    /**
     * If the project has these contracts:
     *   - 'contracts/Greeter.sol:Greeter'
     *   - 'contracts/Meeter.sol:Greeter'
     *   - 'contracts/Greater.sol:Greater'
     *  And the user tries to get an artifact with the name 'Greter', then
     *  the suggestions will be 'Greeter', 'Greeter', and 'Greater'.
     *
     * We don't want to show duplicates here, so we use FQNs for those. The
     * suggestions will then be:
     *   - 'contracts/Greeter.sol:Greeter'
     *   - 'contracts/Meeter.sol:Greeter'
     *   - 'Greater'
     */
    private _filterDuplicatesAsFullyQualifiedNames;
    /**
     *
     * @param givenName can be FQN or contract name
     * @param names MUST match type of givenName (i.e. array of FQN's if givenName is FQN)
     * @returns
     */
    private _getSimilarContractNames;
    private _getValidArtifactPathFromFullyQualifiedNameSync;
    private _getDebugFilePath;
    /**
     * Gets the path to the artifact file for the given contract name.
     * @throws {HardhatError} with descriptor:
     * - {@link ERRORS.ARTIFACTS.NOT_FOUND} if there are no artifacts matching the given contract name.
     * - {@link ERRORS.ARTIFACTS.MULTIPLE_FOUND} if there are multiple artifacts matching the given contract name.
     */
    private _getArtifactPathFromFiles;
    /**
     * Returns the FQN of a contract giving the absolute path to its artifact.
     *
     * For example, given a path like
     * `/path/to/project/artifacts/contracts/Foo.sol/Bar.json`, it'll return the
     * FQN `contracts/Foo.sol:Bar`
     */
    private _getFullyQualifiedNameFromPath;
    /**
     * Remove the artifact file and its debug file.
     */
    private _removeArtifactFiles;
    /**
     * Given the path to a debug file, returns the absolute path to its
     * corresponding build info file if it exists, or undefined otherwise.
     */
    private _getBuildInfoFromDebugFile;
    /**
     * Sync version of _getBuildInfoFromDebugFile
     */
    private _getBuildInfoFromDebugFileSync;
    private _isArtifactPath;
}
/**
 * Retrieves an artifact for the given `contractName` from the compilation output.
 *
 * @param sourceName The contract's source name.
 * @param contractName the contract's name.
 * @param contractOutput the contract's compilation output as emitted by `solc`.
 */
export declare function getArtifactFromContractOutput(sourceName: string, contractName: string, contractOutput: any): Artifact;
//# sourceMappingURL=artifacts.d.ts.map