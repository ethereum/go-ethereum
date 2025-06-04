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
Object.defineProperty(exports, "__esModule", { value: true });
require("./type-extensions");
const task_names_1 = require("hardhat/builtin-tasks/task-names");
const config_1 = require("hardhat/config");
const contract_names_1 = require("hardhat/utils/contract-names");
const config_2 = require("./config");
const constants_1 = require("./constants");
const taskArgsStore = { noTypechain: false, fullRebuild: false };
(0, config_1.extendConfig)((config) => {
    config.typechain = (0, config_2.getDefaultTypechainConfig)(config);
});
(0, config_1.task)(task_names_1.TASK_COMPILE)
    .addFlag('noTypechain', 'Skip Typechain compilation')
    .setAction(async ({ noTypechain }, { config }, runSuper) => {
    // just save task arguments for later b/c there is no easier way to access them in subtask
    taskArgsStore.noTypechain = noTypechain || config.typechain.dontOverrideCompile;
    await runSuper();
});
(0, config_1.subtask)(task_names_1.TASK_COMPILE_SOLIDITY_COMPILE_JOBS, 'Compiles the entire project, building all artifacts').setAction(async (taskArgs, { run }, runSuper) => {
    const compileSolOutput = await runSuper(taskArgs);
    await run(constants_1.TASK_TYPECHAIN_GENERATE_TYPES, { compileSolOutput, quiet: taskArgs.quiet });
    return compileSolOutput;
});
(0, config_1.subtask)(constants_1.TASK_TYPECHAIN_GENERATE_TYPES)
    .addParam('compileSolOutput', 'Solidity compilation output', {}, config_1.types.any)
    .addFlag('quiet', 'Makes the process less verbose')
    .setAction(async ({ compileSolOutput, quiet }, { config, artifacts }) => {
    const artifactFQNs = getFQNamesFromCompilationOutput(compileSolOutput);
    const artifactPaths = Array.from(new Set(artifactFQNs.map((fqn) => artifacts.formArtifactPathFromFullyQualifiedName(fqn))));
    if (taskArgsStore.noTypechain) {
        return compileSolOutput;
    }
    // RUN TYPECHAIN TASK
    const typechainCfg = config.typechain;
    if (artifactPaths.length === 0 && !taskArgsStore.fullRebuild && !typechainCfg.externalArtifacts) {
        if (!quiet) {
            // eslint-disable-next-line no-console
            console.log('No need to generate any newer typings.');
        }
        return compileSolOutput;
    }
    // incremental generation is only supported in 'ethers-v5' and 'ethers-v6'
    // @todo: probably targets should specify somehow if then support incremental generation this won't work with custom targets
    const needsFullRebuild = taskArgsStore.fullRebuild || (typechainCfg.target !== 'ethers-v5' && typechainCfg.target !== 'ethers-v6');
    if (!quiet) {
        // eslint-disable-next-line no-console
        console.log(`Generating typings for: ${artifactPaths.length} artifacts in dir: ${typechainCfg.outDir} for target: ${typechainCfg.target}`);
    }
    const cwd = config.paths.root;
    const { glob } = await Promise.resolve().then(() => __importStar(require('typechain')));
    const allFiles = glob(cwd, [`${config.paths.artifacts}/!(build-info)/**/+([a-zA-Z0-9_]).json`]);
    if (typechainCfg.externalArtifacts) {
        allFiles.push(...glob(cwd, typechainCfg.externalArtifacts, false));
    }
    const typechainOptions = {
        cwd,
        allFiles,
        outDir: typechainCfg.outDir,
        target: typechainCfg.target,
        flags: {
            alwaysGenerateOverloads: typechainCfg.alwaysGenerateOverloads,
            discriminateTypes: typechainCfg.discriminateTypes,
            tsNocheck: typechainCfg.tsNocheck,
            environment: 'hardhat',
            node16Modules: typechainCfg.node16Modules,
        },
    };
    const { runTypeChain } = await Promise.resolve().then(() => __importStar(require('typechain')));
    const result = await runTypeChain({
        ...typechainOptions,
        filesToProcess: needsFullRebuild ? allFiles : glob(cwd, artifactPaths), // only process changed files if not doing full rebuild
    });
    if (!quiet) {
        // eslint-disable-next-line no-console
        console.log(`Successfully generated ${result.filesGenerated} typings!`);
    }
    // if this is not full rebuilding, always re-generate types for external artifacts
    if (!needsFullRebuild && typechainCfg.externalArtifacts) {
        const result = await runTypeChain({
            ...typechainOptions,
            filesToProcess: glob(cwd, typechainCfg.externalArtifacts, false), // only process files with external artifacts
        });
        if (!quiet) {
            // eslint-disable-next-line no-console
            console.log(`Successfully generated ${result.filesGenerated} typings for external artifacts!`);
        }
    }
});
(0, config_1.task)(constants_1.TASK_TYPECHAIN, 'Generate Typechain typings for compiled contracts').setAction(async (_, { run }) => {
    taskArgsStore.fullRebuild = true;
    await run(task_names_1.TASK_COMPILE, { quiet: true });
});
(0, config_1.task)(task_names_1.TASK_CLEAN, 'Clears the cache and deletes all artifacts', async ({ global }, { config }, runSuper) => {
    if (global) {
        return runSuper();
    }
    const fsExtra = await Promise.resolve().then(() => __importStar(require('fs-extra')));
    if (await fsExtra.pathExists(config.typechain.outDir)) {
        await fsExtra.remove(config.typechain.outDir);
    }
    await runSuper();
});
function getFQNamesFromCompilationOutput(compileSolOutput) {
    const allFQNNamesNested = compileSolOutput.artifactsEmittedPerJob.map((a) => {
        return a.artifactsEmittedPerFile.map((artifactPerFile) => {
            return artifactPerFile.artifactsEmitted.map((artifactName) => {
                return (0, contract_names_1.getFullyQualifiedName)(artifactPerFile.file.sourceName, artifactName);
            });
        });
    });
    return allFQNNamesNested.flat(2);
}
//# sourceMappingURL=index.js.map