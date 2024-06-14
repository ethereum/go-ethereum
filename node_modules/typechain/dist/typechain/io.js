"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.skipEmptyAbis = exports.loadFileDescriptions = exports.processOutput = void 0;
const fs_1 = require("fs");
const lodash_1 = require("lodash");
const path_1 = require("path");
const outputTransformers_1 = require("../codegen/outputTransformers");
const abiParser_1 = require("../parser/abiParser");
const debug_1 = require("../utils/debug");
function processOutput(services, cfg, output) {
    const { fs, mkdirp } = services;
    if (!output) {
        return 0;
    }
    const outputFds = (0, lodash_1.isArray)(output) ? output : [output];
    outputFds.forEach((fd) => {
        // ensure directory first
        mkdirp((0, path_1.dirname)(fd.path));
        const finalOutput = outputTransformers_1.outputTransformers.reduce((content, transformer) => transformer(content, services, cfg), fd.contents);
        (0, debug_1.debug)(`Writing file: ${(0, path_1.relative)(cfg.cwd, fd.path)}`);
        fs.writeFileSync(fd.path, finalOutput, 'utf8');
    });
    return outputFds.length;
}
exports.processOutput = processOutput;
function loadFileDescriptions(services, files) {
    const fileDescriptions = files.map((path) => ({
        path,
        contents: services.fs.readFileSync(path, 'utf8'),
    }));
    return fileDescriptions;
}
exports.loadFileDescriptions = loadFileDescriptions;
function skipEmptyAbis(paths) {
    const notEmptyAbis = paths
        .map((p) => ({ path: p, contents: (0, fs_1.readFileSync)(p, 'utf-8') }))
        .filter((fd) => (0, abiParser_1.extractAbi)(fd.contents).length !== 0);
    return notEmptyAbis.map((p) => p.path);
}
exports.skipEmptyAbis = skipEmptyAbis;
//# sourceMappingURL=io.js.map