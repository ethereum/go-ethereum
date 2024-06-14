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
var __spreadArray = (this && this.__spreadArray) || function (to, from) {
    for (var i = 0, il = from.length, j = to.length; i < il; i++, j++)
        to[j] = from[i];
    return to;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getBooleanValues = exports.removeBooleanValues = exports.mergeConfig = exports.normaliseConfig = exports.createCommandLineConfig = void 0;
var options_helper_1 = require("./options.helper");
function createCommandLineConfig(config) {
    return Object.keys(config).map(function (key) {
        var argConfig = config[key];
        var definition = typeof argConfig === 'object' ? argConfig : { type: argConfig };
        return __assign({ name: key }, definition);
    });
}
exports.createCommandLineConfig = createCommandLineConfig;
function normaliseConfig(config) {
    Object.keys(config).forEach(function (key) {
        var argConfig = config[key];
        config[key] = typeof argConfig === 'object' ? argConfig : { type: argConfig };
    });
    return config;
}
exports.normaliseConfig = normaliseConfig;
function mergeConfig(parsedConfig, parsedConfigWithoutDefaults, fileContent, options, jsonPath) {
    var configPath = jsonPath ? parsedConfig[jsonPath] : undefined;
    var configFromFile = resolveConfigFromFile(fileContent, configPath);
    if (configFromFile == null) {
        throw new Error("Could not resolve config object from specified file and path");
    }
    return __assign(__assign(__assign({}, parsedConfig), applyTypeConversion(configFromFile, options)), parsedConfigWithoutDefaults);
}
exports.mergeConfig = mergeConfig;
function resolveConfigFromFile(configfromFile, configPath) {
    if (configPath == null || configPath == '') {
        return configfromFile;
    }
    var paths = configPath.split('.');
    var key = paths.shift();
    if (key == null) {
        return configfromFile;
    }
    var config = configfromFile[key];
    return resolveConfigFromFile(config, paths.join('.'));
}
function applyTypeConversion(configfromFile, options) {
    var transformedParams = {};
    Object.keys(configfromFile).forEach(function (prop) {
        var key = prop;
        var argumentOptions = options[key];
        if (argumentOptions == null) {
            return;
        }
        var fileValue = configfromFile[key];
        if (argumentOptions.multiple || argumentOptions.lazyMultiple) {
            var fileArrayValue = Array.isArray(fileValue) ? fileValue : [fileValue];
            transformedParams[key] = fileArrayValue.map(function (arrayValue) {
                return convertType(arrayValue, argumentOptions);
            });
        }
        else {
            transformedParams[key] = convertType(fileValue, argumentOptions);
        }
    });
    return transformedParams;
}
function convertType(value, propOptions) {
    if (propOptions.type.name === 'Boolean') {
        switch (value) {
            case 'true':
                return propOptions.type(true);
            case 'false':
                return propOptions.type(false);
        }
    }
    return propOptions.type(value);
}
var argNameRegExp = /^-{1,2}(\w+)(=(\w+))?$/;
var booleanValue = ['1', '0', 'true', 'false'];
/**
 * commandLineArgs throws an error if we pass aa value for a boolean arg as follows:
 * myCommand -a=true --booleanArg=false --otherArg true
 * this function removes these booleans so as to avoid errors from commandLineArgs
 * @param args
 * @param config
 */
function removeBooleanValues(args, config) {
    function removeBooleanArgs(argsAndLastValue, arg) {
        var _a = getParamConfig(arg, config), argOptions = _a.argOptions, argValue = _a.argValue;
        var lastOption = argsAndLastValue.lastOption;
        if (lastOption != null && options_helper_1.isBoolean(lastOption) && booleanValue.some(function (boolValue) { return boolValue === arg; })) {
            var args_1 = argsAndLastValue.args.concat();
            args_1.pop();
            return { args: args_1 };
        }
        else if (argOptions != null && options_helper_1.isBoolean(argOptions) && argValue != null) {
            return { args: argsAndLastValue.args };
        }
        else {
            return { args: __spreadArray(__spreadArray([], argsAndLastValue.args), [arg]), lastOption: argOptions };
        }
    }
    return args.reduce(removeBooleanArgs, { args: [] }).args;
}
exports.removeBooleanValues = removeBooleanValues;
/**
 * Gets the values of any boolean arguments that were specified on the command line with a value
 * These arguments were removed by removeBooleanValues
 * @param args
 * @param config
 */
function getBooleanValues(args, config) {
    function getBooleanValues(argsAndLastOption, arg) {
        var _a = getParamConfig(arg, config), argOptions = _a.argOptions, argName = _a.argName, argValue = _a.argValue;
        var lastOption = argsAndLastOption.lastOption;
        if (argOptions != null && options_helper_1.isBoolean(argOptions) && argValue != null && argName != null) {
            argsAndLastOption.partial[argName] = convertType(argValue, argOptions);
        }
        else if (argsAndLastOption.lastName != null &&
            lastOption != null &&
            options_helper_1.isBoolean(lastOption) &&
            booleanValue.some(function (boolValue) { return boolValue === arg; })) {
            argsAndLastOption.partial[argsAndLastOption.lastName] = convertType(arg, lastOption);
        }
        return { partial: argsAndLastOption.partial, lastName: argName, lastOption: argOptions };
    }
    return args.reduce(getBooleanValues, { partial: {} }).partial;
}
exports.getBooleanValues = getBooleanValues;
function getParamConfig(arg, config) {
    var regExpResult = argNameRegExp.exec(arg);
    if (regExpResult == null) {
        return {};
    }
    var nameOrAlias = regExpResult[1];
    for (var argName in config) {
        var argConfig = config[argName];
        if (argName === nameOrAlias || argConfig.alias === nameOrAlias) {
            return { argOptions: argConfig, argName: argName, argValue: regExpResult[3] };
        }
    }
    return {};
}
