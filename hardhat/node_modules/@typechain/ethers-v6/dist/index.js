"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/* eslint-disable import/no-extraneous-dependencies */
const fs_1 = require("fs");
const lodash_1 = require("lodash");
const path_1 = require("path");
const typechain_1 = require("typechain");
const codegen_1 = require("./codegen");
const hardhat_1 = require("./codegen/hardhat");
const common_1 = require("./common");
const DEFAULT_OUT_PATH = './types/ethers-contracts/';
class Ethers extends typechain_1.TypeChainTarget {
    constructor(config) {
        super(config);
        this.name = 'Ethers';
        this.contractsWithoutBytecode = {};
        this.bytecodeCache = {};
        panicOnOldTypeScriptVersion();
        const { cwd, outDir, allFiles } = config;
        this.allFiles = allFiles
            .map((p) => (0, typechain_1.shortenFullJsonFilePath)(p, allFiles))
            .map((p) => (0, path_1.relative)(this.cfg.inputDir, p))
            .map(typechain_1.normalizeSlashes);
        this.outDirAbs = (0, path_1.resolve)(cwd, outDir || DEFAULT_OUT_PATH);
    }
    transformFile(file) {
        const fileExt = (0, typechain_1.getFileExtension)(file.path);
        // For json files with both ABI and bytecode, both the contract typing and factory can be
        // generated at once. For split files (.abi and .bin) we don't know in which order they will
        // be transformed -- so we temporarily store whichever comes first, and generate the factory
        // only when both ABI and bytecode are present.
        if (fileExt === '.bin') {
            return this.transformBinFile(file);
        }
        return this.transformAbiOrFullJsonFile(file);
    }
    transformBinFile(file) {
        const bytecode = (0, typechain_1.extractBytecode)(file.contents);
        if (!bytecode) {
            return;
        }
        if (this.contractsWithoutBytecode[file.path]) {
            const { contract, abi } = this.contractsWithoutBytecode[file.path];
            delete this.contractsWithoutBytecode[file.path];
            return [this.genContractFactoryFile(contract, abi, bytecode)];
        }
        else {
            this.bytecodeCache[file.path] = bytecode;
        }
    }
    transformAbiOrFullJsonFile(file) {
        const abi = (0, typechain_1.extractAbi)(file.contents);
        if (abi.length === 0) {
            return;
        }
        const documentation = (0, typechain_1.extractDocumentation)(file.contents);
        const path = (0, path_1.relative)(this.cfg.inputDir, (0, typechain_1.shortenFullJsonFilePath)(file.path, this.cfg.allFiles));
        const contract = (0, typechain_1.parse)(abi, path, documentation);
        const bytecode = (0, typechain_1.extractBytecode)(file.contents) || this.bytecodeCache[file.path];
        if (bytecode) {
            return [
                this.genContractTypingsFile(contract, this.cfg.flags),
                this.genContractFactoryFile(contract, abi, bytecode),
            ];
        }
        else {
            this.contractsWithoutBytecode[file.path] = { abi, contract };
            return [this.genContractTypingsFile(contract, this.cfg.flags)];
        }
    }
    genContractTypingsFile(contract, codegenConfig) {
        return {
            path: (0, path_1.join)(this.outDirAbs, ...contract.path, `${contract.name}.ts`),
            contents: (0, codegen_1.codegenContractTypings)(contract, codegenConfig),
        };
    }
    genContractFactoryFile(contract, abi, bytecode) {
        return {
            path: (0, path_1.join)(this.outDirAbs, 'factories', ...contract.path, `${contract.name}${common_1.FACTORY_POSTFIX}.ts`),
            contents: (0, codegen_1.codegenContractFactory)(this.cfg.flags, contract, abi, bytecode),
        };
    }
    afterRun() {
        // For each contract that doesn't have bytecode (it's either abstract, or only ABI was provided)
        // generate a simplified factory, that allows to interact with deployed contract instances.
        const moduleSuffix = this.cfg.flags.node16Modules ? '.js' : '';
        const abstractFactoryFiles = Object.keys(this.contractsWithoutBytecode).map((contractName) => {
            const { contract, abi } = this.contractsWithoutBytecode[contractName];
            return {
                path: (0, path_1.join)(this.outDirAbs, 'factories', ...contract.path, `${contract.name}${common_1.FACTORY_POSTFIX}.ts`),
                contents: (0, codegen_1.codegenAbstractContractFactory)(contract, abi, moduleSuffix),
            };
        });
        const common = {
            path: (0, path_1.join)(this.outDirAbs, 'common.ts'),
            contents: (0, fs_1.readFileSync)((0, path_1.join)(__dirname, '../static/common.ts'), 'utf-8'),
        };
        const allContracts = this.allFiles.map((path) => (0, typechain_1.normalizeName)((0, typechain_1.getFilename)(path)));
        const hardhatHelper = this.cfg.flags.environment === 'hardhat'
            ? { path: (0, path_1.join)(this.outDirAbs, 'hardhat.d.ts'), contents: (0, hardhat_1.generateHardhatHelper)(allContracts) }
            : undefined;
        const typesBarrels = (0, typechain_1.createBarrelFiles)(this.allFiles, { typeOnly: true, moduleSuffix });
        const factoriesBarrels = (0, typechain_1.createBarrelFiles)(this.allFiles.map((s) => `factories/${s}`), { typeOnly: false, postfix: common_1.FACTORY_POSTFIX, moduleSuffix });
        const allBarrels = typesBarrels.concat(factoriesBarrels);
        const [rootIndexes, otherBarrels] = (0, lodash_1.partition)(allBarrels, (fd) => fd.path === 'index.ts');
        const rootIndex = {
            path: (0, path_1.join)(this.outDirAbs, 'index.ts'),
            contents: createRootIndexContent(rootIndexes, this.allFiles, moduleSuffix),
        };
        const allFiles = (0, lodash_1.compact)([
            common,
            hardhatHelper,
            rootIndex,
            ...abstractFactoryFiles,
            ...otherBarrels.map((fd) => ({
                path: (0, path_1.join)(this.outDirAbs, fd.path),
                contents: fd.contents,
            })),
        ]);
        return allFiles;
    }
}
exports.default = Ethers;
// root index.ts reexports also from deeper paths
function createRootIndexContent(rootIndexes, paths, moduleSuffix = '') {
    const contracts = paths.map(typechain_1.parseContractPath);
    const rootReexports = (0, lodash_1.uniqBy)(Object.values(contracts), (c) => c.name).flatMap((c) => {
        const path = c.path.length === 0 ? c.name : `${c.path.join('/')}/${c.name}`;
        return [
            `export type { ${c.name} } from './${path}${moduleSuffix}';`,
            `export { ${c.name}${common_1.FACTORY_POSTFIX} } from './factories/${path}${common_1.FACTORY_POSTFIX}${moduleSuffix}';`,
        ];
    });
    const rootIndexContent = new Set([
        ...rootIndexes[0].contents.split('\n'),
        ...rootIndexes[1].contents.split('\n'),
        ...rootReexports,
    ]);
    return [...rootIndexContent].join('\n');
}
function panicOnOldTypeScriptVersion() {
    const requiredTSVersion = { major: 4, minor: 3 };
    const targetEthersVersion = require('../package.json').version;
    let major;
    let minor;
    try {
        const { version } = require('typescript');
        [major, minor] = version.split('.').map(Number);
        if (major > requiredTSVersion.major || (major === requiredTSVersion.major && minor >= requiredTSVersion.minor)) {
            // the user has current version
            return;
        }
    }
    catch (err) {
        // we couldn't require `typescript`
        return;
    }
    const bold = (text) => `\x1b[1m${text}\x1b[0m`;
    const tsStr = `${requiredTSVersion.major}.${requiredTSVersion.minor}`;
    const errorMessage = `@typechain/ethers-v6 ${targetEthersVersion} needs TypeScript version ${tsStr} or newer.`;
    // eslint-disable-next-line no-console
    console.error(`
    ⚠  ${bold(errorMessage)}

    Generated code will cause syntax errors in older TypeScript versions.
    Your TypeScript version is ${major}.${minor}.
  `);
    throw new Error(errorMessage);
}
//# sourceMappingURL=index.js.map