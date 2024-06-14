import { CompilationJob, CompilerInput } from "../../../types";

export function getInputFromCompilationJob(
  compilationJob: CompilationJob
): CompilerInput {
  const sources: { [sourceName: string]: { content: string } } = {};

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
