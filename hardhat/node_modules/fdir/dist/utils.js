"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.cleanPath = cleanPath;
exports.convertSlashes = convertSlashes;
exports.isRootDirectory = isRootDirectory;
exports.normalizePath = normalizePath;
const path_1 = require("path");
function cleanPath(path) {
    let normalized = (0, path_1.normalize)(path);
    // we have to remove the last path separator
    // to account for / root path
    if (normalized.length > 1 && normalized[normalized.length - 1] === path_1.sep)
        normalized = normalized.substring(0, normalized.length - 1);
    return normalized;
}
const SLASHES_REGEX = /[\\/]/g;
function convertSlashes(path, separator) {
    return path.replace(SLASHES_REGEX, separator);
}
const WINDOWS_ROOT_DIR_REGEX = /^[a-z]:[\\/]$/i;
function isRootDirectory(path) {
    return path === "/" || WINDOWS_ROOT_DIR_REGEX.test(path);
}
function normalizePath(path, options) {
    const { resolvePaths, normalizePath, pathSeparator } = options;
    const pathNeedsCleaning = (process.platform === "win32" && path.includes("/")) ||
        path.startsWith(".");
    if (resolvePaths)
        path = (0, path_1.resolve)(path);
    if (normalizePath || pathNeedsCleaning)
        path = cleanPath(path);
    if (path === ".")
        return "";
    const needsSeperator = path[path.length - 1] !== pathSeparator;
    return convertSlashes(needsSeperator ? path + pathSeparator : path, pathSeparator);
}
