#!/usr/bin/env node
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
const commander = __importStar(require("commander"));
const fs = __importStar(require("fs"));
const os = __importStar(require("os"));
const path = __importStar(require("path"));
const index_1 = __importDefault(require("./index"));
const smtchecker_1 = __importDefault(require("./smtchecker"));
const smtsolver_1 = __importDefault(require("./smtsolver"));
// hold on to any exception handlers that existed prior to this script running, we'll be adding them back at the end
const originalUncaughtExceptionListeners = process.listeners('uncaughtException');
// FIXME: remove annoying exception catcher of Emscripten
//        see https://github.com/chriseth/browser-solidity/issues/167
process.removeAllListeners('uncaughtException');
const program = new commander.Command();
const commanderParseInt = function (value) {
    const parsedValue = parseInt(value, 10);
    if (isNaN(parsedValue)) {
        throw new commander.InvalidArgumentError('Not a valid integer.');
    }
    return parsedValue;
};
program.name('solcjs');
program.version(index_1.default.version());
program
    .option('--version', 'Show version and exit.')
    .option('--optimize', 'Enable bytecode optimizer.', false)
    .option('--optimize-runs <optimize-runs>', 'The number of runs specifies roughly how often each opcode of the deployed code will be executed across the lifetime of the contract. ' +
    'Lower values will optimize more for initial deployment cost, higher values will optimize more for high-frequency usage.', commanderParseInt)
    .option('--bin', 'Binary of the contracts in hex.')
    .option('--abi', 'ABI of the contracts.')
    .option('--standard-json', 'Turn on Standard JSON Input / Output mode.')
    .option('--base-path <path>', 'Root of the project source tree. ' +
    'The import callback will attempt to interpret all import paths as relative to this directory.')
    .option('--include-path <path...>', 'Extra source directories available to the import callback. ' +
    'When using a package manager to install libraries, use this option to specify directories where packages are installed. ' +
    'Can be used multiple times to provide multiple locations.')
    .option('-o, --output-dir <output-directory>', 'Output directory for the contracts.')
    .option('-p, --pretty-json', 'Pretty-print all JSON output.', false)
    .option('-v, --verbose', 'More detailed console output.', false);
