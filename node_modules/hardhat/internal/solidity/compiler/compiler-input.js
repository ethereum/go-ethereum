"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getInputFromCompilationJob = void 0;
function getInputFromCompilationJob(compilationJob) {
    const sources = {};
    // we sort the files so that we always get the same compilation input
    const resolvedFiles = compilationJob
        .getResolvedFiles()
        .sort((a, b) => a.sourceName.localeCompare(b.sourceName));
    for (const file of resolvedFiles) {
        sources[file.sourceName] = {
            content: file.content.rawContent,
        };
    }
    const { settings } = compilationJob.getSolcConfig();
    return {
        language: "Solidity",
        sources,
        settings,
    };
}
exports.getInputFromCompilationJob = getInputFromCompilationJob;
//# sourceMappingURL=compiler-input.js.map