"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.joinPathWithBasePath = joinPathWithBasePath;
exports.joinDirectoryPath = joinDirectoryPath;
exports.build = build;
const path_1 = require("path");
const utils_1 = require("../../utils");
function joinPathWithBasePath(filename, directoryPath) {
    return directoryPath + filename;
}
function joinPathWithRelativePath(root, options) {
    return function (filename, directoryPath) {
        const sameRoot = directoryPath.startsWith(root);
        if (sameRoot)
            return directoryPath.replace(root, "") + filename;
        else
            return ((0, utils_1.convertSlashes)((0, path_1.relative)(root, directoryPath), options.pathSeparator) +
                options.pathSeparator +
                filename);
    };
}
function joinPath(filename) {
    return filename;
}
function joinDirectoryPath(filename, directoryPath, separator) {
    return directoryPath + filename + separator;
}
function build(root, options) {
    const { relativePaths, includeBasePath } = options;
    return relativePaths && root
        ? joinPathWithRelativePath(root, options)
        : includeBasePath
            ? joinPathWithBasePath
            : joinPath;
}
