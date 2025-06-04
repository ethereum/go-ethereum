"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getArtifactFromContractOutput = exports.Artifacts = void 0;
const debug_1 = __importDefault(require("debug"));
const fs_extra_1 = __importDefault(require("fs-extra"));
const os = __importStar(require("os"));
const path = __importStar(require("path"));
const promises_1 = __importDefault(require("fs/promises"));
const contract_names_1 = require("../utils/contract-names");
const source_names_1 = require("../utils/source-names");
const constants_1 = require("./constants");
const errors_1 = require("./core/errors");
const errors_list_1 = require("./core/errors-list");
const hash_1 = require("./util/hash");
const fs_utils_1 = require("./util/fs-utils");
const log = (0, debug_1.default)("hardhat:core:artifacts");
class Artifacts {
    constructor(_artifactsPath) {
        this._artifactsPath = _artifactsPath;
        // Undefined means that the cache is disabled.
        this._cache = {
            artifactNameToArtifactPathCache: new Map(),
            artifactFQNToBuildInfoPathCache: new Map(),
        };
        this._validArtifacts = [];
    }
    addValidArtifacts(validArtifacts) {
        this._validArtifacts.push(...validArtifacts);
    }
    async readArtifact(name) {
        const artifactPath = await this._getArtifactPath(name);
        return fs_extra_1.default.readJson(artifactPath);
    }
    readArtifactSync(name) {
        const artifactPath = this._getArtifactPathSync(name);
        return fs_extra_1.default.readJsonSync(artifactPath);
    }
    async artifactExists(name) {
        let artifactPath;
        try {
            artifactPath = await this._getArtifactPath(name);
        }
        catch (e) {
            if (errors_1.HardhatError.isHardhatError(e)) {
                return false;
            }
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw e;
        }
        return fs_extra_1.default.pathExists(artifactPath);
    }
    async getAllFullyQualifiedNames() {
        const paths = await this.getArtifactPaths();
        return paths.map((p) => this._getFullyQualifiedNameFromPath(p)).sort();
    }
    async getBuildInfo(fullyQualifiedName) {
        let buildInfoPath = this._cache?.artifactFQNToBuildInfoPathCache.get(fullyQualifiedName);
        if (buildInfoPath === undefined) {
            const artifactPath = this.formArtifactPathFromFullyQualifiedName(fullyQualifiedName);
            const debugFilePath = this._getDebugFilePath(artifactPath);
            buildInfoPath = await this._getBuildInfoFromDebugFile(debugFilePath);
            if (buildInfoPath === undefined) {
                return undefined;
            }
            this._cache?.artifactFQNToBuildInfoPathCache.set(fullyQualifiedName, buildInfoPath);
        }
        return fs_extra_1.default.readJSON(buildInfoPath);
    }
    getBuildInfoSync(fullyQualifiedName) {
        let buildInfoPath = this._cache?.artifactFQNToBuildInfoPathCache.get(fullyQualifiedName);
        if (buildInfoPath === undefined) {
            const artifactPath = this.formArtifactPathFromFullyQualifiedName(fullyQualifiedName);
            const debugFilePath = this._getDebugFilePath(artifactPath);
            buildInfoPath = this._getBuildInfoFromDebugFileSync(debugFilePath);
            if (buildInfoPath === undefined) {
                return undefined;
            }
            this._cache?.artifactFQNToBuildInfoPathCache.set(fullyQualifiedName, buildInfoPath);
        }
        return fs_extra_1.default.readJSONSync(buildInfoPath);
    }
    async getArtifactPaths() {
        const cached = this._cache?.artifactPaths;
        if (cached !== undefined) {
            return cached;
        }
        const paths = await (0, fs_utils_1.getAllFilesMatching)(this._artifactsPath, (f) => this._isArtifactPath(f));
        const result = paths.sort();
        if (this._cache !== undefined) {
            this._cache.artifactPaths = result;
        }
        return result;
    }
    async getBuildInfoPaths() {
        const cached = this._cache?.buildInfoPaths;
        if (cached !== undefined) {
            return cached;
        }
        const paths = await (0, fs_utils_1.getAllFilesMatching)(path.join(this._artifactsPath, constants_1.BUILD_INFO_DIR_NAME), (f) => f.endsWith(".json"));
        const result = paths.sort();
        if (this._cache !== undefined) {
            this._cache.buildInfoPaths = result;
        }
        return result;
    }
    async getDebugFilePaths() {
        const cached = this._cache?.debugFilePaths;
        if (cached !== undefined) {
            return cached;
        }
        const paths = await (0, fs_utils_1.getAllFilesMatching)(path.join(this._artifactsPath), (f) => f.endsWith(".dbg.json"));
        const result = paths.sort();
        if (this._cache !== undefined) {
            this._cache.debugFilePaths = result;
        }
        return result;
    }
    async saveArtifactAndDebugFile(artifact, pathToBuildInfo) {
        try {
            // artifact
            const fullyQualifiedName = (0, contract_names_1.getFullyQualifiedName)(artifact.sourceName, artifact.contractName);
            const artifactPath = this.formArtifactPathFromFullyQualifiedName(fullyQualifiedName);
            await fs_extra_1.default.ensureDir(path.dirname(artifactPath));
            await Promise.all([
                fs_extra_1.default.writeJSON(artifactPath, artifact, {
                    spaces: 2,
                }),
                (async () => {
                    if (pathToBuildInfo === undefined) {
                        return;
                    }
                    // save debug file
                    const debugFilePath = this._getDebugFilePath(artifactPath);
                    const debugFile = this._createDebugFile(artifactPath, pathToBuildInfo);
                    await fs_extra_1.default.writeJSON(debugFilePath, debugFile, {
                        spaces: 2,
                    });
                })(),
            ]);
        }
        finally {
            this.clearCache();
        }
    }
    async saveBuildInfo(solcVersion, solcLongVersion, input, output) {
        try {
            const buildInfoDir = path.join(this._artifactsPath, constants_1.BUILD_INFO_DIR_NAME);
            await fs_extra_1.default.ensureDir(buildInfoDir);
            const buildInfoName = this._getBuildInfoName(solcVersion, solcLongVersion, input);
            const buildInfo = this._createBuildInfo(buildInfoName, solcVersion, solcLongVersion, input, output);
            const buildInfoPath = path.join(buildInfoDir, `${buildInfoName}.json`);
            // JSON.stringify of the entire build info can be really slow
            // in larger projects, so we stringify per part and incrementally create
            // the JSON in the file.
            //
            // We split this code into different curly-brace-enclosed scopes so that
            // partial JSON strings get out of scope sooner and hence can be reclaimed
            // by the GC if needed.
            const file = await promises_1.default.open(buildInfoPath, "w");
            try {
                {
                    const withoutOutput = JSON.stringify({
                        ...buildInfo,
                        output: undefined,
                    });
                    // We write the JSON (without output) except the last }
                    await file.write(withoutOutput.slice(0, -1));
                }
                {
                    const outputWithoutSourcesAndContracts = JSON.stringify({
                        ...buildInfo.output,
                        sources: undefined,
                        contracts: undefined,
                    });
                    // We start writing the output
                    await file.write(',"output":');
                    // Write the output object except for the last }
                    await file.write(outputWithoutSourcesAndContracts.slice(0, -1));
                    // If there were other field apart from sources and contracts we need
                    // a comma
                    if (outputWithoutSourcesAndContracts.length > 2) {
                        await file.write(",");
                    }
                }
                // Writing the sources
                await file.write('"sources":{');
                let isFirst = true;
                for (const [name, value] of Object.entries(buildInfo.output.sources ?? {})) {
                    if (isFirst) {
                        isFirst = false;
                    }
                    else {
                        await file.write(",");
                    }
                    await file.write(`${JSON.stringify(name)}:${JSON.stringify(value)}`);
                }
                // Close sources object
                await file.write("}");
                // Writing the contracts
                await file.write(',"contracts":{');
                isFirst = true;
                for (const [name, value] of Object.entries(buildInfo.output.contracts ?? {})) {
                    if (isFirst) {
                        isFirst = false;
                    }
                    else {
                        await file.write(",");
                    }
                    await file.write(`${JSON.stringify(name)}:${JSON.stringify(value)}`);
                }
                // close contracts object
                await file.write("}");
                // close output object
                await file.write("}");
                // close build info object
                await file.write("}");
            }
            finally {
                await file.close();
            }
            return buildInfoPath;
        }
        finally {
            this.clearCache();
        }
    }
    /**
     * Remove all artifacts that don't correspond to the current solidity files
     */
    async removeObsoleteArtifacts() {
        // We clear the cache here, as we want to be sure this runs correctly
        this.clearCache();
        try {
            const validArtifactPaths = await Promise.all(this._validArtifacts.flatMap(({ sourceName, artifacts }) => artifacts.map((artifactName) => this._getArtifactPath((0, contract_names_1.getFullyQualifiedName)(sourceName, artifactName)))));
            const validArtifactsPathsSet = new Set(validArtifactPaths);
            for (const { sourceName, artifacts } of this._validArtifacts) {
                for (const artifactName of artifacts) {
                    validArtifactsPathsSet.add(this.formArtifactPathFromFullyQualifiedName((0, contract_names_1.getFullyQualifiedName)(sourceName, artifactName)));
                }
            }
            const existingArtifactsPaths = await this.getArtifactPaths();
            await Promise.all(existingArtifactsPaths
                .filter((artifactPath) => !validArtifactsPathsSet.has(artifactPath))
                .map((artifactPath) => this._removeArtifactFiles(artifactPath)));
            await this._removeObsoleteBuildInfos();
        }
        finally {
            // We clear the cache here, as this may have non-existent paths now
            this.clearCache();
        }
    }
    /**
     * Returns the absolute path to the given artifact
     * @throws {HardhatError} If the name is not fully qualified.
     */
    formArtifactPathFromFullyQualifiedName(fullyQualifiedName) {
        const { sourceName, contractName } = (0, contract_names_1.parseFullyQualifiedName)(fullyQualifiedName);
        return path.join(this._artifactsPath, sourceName, `${contractName}.json`);
    }
    clearCache() {
        // Avoid accidentally re-enabling the cache
        if (this._cache === undefined) {
            return;
        }
        this._cache = {
            artifactFQNToBuildInfoPathCache: new Map(),
            artifactNameToArtifactPathCache: new Map(),
        };
    }
    disableCache() {
        this._cache = undefined;
    }
    /**
     * Remove all build infos that aren't used by any debug file
     */
    async _removeObsoleteBuildInfos() {
        const debugFiles = await this.getDebugFilePaths();
        const buildInfos = await Promise.all(debugFiles.map(async (debugFile) => {
            const buildInfoFile = await this._getBuildInfoFromDebugFile(debugFile);
            if (buildInfoFile !== undefined) {
                return path.resolve(path.dirname(debugFile), buildInfoFile);
            }
        }));
        const filteredBuildInfos = buildInfos.filter((bf) => typeof bf === "string");
        const validBuildInfos = new Set(filteredBuildInfos);
        const buildInfoFiles = await this.getBuildInfoPaths();
        await Promise.all(buildInfoFiles
            .filter((buildInfoFile) => !validBuildInfos.has(buildInfoFile))
            .map(async (buildInfoFile) => {
            log(`Removing buildInfo '${buildInfoFile}'`);
            await fs_extra_1.default.unlink(buildInfoFile);
        }));
    }
    _getBuildInfoName(solcVersion, solcLongVersion, input) {
        const json = JSON.stringify({
            _format: constants_1.BUILD_INFO_FORMAT_VERSION,
            solcVersion,
            solcLongVersion,
            input,
        });
        return (0, hash_1.createNonCryptographicHashBasedIdentifier)(Buffer.from(json)).toString("hex");
    }
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
    async _getArtifactPath(name) {
        const cached = this._cache?.artifactNameToArtifactPathCache.get(name);
        if (cached !== undefined) {
            return cached;
        }
        let result;
        if ((0, contract_names_1.isFullyQualifiedName)(name)) {
            result = await this._getValidArtifactPathFromFullyQualifiedName(name);
        }
        else {
            const files = await this.getArtifactPaths();
            result = this._getArtifactPathFromFiles(name, files);
        }
        this._cache?.artifactNameToArtifactPathCache.set(name, result);
        return result;
    }
    _createBuildInfo(id, solcVersion, solcLongVersion, input, output) {
        return {
            id,
            _format: constants_1.BUILD_INFO_FORMAT_VERSION,
            solcVersion,
            solcLongVersion,
            input,
            output,
        };
    }
    _createDebugFile(artifactPath, pathToBuildInfo) {
        const relativePathToBuildInfo = path.relative(path.dirname(artifactPath), pathToBuildInfo);
        const debugFile = {
            _format: constants_1.DEBUG_FILE_FORMAT_VERSION,
            buildInfo: relativePathToBuildInfo,
        };
        return debugFile;
    }
    _getArtifactPathsSync() {
        const cached = this._cache?.artifactPaths;
        if (cached !== undefined) {
            return cached;
        }
        const paths = (0, fs_utils_1.getAllFilesMatchingSync)(this._artifactsPath, (f) => this._isArtifactPath(f));
        const result = paths.sort();
        if (this._cache !== undefined) {
            this._cache.artifactPaths = result;
        }
        return result;
    }
    /**
     * Sync version of _getArtifactPath
     */
    _getArtifactPathSync(name) {
        const cached = this._cache?.artifactNameToArtifactPathCache.get(name);
        if (cached !== undefined) {
            return cached;
        }
        let result;
        if ((0, contract_names_1.isFullyQualifiedName)(name)) {
            result = this._getValidArtifactPathFromFullyQualifiedNameSync(name);
        }
        else {
            const files = this._getArtifactPathsSync();
            result = this._getArtifactPathFromFiles(name, files);
        }
        this._cache?.artifactNameToArtifactPathCache.set(name, result);
        return result;
    }
    /**
     * DO NOT DELETE OR CHANGE
     *
     * use this.formArtifactPathFromFullyQualifiedName instead
     * @deprecated until typechain migrates to public version
     * @see https://github.com/dethcrypto/TypeChain/issues/544
     */
    _getArtifactPathFromFullyQualifiedName(fullyQualifiedName) {
        const { sourceName, contractName } = (0, contract_names_1.parseFullyQualifiedName)(fullyQualifiedName);
        return path.join(this._artifactsPath, sourceName, `${contractName}.json`);
    }
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
    async _getValidArtifactPathFromFullyQualifiedName(fullyQualifiedName) {
        const artifactPath = this.formArtifactPathFromFullyQualifiedName(fullyQualifiedName);
        try {
            const trueCasePath = path.join(this._artifactsPath, await (0, fs_utils_1.getFileTrueCase)(this._artifactsPath, path.relative(this._artifactsPath, artifactPath)));
            if (artifactPath !== trueCasePath) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.ARTIFACTS.WRONG_CASING, {
                    correct: this._getFullyQualifiedNameFromPath(trueCasePath),
                    incorrect: fullyQualifiedName,
                });
            }
            return trueCasePath;
        }
        catch (e) {
            if (e instanceof fs_utils_1.FileNotFoundError) {
                return this._handleWrongArtifactForFullyQualifiedName(fullyQualifiedName);
            }
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw e;
        }
    }
    _getAllContractNamesFromFiles(files) {
        return files.map((file) => {
            const fqn = this._getFullyQualifiedNameFromPath(file);
            return (0, contract_names_1.parseFullyQualifiedName)(fqn).contractName;
        });
    }
    _getAllFullyQualifiedNamesSync() {
        const paths = this._getArtifactPathsSync();
        return paths.map((p) => this._getFullyQualifiedNameFromPath(p)).sort();
    }
    _formatSuggestions(names, contractName) {
        switch (names.length) {
            case 0:
                return "";
            case 1:
                return `Did you mean "${names[0]}"?`;
            default:
                return `We found some that were similar:

${names.map((n) => `  * ${n}`).join(os.EOL)}

Please replace "${contractName}" for the correct contract name wherever you are trying to read its artifact.
`;
        }
    }
    /**
     * @throws {HardhatError} with a list of similar contract names.
     */
    _handleWrongArtifactForFullyQualifiedName(fullyQualifiedName) {
        const names = this._getAllFullyQualifiedNamesSync();
        const similarNames = this._getSimilarContractNames(fullyQualifiedName, names);
        throw new errors_1.HardhatError(errors_list_1.ERRORS.ARTIFACTS.NOT_FOUND, {
            contractName: fullyQualifiedName,
            suggestion: this._formatSuggestions(similarNames, fullyQualifiedName),
        });
    }
    /**
     * @throws {HardhatError} with a list of similar contract names.
     */
    _handleWrongArtifactForContractName(contractName, files) {
        const names = this._getAllContractNamesFromFiles(files);
        let similarNames = this._getSimilarContractNames(contractName, names);
        if (similarNames.length > 1) {
            similarNames = this._filterDuplicatesAsFullyQualifiedNames(files, similarNames);
        }
        throw new errors_1.HardhatError(errors_list_1.ERRORS.ARTIFACTS.NOT_FOUND, {
            contractName,
            suggestion: this._formatSuggestions(similarNames, contractName),
        });
    }
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
    _filterDuplicatesAsFullyQualifiedNames(files, similarNames) {
        const outputNames = [];
        const groups = similarNames.reduce((obj, cur) => {
            // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
            obj[cur] = obj[cur] ? obj[cur] + 1 : 1;
            return obj;
        }, {});
        for (const [name, occurrences] of Object.entries(groups)) {
            if (occurrences > 1) {
                for (const file of files) {
                    if (path.basename(file) === `${name}.json`) {
                        outputNames.push(this._getFullyQualifiedNameFromPath(file));
                    }
                }
                continue;
            }
            outputNames.push(name);
        }
        return outputNames;
    }
    /**
     *
     * @param givenName can be FQN or contract name
     * @param names MUST match type of givenName (i.e. array of FQN's if givenName is FQN)
     * @returns
     */
    _getSimilarContractNames(givenName, names) {
        let shortestDistance = constants_1.EDIT_DISTANCE_THRESHOLD;
        let mostSimilarNames = [];
        for (const name of names) {
            const distance = (0, contract_names_1.findDistance)(givenName, name);
            if (distance < shortestDistance) {
                shortestDistance = distance;
                mostSimilarNames = [name];
                continue;
            }
            if (distance === shortestDistance) {
                mostSimilarNames.push(name);
                continue;
            }
        }
        return mostSimilarNames;
    }
    _getValidArtifactPathFromFullyQualifiedNameSync(fullyQualifiedName) {
        const artifactPath = this.formArtifactPathFromFullyQualifiedName(fullyQualifiedName);
        try {
            const trueCasePath = path.join(this._artifactsPath, (0, fs_utils_1.getFileTrueCaseSync)(this._artifactsPath, path.relative(this._artifactsPath, artifactPath)));
            if (artifactPath !== trueCasePath) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.ARTIFACTS.WRONG_CASING, {
                    correct: this._getFullyQualifiedNameFromPath(trueCasePath),
                    incorrect: fullyQualifiedName,
                });
            }
            return trueCasePath;
        }
        catch (e) {
            if (e instanceof fs_utils_1.FileNotFoundError) {
                return this._handleWrongArtifactForFullyQualifiedName(fullyQualifiedName);
            }
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw e;
        }
    }
    _getDebugFilePath(artifactPath) {
        return artifactPath.replace(/\.json$/, ".dbg.json");
    }
    /**
     * Gets the path to the artifact file for the given contract name.
     * @throws {HardhatError} with descriptor:
     * - {@link ERRORS.ARTIFACTS.NOT_FOUND} if there are no artifacts matching the given contract name.
     * - {@link ERRORS.ARTIFACTS.MULTIPLE_FOUND} if there are multiple artifacts matching the given contract name.
     */
    _getArtifactPathFromFiles(contractName, files) {
        const matchingFiles = files.filter((file) => {
            return path.basename(file) === `${contractName}.json`;
        });
        if (matchingFiles.length === 0) {
            return this._handleWrongArtifactForContractName(contractName, files);
        }
        if (matchingFiles.length > 1) {
            const candidates = matchingFiles.map((file) => this._getFullyQualifiedNameFromPath(file));
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARTIFACTS.MULTIPLE_FOUND, {
                contractName,
                candidates: candidates.join(os.EOL),
            });
        }
        return matchingFiles[0];
    }
    /**
     * Returns the FQN of a contract giving the absolute path to its artifact.
     *
     * For example, given a path like
     * `/path/to/project/artifacts/contracts/Foo.sol/Bar.json`, it'll return the
     * FQN `contracts/Foo.sol:Bar`
     */
    _getFullyQualifiedNameFromPath(absolutePath) {
        const sourceName = (0, source_names_1.replaceBackslashes)(path.relative(this._artifactsPath, path.dirname(absolutePath)));
        const contractName = path.basename(absolutePath).replace(".json", "");
        return (0, contract_names_1.getFullyQualifiedName)(sourceName, contractName);
    }
    /**
     * Remove the artifact file and its debug file.
     */
    async _removeArtifactFiles(artifactPath) {
        await fs_extra_1.default.remove(artifactPath);
        const debugFilePath = this._getDebugFilePath(artifactPath);
        await fs_extra_1.default.remove(debugFilePath);
    }
    /**
     * Given the path to a debug file, returns the absolute path to its
     * corresponding build info file if it exists, or undefined otherwise.
     */
    async _getBuildInfoFromDebugFile(debugFilePath) {
        if (await fs_extra_1.default.pathExists(debugFilePath)) {
            const { buildInfo } = await fs_extra_1.default.readJson(debugFilePath);
            return path.resolve(path.dirname(debugFilePath), buildInfo);
        }
        return undefined;
    }
    /**
     * Sync version of _getBuildInfoFromDebugFile
     */
    _getBuildInfoFromDebugFileSync(debugFilePath) {
        if (fs_extra_1.default.pathExistsSync(debugFilePath)) {
            const { buildInfo } = fs_extra_1.default.readJsonSync(debugFilePath);
            return path.resolve(path.dirname(debugFilePath), buildInfo);
        }
        return undefined;
    }
    _isArtifactPath(file) {
        return (file.endsWith(".json") &&
            file !== path.join(this._artifactsPath, "package.json") &&
            !file.startsWith(path.join(this._artifactsPath, constants_1.BUILD_INFO_DIR_NAME)) &&
            !file.endsWith(".dbg.json"));
    }
}
exports.Artifacts = Artifacts;
/**
 * Retrieves an artifact for the given `contractName` from the compilation output.
 *
 * @param sourceName The contract's source name.
 * @param contractName the contract's name.
 * @param contractOutput the contract's compilation output as emitted by `solc`.
 */
function getArtifactFromContractOutput(sourceName, contractName, contractOutput) {
    const evmBytecode = contractOutput.evm?.bytecode;
    let bytecode = evmBytecode?.object ?? "";
    if (bytecode.slice(0, 2).toLowerCase() !== "0x") {
        bytecode = `0x${bytecode}`;
    }
    const evmDeployedBytecode = contractOutput.evm?.deployedBytecode;
    let deployedBytecode = evmDeployedBytecode?.object ?? "";
    if (deployedBytecode.slice(0, 2).toLowerCase() !== "0x") {
        deployedBytecode = `0x${deployedBytecode}`;
    }
    const linkReferences = evmBytecode?.linkReferences ?? {};
    const deployedLinkReferences = evmDeployedBytecode?.linkReferences ?? {};
    return {
        _format: constants_1.ARTIFACT_FORMAT_VERSION,
        contractName,
        sourceName,
        abi: contractOutput.abi,
        bytecode,
        deployedBytecode,
        linkReferences,
        deployedLinkReferences,
    };
}
exports.getArtifactFromContractOutput = getArtifactFromContractOutput;
//# sourceMappingURL=artifacts.js.map