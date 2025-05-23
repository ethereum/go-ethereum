"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
const memorystream_1 = __importDefault(require("memorystream"));
const follow_redirects_1 = require("follow-redirects");
const formatters_1 = require("./formatters");
const helpers_1 = require("./common/helpers");
const bindings_1 = __importDefault(require("./bindings"));
const translate_1 = __importDefault(require("./translate"));
const Module = module.constructor;
function wrapper(soljson) {
    const { coreBindings, compileBindings, methodFlags } = (0, bindings_1.default)(soljson);
    return {
        version: coreBindings.version,
        semver: coreBindings.versionToSemver,
        license: coreBindings.license,
        lowlevel: {
            compileSingle: compileBindings.compileJson,
            compileMulti: compileBindings.compileJsonMulti,
            compileCallback: compileBindings.compileJsonCallback,
            compileStandard: compileBindings.compileStandard
        },
        features: {
            legacySingleInput: methodFlags.compileJsonStandardSupported,
            multipleInputs: methodFlags.compileJsonMultiSupported || methodFlags.compileJsonStandardSupported,
            importCallback: methodFlags.compileJsonCallbackSuppported || methodFlags.compileJsonStandardSupported,
            nativeStandardJSON: methodFlags.compileJsonStandardSupported
        },
        compile: compileStandardWrapper.bind(this, compileBindings),
        // Loads the compiler of the given version from the github repository
        // instead of from the local filesystem.
        loadRemoteVersion,
        // Use this if you want to add wrapper functions around the pure module.
        setupMethods: wrapper
    };
}
function loadRemoteVersion(versionString, callback) {
    const memoryStream = new memorystream_1.default(null, { readable: false });
    const url = `https://binaries.soliditylang.org/bin/soljson-${versionString}.js`;
    follow_redirects_1.https.get(url, response => {
        if (response.statusCode !== 200) {
            callback(new Error(`Error retrieving binary: ${response.statusMessage}`));
        }
        else {
            response.pipe(memoryStream);
            response.on('end', () => {
                // Based on the require-from-string package.
                const soljson = new Module();
                soljson._compile(memoryStream.toString(), `soljson-${versionString}.js`);
                if (module.parent && module.parent.children) {
                    // Make sure the module is plugged into the hierarchy correctly to have parent
                    // properly garbage collected.
                    module.parent.children.splice(module.parent.children.indexOf(soljson), 1);
                }
                callback(null, wrapper(soljson.exports));
            });
        }
    }).on('error', function (error) {
        callback(error);
    });
}
// Expects a Standard JSON I/O but supports old compilers
function compileStandardWrapper(compile, inputRaw, readCallback) {
    if (!(0, helpers_1.isNil)(compile.compileStandard)) {
        return compile.compileStandard(inputRaw, readCallback);
    }
    let input;
    try {
        input = JSON.parse(inputRaw);
    }
    catch (e) {
        return (0, formatters_1.formatFatalError)(`Invalid JSON supplied: ${e.message}`);
    }
    if (input.language !== 'Solidity') {
        return (0, formatters_1.formatFatalError)('Only "Solidity" is supported as a language.');
    }
    // NOTE: this is deliberately `== null`
    if ((0, helpers_1.isNil)(input.sources) || input.sources.length === 0) {
        return (0, formatters_1.formatFatalError)('No input sources specified.');
    }
    const sources = translateSources(input);
    const optimize = isOptimizerEnabled(input);
    const libraries = librariesSupplied(input);
    if ((0, helpers_1.isNil)(sources) || Object.keys(sources).length === 0) {
        return (0, formatters_1.formatFatalError)('Failed to process sources.');
    }
    // Try to wrap around old versions
    if (!(0, helpers_1.isNil)(compile.compileJsonCallback)) {
        const inputJson = JSON.stringify({ sources: sources });
        const output = compile.compileJsonCallback(inputJson, optimize, readCallback);
        return translateOutput(output, libraries);
    }
    if (!(0, helpers_1.isNil)(compile.compileJsonMulti)) {
        const output = compile.compileJsonMulti(JSON.stringify({ sources: sources }), optimize);
        return translateOutput(output, libraries);
    }
    // Try our luck with an ancient compiler
    if (!(0, helpers_1.isNil)(compile.compileJson)) {
        if (Object.keys(sources).length > 1) {
            return (0, formatters_1.formatFatalError)('Multiple sources provided, but compiler only supports single input.');
        }
        const input = sources[Object.keys(sources)[0]];
        const output = compile.compileJson(input, optimize);
        return translateOutput(output, libraries);
    }
    return (0, formatters_1.formatFatalError)('Compiler does not support any known interface.');
}
function isOptimizerEnabled(input) {
    return input.settings && input.settings.optimizer && input.settings.optimizer.enabled;
}
function translateSources(input) {
    const sources = {};
    for (const source in input.sources) {
        if (input.sources[source].content !== null) {
            sources[source] = input.sources[source].content;
        }
        else {
            // force failure
            return null;
        }
    }
    return sources;
}
function librariesSupplied(input) {
    if (!(0, helpers_1.isNil)(input.settings))
        return input.settings.libraries;
}
function translateOutput(outputRaw, libraries) {
    let parsedOutput;
    try {
        parsedOutput = JSON.parse(outputRaw);
    }
    catch (e) {
        return (0, formatters_1.formatFatalError)(`Compiler returned invalid JSON: ${e.message}`);
    }
    const output = translate_1.default.translateJsonCompilerOutput(parsedOutput, libraries);
    if ((0, helpers_1.isNil)(output)) {
        return (0, formatters_1.formatFatalError)('Failed to process output.');
    }
    return JSON.stringify(output);
}
module.exports = wrapper;
