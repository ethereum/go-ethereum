"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getFileTrueCaseSync = exports.getFileTrueCase = exports.getAllFilesMatchingSync = exports.getAllFilesMatching = exports.getRealPathSync = exports.getRealPath = exports.InvalidDirectoryError = exports.FileNotFoundError = exports.FileSystemAccessError = void 0;
const promises_1 = __importDefault(require("fs/promises"));
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const errors_1 = require("../core/errors");
// We use this error to encapsulate any other error possibly thrown by node's
// fs apis, as sometimes their errors don't have stack traces.
class FileSystemAccessError extends errors_1.CustomError {
}
exports.FileSystemAccessError = FileSystemAccessError;
class FileNotFoundError extends errors_1.CustomError {
    constructor(filePath, parent) {
        super(`File ${filePath} not found`, parent);
    }
}
exports.FileNotFoundError = FileNotFoundError;
class InvalidDirectoryError extends errors_1.CustomError {
    constructor(filePath, parent) {
        super(`Invalid directory ${filePath}`, parent);
    }
}
exports.InvalidDirectoryError = InvalidDirectoryError;
/**
 * Returns the real path of absolutePath, resolving symlinks.
 *
 * @throws FileNotFoundError if absolutePath doesn't exist.
 */
async function getRealPath(absolutePath) {
    try {
        // This method returns the actual casing.
        // Please read Node.js' docs to learn more.
        return await promises_1.default.realpath(path_1.default.normalize(absolutePath));
    }
    catch (e) {
        if (e.code === "ENOENT") {
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw new FileNotFoundError(absolutePath, e);
        }
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw new FileSystemAccessError(e.message, e);
    }
}
exports.getRealPath = getRealPath;
/**
 * Sync version of getRealPath
 *
 * @see getRealCase
 */
function getRealPathSync(absolutePath) {
    try {
        // This method returns the actual casing.
        // Please read Node.js' docs to learn more.
        return fs_1.default.realpathSync.native(path_1.default.normalize(absolutePath));
    }
    catch (e) {
        if (e.code === "ENOENT") {
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw new FileNotFoundError(absolutePath, e);
        }
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw new FileSystemAccessError(e.message, e);
    }
}
exports.getRealPathSync = getRealPathSync;
/**
 * Returns an array of files (not dirs) that match a condition.
 *
 * @param absolutePathToDir A directory. If it doesn't exist `[]` is returned.
 * @param matches A function to filter files (not directories)
 * @returns An array of absolute paths. Each file has its true case, except
 *  for the initial absolutePathToDir part, which preserves the given casing.
 *  No order is guaranteed.
 */
async function getAllFilesMatching(absolutePathToDir, matches) {
    const dir = await readdir(absolutePathToDir);
    const results = await Promise.all(dir.map(async (file) => {
        const absolutePathToFile = path_1.default.join(absolutePathToDir, file);
        const stats = await promises_1.default.stat(absolutePathToFile);
        if (stats.isDirectory()) {
            const files = await getAllFilesMatching(absolutePathToFile, matches);
            return files.flat();
        }
        else if (matches === undefined || matches(absolutePathToFile)) {
            return absolutePathToFile;
        }
        else {
            return [];
        }
    }));
    return results.flat();
}
exports.getAllFilesMatching = getAllFilesMatching;
/**
 * Sync version of getAllFilesMatching
 *
 * @see getAllFilesMatching
 */
function getAllFilesMatchingSync(absolutePathToDir, matches) {
    const dir = readdirSync(absolutePathToDir);
    const results = dir.map((file) => {
        const absolutePathToFile = path_1.default.join(absolutePathToDir, file);
        const stats = fs_1.default.statSync(absolutePathToFile);
        if (stats.isDirectory()) {
            return getAllFilesMatchingSync(absolutePathToFile, matches).flat();
        }
        else if (matches === undefined || matches(absolutePathToFile)) {
            return absolutePathToFile;
        }
        else {
            return [];
        }
    });
    return results.flat();
}
exports.getAllFilesMatchingSync = getAllFilesMatchingSync;
/**
 * Returns the true case relative path of `relativePath` from `from`, without
 * resolving symlinks.
 */
async function getFileTrueCase(from, relativePath) {
    const dirEntries = await readdir(from);
    const parts = relativePath.split(path_1.default.sep);
    const nextDirLowerCase = parts[0].toLowerCase();
    for (const dirEntry of dirEntries) {
        if (dirEntry.toLowerCase() === nextDirLowerCase) {
            if (parts.length === 1) {
                return dirEntry;
            }
            return path_1.default.join(dirEntry, await getFileTrueCase(path_1.default.join(from, dirEntry), path_1.default.relative(parts[0], relativePath)));
        }
    }
    // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
    throw new FileNotFoundError(path_1.default.join(from, relativePath));
}
exports.getFileTrueCase = getFileTrueCase;
/**
 * Sync version of getFileTrueCase
 *
 * @see getFileTrueCase
 */
function getFileTrueCaseSync(from, relativePath) {
    const dirEntries = readdirSync(from);
    const parts = relativePath.split(path_1.default.sep);
    const nextDirLowerCase = parts[0].toLowerCase();
    for (const dirEntry of dirEntries) {
        if (dirEntry.toLowerCase() === nextDirLowerCase) {
            if (parts.length === 1) {
                return dirEntry;
            }
            return path_1.default.join(dirEntry, getFileTrueCaseSync(path_1.default.join(from, dirEntry), path_1.default.relative(parts[0], relativePath)));
        }
    }
    // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
    throw new FileNotFoundError(path_1.default.join(from, relativePath));
}
exports.getFileTrueCaseSync = getFileTrueCaseSync;
async function readdir(absolutePathToDir) {
    try {
        return await promises_1.default.readdir(absolutePathToDir);
    }
    catch (e) {
        if (e.code === "ENOENT") {
            return [];
        }
        if (e.code === "ENOTDIR") {
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw new InvalidDirectoryError(absolutePathToDir, e);
        }
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw new FileSystemAccessError(e.message, e);
    }
}
function readdirSync(absolutePathToDir) {
    try {
        return fs_1.default.readdirSync(absolutePathToDir);
    }
    catch (e) {
        if (e.code === "ENOENT") {
            return [];
        }
        if (e.code === "ENOTDIR") {
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw new InvalidDirectoryError(absolutePathToDir, e);
        }
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw new FileSystemAccessError(e.message, e);
    }
}
//# sourceMappingURL=fs-utils.js.map