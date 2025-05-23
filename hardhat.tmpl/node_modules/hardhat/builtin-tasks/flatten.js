"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const picocolors_1 = __importDefault(require("picocolors"));
const node_fs_1 = require("node:fs");
const config_env_1 = require("../internal/core/config/config-env");
const errors_1 = require("../internal/core/errors");
const errors_list_1 = require("../internal/core/errors-list");
const packageInfo_1 = require("../internal/util/packageInfo");
const fs_utils_1 = require("../internal/util/fs-utils");
const task_names_1 = require("./task-names");
// Match every group where a SPDX license is defined. The first captured group is the license.
const SPDX_LICENSES_REGEX = /^(?:\/\/|\/\*)\s*SPDX-License-Identifier:\s*([a-zA-Z\d+.-]+).*/gm;
// Match every group where a pragma directive is defined. The first captured group is the pragma directive.
const PRAGMA_DIRECTIVES_REGEX = /^(?: |\t)*(pragma\s*abicoder\s*v(1|2)|pragma\s*experimental\s*ABIEncoderV2)\s*;/gim;
function getSortedFiles(dependenciesGraph) {
    const tsort = require("tsort");
    const graph = tsort();
    // sort the graph entries to make the results deterministic
    const dependencies = dependenciesGraph
        .entries()
        .sort(([a], [b]) => a.sourceName.localeCompare(b.sourceName));
    const filesMap = {};
    const resolvedFiles = dependencies.map(([file, _deps]) => file);
    resolvedFiles.forEach((f) => (filesMap[f.sourceName] = f));
    for (const [from, deps] of dependencies) {
        // sort the dependencies to make the results deterministic
        const sortedDeps = [...deps].sort((a, b) => a.sourceName.localeCompare(b.sourceName));
        for (const to of sortedDeps) {
            graph.add(to.sourceName, from.sourceName);
        }
    }
    try {
        const topologicalSortedNames = graph.sort();
        // If an entry has no dependency it won't be included in the graph, so we
        // add them and then dedup the array
        const withEntries = topologicalSortedNames.concat(resolvedFiles.map((f) => f.sourceName));
        const sortedNames = [...new Set(withEntries)];
        return sortedNames.map((n) => filesMap[n]);
    }
    catch (error) {
        if (error instanceof Error) {
            if (error.toString().includes("Error: There is a cycle in the graph.")) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.BUILTIN_TASKS.FLATTEN_CYCLE, {}, error);
            }
        }
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw error;
    }
}
function getFileWithoutImports(resolvedFile) {
    const IMPORT_SOLIDITY_REGEX = /^\s*import(\s+)[\s\S]*?;\s*$/gm;
    return resolvedFile.content.rawContent
        .replace(IMPORT_SOLIDITY_REGEX, "")
        .trim();
}
function getLicensesInfo(sortedFiles) {
    const licenses = new Set();
    const filesWithoutLicenses = new Set();
    for (const file of sortedFiles) {
        const matches = [...file.content.rawContent.matchAll(SPDX_LICENSES_REGEX)];
        if (matches.length === 0) {
            filesWithoutLicenses.add(file.sourceName);
            continue;
        }
        for (const groups of matches) {
            licenses.add(groups[1]);
        }
    }
    // Sort alphabetically
    return [Array.from(licenses).sort(), Array.from(filesWithoutLicenses).sort()];
}
function getLicensesHeader(licenses) {
    return licenses.length <= 0
        ? ""
        : `\n\n// SPDX-License-Identifier: ${licenses.join(" AND ")}`;
}
function removeUnnecessarySpaces(str) {
    return str.replace(/\s+/g, " ").trim();
}
function getPragmaAbicoderDirectiveInfo(sortedFiles) {
    let directive = "";
    const directivesByImportance = [
        "pragma abicoder v1",
        "pragma experimental ABIEncoderV2",
        "pragma abicoder v2",
    ];
    const filesWithoutPragmaDirectives = new Set();
    const filesWithMostImportantDirective = []; // Every array element has the structure: [ fileName, fileMostImportantDirective ]
    for (const file of sortedFiles) {
        const matches = [
            ...file.content.rawContent.matchAll(PRAGMA_DIRECTIVES_REGEX),
        ];
        if (matches.length === 0) {
            filesWithoutPragmaDirectives.add(file.sourceName);
            continue;
        }
        let fileMostImportantDirective = "";
        for (const groups of matches) {
            const normalizedPragma = removeUnnecessarySpaces(groups[1]);
            // Update the most important pragma directive among all the files
            if (directivesByImportance.indexOf(normalizedPragma) >
                directivesByImportance.indexOf(directive)) {
                directive = normalizedPragma;
            }
            // Update the most important pragma directive for the current file
            if (directivesByImportance.indexOf(normalizedPragma) >
                directivesByImportance.indexOf(fileMostImportantDirective)) {
                fileMostImportantDirective = normalizedPragma;
            }
        }
        // Add in the array the most important directive for the current file
        filesWithMostImportantDirective.push([
            file.sourceName,
            fileMostImportantDirective,
        ]);
    }
    // Add to the array the files that have a pragma directive which is not the same as the main one that
    // is going to be used in the flatten file
    const filesWithDifferentPragmaDirectives = filesWithMostImportantDirective
        .filter(([, fileDirective]) => fileDirective !== directive)
        .map(([fileName]) => fileName);
    // Sort alphabetically
    return [
        directive,
        Array.from(filesWithoutPragmaDirectives).sort(),
        filesWithDifferentPragmaDirectives.sort(),
    ];
}
function getPragmaAbicoderDirectiveHeader(pragmaDirective) {
    return pragmaDirective === "" ? "" : `\n\n${pragmaDirective};`;
}
function replaceLicenses(file) {
    return file.replaceAll(SPDX_LICENSES_REGEX, (...groups) => `// Original license: SPDX_License_Identifier: ${groups[1]}`);
}
function replacePragmaAbicoderDirectives(file) {
    return file.replaceAll(PRAGMA_DIRECTIVES_REGEX, (...groups) => {
        return `// Original pragma directive: ${removeUnnecessarySpaces(groups[1])}`;
    });
}
(0, config_env_1.subtask)(task_names_1.TASK_FLATTEN_GET_FLATTENED_SOURCE_AND_METADATA, "Returns all contracts and their dependencies flattened. Also return metadata about pragma directives and SPDX licenses")
    .addOptionalParam("files", undefined, undefined, config_env_1.types.any)
    .setAction(async ({ files }, { run }) => {
    const dependencyGraph = await run(task_names_1.TASK_FLATTEN_GET_DEPENDENCY_GRAPH, { files });
    let flattened = "";
    if (dependencyGraph.getResolvedFiles().length === 0) {
        return [flattened, null];
    }
    const packageJson = await (0, packageInfo_1.getPackageJson)();
    flattened += `// Sources flattened with hardhat v${packageJson.version} https://hardhat.org`;
    const sortedFiles = getSortedFiles(dependencyGraph);
    const [licenses, filesWithoutLicenses] = getLicensesInfo(sortedFiles);
    const [pragmaDirective, filesWithoutPragmaDirectives, filesWithDifferentPragmaDirectives,] = getPragmaAbicoderDirectiveInfo(sortedFiles);
    flattened += getLicensesHeader(licenses);
    flattened += getPragmaAbicoderDirectiveHeader(pragmaDirective);
    for (const file of sortedFiles) {
        let tmpFile = getFileWithoutImports(file);
        tmpFile = replaceLicenses(tmpFile);
        tmpFile = replacePragmaAbicoderDirectives(tmpFile);
        flattened += `\n\n// File ${file.getVersionedName()}\n`;
        flattened += `\n${tmpFile}\n`;
    }
    return [
        flattened.trim(),
        {
            filesWithoutLicenses,
            pragmaDirective,
            filesWithoutPragmaDirectives,
            filesWithDifferentPragmaDirectives,
        },
    ];
});
// The following task is kept for backwards-compatibility reasons
(0, config_env_1.subtask)(task_names_1.TASK_FLATTEN_GET_FLATTENED_SOURCE, "Returns all contracts and their dependencies flattened")
    .addOptionalParam("files", undefined, undefined, config_env_1.types.any)
    .setAction(async ({ files }, { run }) => {
    return (await run(task_names_1.TASK_FLATTEN_GET_FLATTENED_SOURCE_AND_METADATA, { files }))[0];
});
(0, config_env_1.subtask)(task_names_1.TASK_FLATTEN_GET_DEPENDENCY_GRAPH)
    .addOptionalParam("files", undefined, undefined, config_env_1.types.any)
    .setAction(async ({ files }, { run }) => {
    const sourcePaths = files === undefined
        ? await run(task_names_1.TASK_COMPILE_SOLIDITY_GET_SOURCE_PATHS)
        : files.map((f) => (0, fs_utils_1.getRealPathSync)(f));
    const sourceNames = await run(task_names_1.TASK_COMPILE_SOLIDITY_GET_SOURCE_NAMES, {
        sourcePaths,
    });
    const dependencyGraph = await run(task_names_1.TASK_COMPILE_SOLIDITY_GET_DEPENDENCY_GRAPH, { sourceNames });
    return dependencyGraph;
});
(0, config_env_1.task)(task_names_1.TASK_FLATTEN, "Flattens and prints contracts and their dependencies. If no file is passed, all the contracts in the project will be flattened.")
    .addOptionalVariadicPositionalParam("files", "The files to flatten", undefined, config_env_1.types.inputFile)
    .addOptionalParam("output", "The output file containing the flattened contracts", undefined, config_env_1.types.string)
    .setAction(async ({ files, output, }, { run }) => {
    const [flattenedFile, metadata] = await run(task_names_1.TASK_FLATTEN_GET_FLATTENED_SOURCE_AND_METADATA, { files });
    if (output !== undefined) {
        (0, node_fs_1.writeFileSync)(output, flattenedFile, { encoding: "utf-8" });
    }
    else {
        console.log(flattenedFile);
    }
    if (metadata === null)
        return;
    if (metadata.filesWithoutLicenses.length > 0) {
        console.warn(picocolors_1.default.yellow(`\nThe following file(s) do NOT specify SPDX licenses: ${metadata.filesWithoutLicenses.join(", ")}`));
    }
    if (metadata.pragmaDirective !== "" &&
        metadata.filesWithoutPragmaDirectives.length > 0) {
        console.warn(picocolors_1.default.yellow(`\nPragma abicoder directives are defined in some files, but they are not defined in the following ones: ${metadata.filesWithoutPragmaDirectives.join(", ")}`));
    }
    if (metadata.filesWithDifferentPragmaDirectives.length > 0) {
        console.warn(picocolors_1.default.yellow(`\nThe flattened file is using the pragma abicoder directive '${metadata.pragmaDirective}' but these files have a different pragma abicoder directive: ${metadata.filesWithDifferentPragmaDirectives.join(", ")}`));
    }
});
//# sourceMappingURL=flatten.js.map