program.parse(process.argv);
const options = program.opts();
const files = program.args;
const destination = options.outputDir || '.';
function abort(msg) {
    console.error(msg || 'Error occurred');
    process.exit(1);
}
function readFileCallback(sourcePath) {
    const prefixes = [options.basePath ? options.basePath : ''].concat(options.includePath ? options.includePath : []);
    for (const prefix of prefixes) {
        const prefixedSourcePath = (prefix ? prefix + '/' : '') + sourcePath;
        if (fs.existsSync(prefixedSourcePath)) {
            try {
                return { contents: fs.readFileSync(prefixedSourcePath).toString('utf8') };
            }
            catch (e) {
                return { error: 'Error reading ' + prefixedSourcePath + ': ' + e };
            }
        }
    }
    return { error: 'File not found inside the base path or any of the include paths.' };
}
function withUnixPathSeparators(filePath) {
    // On UNIX-like systems forward slashes in paths are just a part of the file name.
    if (os.platform() !== 'win32') {
        return filePath;
    }
    return filePath.replace(/\\/g, '/');
}
function makeSourcePathRelativeIfPossible(sourcePath) {
    const absoluteBasePath = (options.basePath ? path.resolve(options.basePath) : path.resolve('.'));
    const absoluteIncludePaths = (options.includePath
        ? options.includePath.map((prefix) => { return path.resolve(prefix); })
        : []);
    // Compared to base path stripping logic in solc this is much simpler because path.resolve()
    // handles symlinks correctly (does not resolve them except in work dir) and strips .. segments
    // from paths going beyond root (e.g. `/../../a/b/c` -> `/a/b/c/`). It's simpler also because it
    // ignores less important corner cases: drive letters are not stripped from absolute paths on
    // Windows and UNC paths are not handled in a special way (at least on Linux). Finally, it has
    // very little test coverage so there might be more differences that we are just not aware of.
    const absoluteSourcePath = path.resolve(sourcePath);
    for (const absolutePrefix of [absoluteBasePath].concat(absoluteIncludePaths)) {
        const relativeSourcePath = path.relative(absolutePrefix, absoluteSourcePath);
        if (!relativeSourcePath.startsWith('../')) {
            return withUnixPathSeparators(relativeSourcePath);
        }
    }
    // File is not located inside base path or include paths so use its absolute path.
    return withUnixPathSeparators(absoluteSourcePath);
}
function toFormattedJson(input) {
    return JSON.stringify(input, null, program.prettyJson ? 4 : 0);
}
function reformatJsonIfRequested(inputJson) {
    return (program.prettyJson ? toFormattedJson(JSON.parse(inputJson)) : inputJson);
}
let callbacks;
if (options.basePath || !options.standardJson) {
    callbacks = { import: readFileCallback };
}
if (options.standardJson) {
    const input = fs.readFileSync(process.stdin.fd).toString('utf8');
    if (program.verbose) {
        console.log('>>> Compiling:\n' + reformatJsonIfRequested(input) + '\n');
    }
    let output = reformatJsonIfRequested(index_1.default.compile(input, callbacks));
    try {
        if (smtsolver_1.default.availableSolvers.length === 0) {
            console.log('>>> Cannot retry compilation with SMT because there are no SMT solvers available.');
        }
        else {
            const inputJSON = smtchecker_1.default.handleSMTQueries(JSON.parse(input), JSON.parse(output), smtsolver_1.default.smtSolver, smtsolver_1.default.availableSolvers[0]);
            if (inputJSON) {
                if (program.verbose) {
                    console.log('>>> Retrying compilation with SMT:\n' + toFormattedJson(inputJSON) + '\n');
                }
                output = reformatJsonIfRequested(index_1.default.compile(JSON.stringify(inputJSON), callbacks));
            }
        }
    }
    catch (e) {
        const addError = {
            component: 'general',
            formattedMessage: e.toString(),
            message: e.toString(),
            type: 'Warning'
        };
        const outputJSON = JSON.parse(output);
        if (!outputJSON.errors) {
            outputJSON.errors = [];
        }
        outputJSON.errors.push(addError);
        output = toFormattedJson(outputJSON);
    }
    if (program.verbose) {
        console.log('>>> Compilation result:');
    }
    console.log(output);
    process.exit(0);
}
else if (files.length === 0) {
    console.error('Must provide a file');
    process.exit(1);
}
if (!(options.bin || options.abi)) {
    abort('Invalid option selected, must specify either --bin or --abi');
}
if (!options.basePath && options.includePath && options.includePath.length > 0) {
    abort('--include-path option requires a non-empty base path.');
}
if (options.includePath) {
    for (const includePath of options.includePath) {
        if (!includePath) {
            abort('Empty values are not allowed in --include-path.');
        }
    }
}
const sources = {};
for (let i = 0; i < files.length; i++) {
    try {
        sources[makeSourcePathRelativeIfPossible(files[i])] = {
            content: fs.readFileSync(files[i]).toString()
        };
    }
    catch (e) {
        abort('Error reading ' + files[i] + ': ' + e);
    }
}
const cliInput = {
    language: 'Solidity',
    settings: {
        optimizer: {
            enabled: options.optimize,
            runs: options.optimizeRuns
        },
        outputSelection: {
            '*': {
                '*': ['abi', 'evm.bytecode']
            }
        }
    },
    sources: sources
};
if (program.verbose) {
    console.log('>>> Compiling:\n' + toFormattedJson(cliInput) + '\n');
}
const output = JSON.parse(index_1.default.compile(JSON.stringify(cliInput), callbacks));
let hasError = false;
if (!output) {
    abort('No output from compiler');
}
else if (output.errors) {
    for (const error in output.errors) {
        const message = output.errors[error];
        if (message.severity === 'warning') {
            console.log(message.formattedMessage);
        }
        else {
            console.error(message.formattedMessage);
            hasError = true;
        }
    }
}
fs.mkdirSync(destination, { recursive: true });
function writeFile(file, content) {
    file = path.join(destination, file);
    fs.writeFile(file, content, function (err) {
        if (err) {
            console.error('Failed to write ' + file + ': ' + err);
        }
    });
}
for (const fileName in output.contracts) {
    for (const contractName in output.contracts[fileName]) {
        let contractFileName = fileName + ':' + contractName;
        contractFileName = contractFileName.replace(/[:./\\]/g, '_');
        if (options.bin) {
            writeFile(contractFileName + '.bin', output.contracts[fileName][contractName].evm.bytecode.object);
        }
        if (options.abi) {
            writeFile(contractFileName + '.abi', toFormattedJson(output.contracts[fileName][contractName].abi));
        }
    }
}
// Put back original exception handlers.
originalUncaughtExceptionListeners.forEach(function (listener) {
    process.addListener('uncaughtException', listener);
});
if (hasError) {
    process.exit(1);
}
