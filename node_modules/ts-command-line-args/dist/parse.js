"use strict";
var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
var __rest = (this && this.__rest) || function (s, e) {
    var t = {};
    for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p) && e.indexOf(p) < 0)
        t[p] = s[p];
    if (s != null && typeof Object.getOwnPropertySymbols === "function")
        for (var i = 0, p = Object.getOwnPropertySymbols(s); i < p.length; i++) {
            if (e.indexOf(p[i]) < 0 && Object.prototype.propertyIsEnumerable.call(s, p[i]))
                t[p[i]] = s[p[i]];
        }
    return t;
};
var __spreadArray = (this && this.__spreadArray) || function (to, from) {
    for (var i = 0, il = from.length, j = to.length; i < il; i++, j++)
        to[j] = from[i];
    return to;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.parse = void 0;
var command_line_args_1 = __importDefault(require("command-line-args"));
var command_line_usage_1 = __importDefault(require("command-line-usage"));
var helpers_1 = require("./helpers");
var options_helper_1 = require("./helpers/options.helper");
var string_helper_1 = require("./helpers/string.helper");
var fs_1 = require("fs");
var path_1 = require("path");
/**
 * parses command line arguments and returns an object with all the arguments in IF all required options passed
 * @param config the argument config. Required, used to determine what arguments are expected
 * @param options
 * @param exitProcess defaults to true. The process will exit if any required arguments are omitted
 * @param addCommandLineResults defaults to false. If passed an additional _commandLineResults object will be returned in the result
 * @returns
 */
function parse(config, options, exitProcess, addCommandLineResults) {
    if (options === void 0) { options = {}; }
    if (exitProcess === void 0) { exitProcess = true; }
    options = options || {};
    var argsWithBooleanValues = options.argv || process.argv.slice(2);
    var logger = options.logger || console;
    var normalisedConfig = helpers_1.normaliseConfig(config);
    options.argv = helpers_1.removeBooleanValues(argsWithBooleanValues, normalisedConfig);
    var optionList = helpers_1.createCommandLineConfig(normalisedConfig);
    var parsedArgs = command_line_args_1.default(optionList, options);
    if (parsedArgs['_all'] != null) {
        var unknown = parsedArgs['_unknown'];
        parsedArgs = parsedArgs['_all'];
        if (unknown) {
            parsedArgs['_unknown'] = unknown;
        }
    }
    var booleanValues = helpers_1.getBooleanValues(argsWithBooleanValues, normalisedConfig);
    parsedArgs = __assign(__assign({}, parsedArgs), booleanValues);
    if (options.loadFromFileArg != null && parsedArgs[options.loadFromFileArg] != null) {
        var configFromFile = JSON.parse(fs_1.readFileSync(path_1.resolve(parsedArgs[options.loadFromFileArg])).toString());
        var parsedArgsWithoutDefaults = command_line_args_1.default(
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        optionList.map(function (_a) {
            var defaultValue = _a.defaultValue, option = __rest(_a, ["defaultValue"]);
            return (__assign({}, option));
        }), options);
        parsedArgs = helpers_1.mergeConfig(parsedArgs, __assign(__assign({}, parsedArgsWithoutDefaults), booleanValues), configFromFile, normalisedConfig, options.loadFromFileJsonPathArg);
    }
    var missingArgs = listMissingArgs(optionList, parsedArgs);
    if (options.helpArg != null && parsedArgs[options.helpArg]) {
        printHelpGuide(options, optionList, logger);
        if (exitProcess) {
            return process.exit(resolveExitCode(options, 'usageGuide', parsedArgs, missingArgs));
        }
    }
    else if (missingArgs.length > 0) {
        if (options.showHelpWhenArgsMissing) {
            var missingArgsHeader = typeof options.helpWhenArgMissingHeader === 'function'
                ? options.helpWhenArgMissingHeader(missingArgs)
                : options.helpWhenArgMissingHeader;
            var additionalHeaderSections = missingArgsHeader != null ? [missingArgsHeader] : [];
            printHelpGuide(options, optionList, logger, additionalHeaderSections);
        }
        else if (options.hideMissingArgMessages !== true) {
            printMissingArgErrors(missingArgs, logger, options.baseCommand);
            printUsageGuideMessage(__assign(__assign({}, options), { logger: logger }), options.helpArg != null ? optionList.filter(function (option) { return option.name === options.helpArg; })[0] : undefined);
        }
    }
    var _commandLineResults = {
        missingArgs: missingArgs,
        printHelp: function () { return printHelpGuide(options, optionList, logger); },
    };
    if (missingArgs.length > 0 && exitProcess) {
        process.exit(resolveExitCode(options, 'missingArgs', parsedArgs, missingArgs));
    }
    else {
        if (addCommandLineResults) {
            parsedArgs = __assign(__assign({}, parsedArgs), { _commandLineResults: _commandLineResults });
        }
        return parsedArgs;
    }
}
exports.parse = parse;
function resolveExitCode(options, reason, passedArgs, missingArgs) {
    switch (typeof options.processExitCode) {
        case 'number':
            return options.processExitCode;
        case 'function':
            return options.processExitCode(reason, passedArgs, missingArgs);
        default:
            return 0;
    }
}
function printHelpGuide(options, optionList, logger, additionalHeaderSections) {
    var _a, _b;
    if (additionalHeaderSections === void 0) { additionalHeaderSections = []; }
    var sections = __spreadArray(__spreadArray(__spreadArray(__spreadArray(__spreadArray([], additionalHeaderSections), (((_a = options.headerContentSections) === null || _a === void 0 ? void 0 : _a.filter(filterCliSections)) || [])), options_helper_1.getOptionSections(options).map(function (option) { return options_helper_1.addOptions(option, optionList, options); })), options_helper_1.getOptionFooterSection(optionList, options)), (((_b = options.footerContentSections) === null || _b === void 0 ? void 0 : _b.filter(filterCliSections)) || []));
    helpers_1.visit(sections, function (value) {
        switch (typeof value) {
            case 'string':
                return string_helper_1.removeAdditionalFormatting(value);
            default:
                return value;
        }
    });
    var usageGuide = command_line_usage_1.default(sections);
    logger.log(usageGuide);
}
function filterCliSections(section) {
    return section.includeIn == null || section.includeIn === 'both' || section.includeIn === 'cli';
}
function printMissingArgErrors(missingArgs, logger, baseCommand) {
    baseCommand = baseCommand ? baseCommand + " " : "";
    missingArgs.forEach(function (config) {
        var aliasMessage = config.alias != null ? " or '" + baseCommand + "-" + config.alias + " passedValue'" : "";
        var runCommand = baseCommand !== ''
            ? "running '" + baseCommand + "--" + config.name + "=passedValue'" + aliasMessage
            : "passing '--" + config.name + "=passedValue'" + aliasMessage + " in command line arguments";
        logger.error("Required parameter '" + config.name + "' was not passed. Please provide a value by " + runCommand);
    });
}
function printUsageGuideMessage(options, helpParam) {
    if (helpParam != null) {
        var helpArg = helpParam.alias != null ? "-" + helpParam.alias : "--" + helpParam.name;
        var command = options.baseCommand != null ? "run '" + options.baseCommand + " " + helpArg + "'" : "pass '" + helpArg + "'";
        options.logger.log("To view the help guide " + command);
    }
}
function listMissingArgs(commandLineConfig, parsedArgs) {
    return commandLineConfig
        .filter(function (config) { return config.optional == null && parsedArgs[config.name] == null; })
        .filter(function (config) {
        if (config.type.name === 'Boolean') {
            parsedArgs[config.name] = false;
            return false;
        }
        return true;
    });
}
