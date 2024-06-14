"use strict";
var __spreadArray = (this && this.__spreadArray) || function (to, from) {
    for (var i = 0, il = from.length, j = to.length; i < il; i++, j++)
        to[j] = from[i];
    return to;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.loadArgConfig = exports.generateUsageGuides = exports.getType = exports.createOptionRow = exports.createHeading = exports.createOptionsSection = exports.createOptionsSections = exports.createSectionTable = exports.createSectionContent = exports.createSection = exports.createUsageGuide = void 0;
var path_1 = require("path");
var command_line_helper_1 = require("./command-line.helper");
var options_helper_1 = require("./options.helper");
var string_helper_1 = require("./string.helper");
function createUsageGuide(config) {
    var options = config.parseOptions || {};
    var headerSections = options.headerContentSections || [];
    var footerSections = options.footerContentSections || [];
    return __spreadArray(__spreadArray(__spreadArray([], headerSections.filter(filterMarkdownSections).map(function (section) { return createSection(section, config); })), createOptionsSections(config.arguments, options)), footerSections.filter(filterMarkdownSections).map(function (section) { return createSection(section, config); })).join('\n');
}
exports.createUsageGuide = createUsageGuide;
function filterMarkdownSections(section) {
    return section.includeIn == null || section.includeIn === 'both' || section.includeIn === 'markdown';
}
function createSection(section, config) {
    var _a;
    return "\n" + createHeading(section, ((_a = config.parseOptions) === null || _a === void 0 ? void 0 : _a.defaultSectionHeaderLevel) || 1) + "\n" + createSectionContent(section) + "\n";
}
exports.createSection = createSection;
function createSectionContent(section) {
    if (typeof section.content === 'string') {
        return string_helper_1.convertChalkStringToMarkdown(section.content);
    }
    if (Array.isArray(section.content)) {
        if (section.content.every(function (content) { return typeof content === 'string'; })) {
            return section.content.map(string_helper_1.convertChalkStringToMarkdown).join('\n');
        }
        else if (section.content.every(function (content) { return typeof content === 'object'; })) {
            return createSectionTable(section.content);
        }
    }
    return '';
}
exports.createSectionContent = createSectionContent;
function createSectionTable(rows) {
    if (rows.length === 0) {
        return "";
    }
    var cellKeys = Object.keys(rows[0]);
    return "\n|" + cellKeys.map(function (key) { return " " + key + " "; }).join('|') + "|\n|" + cellKeys.map(function () { return '-'; }).join('|') + "|\n" + rows.map(function (row) { return "| " + cellKeys.map(function (key) { return string_helper_1.convertChalkStringToMarkdown(row[key]); }).join(' | ') + " |"; }).join('\n');
}
exports.createSectionTable = createSectionTable;
function createOptionsSections(cliArguments, options) {
    var normalisedConfig = command_line_helper_1.normaliseConfig(cliArguments);
    var optionList = command_line_helper_1.createCommandLineConfig(normalisedConfig);
    if (optionList.length === 0) {
        return [];
    }
    return options_helper_1.getOptionSections(options).map(function (section) { return createOptionsSection(optionList, section, options); });
}
exports.createOptionsSections = createOptionsSections;
function createOptionsSection(optionList, content, options) {
    optionList = optionList.filter(function (option) { return filterOptions(option, content.group); });
    var anyAlias = optionList.some(function (option) { return option.alias != null; });
    var anyDescription = optionList.some(function (option) { return option.description != null; });
    var footer = options_helper_1.generateTableFooter(optionList, options);
    return "\n" + createHeading(content, 2) + "\n| Argument |" + (anyAlias ? ' Alias |' : '') + " Type |" + (anyDescription ? ' Description |' : '') + "\n|-|" + (anyAlias ? '-|' : '') + "-|" + (anyDescription ? '-|' : '') + "\n" + optionList
        .map(function (option) { return options_helper_1.mapDefinitionDetails(option, options); })
        .map(function (option) { return createOptionRow(option, anyAlias, anyDescription); })
        .join('\n') + "\n" + (footer != null ? footer + '\n' : '');
}
exports.createOptionsSection = createOptionsSection;
function filterOptions(option, groups) {
    return (groups == null ||
        (typeof groups === 'string' && (groups === option.group || (groups === '_none' && option.group == null))) ||
        (Array.isArray(groups) &&
            (groups.some(function (group) { return group === option.group; }) ||
                (groups.some(function (group) { return group === '_none'; }) && option.group == null))));
}
function createHeading(section, defaultLevel) {
    if (section.header == null) {
        return '';
    }
    var headingLevel = Array.from({ length: section.headerLevel || defaultLevel })
        .map(function () { return "#"; })
        .join('');
    return headingLevel + " " + section.header + "\n";
}
exports.createHeading = createHeading;
function createOptionRow(option, includeAlias, includeDescription) {
    if (includeAlias === void 0) { includeAlias = true; }
    if (includeDescription === void 0) { includeDescription = true; }
    var alias = includeAlias ? " " + (option.alias == null ? '' : '**' + option.alias + '** ') + "|" : "";
    var description = includeDescription
        ? " " + (option.description == null ? '' : string_helper_1.convertChalkStringToMarkdown(option.description) + ' ') + "|"
        : "";
    return "| **" + option.name + "** |" + alias + " " + getType(option) + "|" + description;
}
exports.createOptionRow = createOptionRow;
function getType(option) {
    if (option.typeLabel) {
        return string_helper_1.convertChalkStringToMarkdown(option.typeLabel) + " ";
    }
    //TODO: add modifiers
    var type = option.type ? option.type.name.toLowerCase() : 'string';
    var multiple = option.multiple || option.lazyMultiple ? '[]' : '';
    return "" + type + multiple + " ";
}
exports.getType = getType;
function generateUsageGuides(args) {
    if (args.jsFile == null) {
        console.log("No jsFile defined for usage guide generation. See 'write-markdown -h' for details on generating usage guides.");
        return undefined;
    }
    function mapJsImports(imports, jsFile) {
        return __spreadArray(__spreadArray([], imports), args.configImportName.map(function (importName) { return ({ jsFile: jsFile, importName: importName }); }));
    }
    return args.jsFile
        .reduce(mapJsImports, new Array())
        .map(function (_a) {
        var jsFile = _a.jsFile, importName = _a.importName;
        return loadArgConfig(jsFile, importName);
    })
        .filter(isDefined)
        .map(createUsageGuide);
}
exports.generateUsageGuides = generateUsageGuides;
function loadArgConfig(jsFile, importName) {
    var jsPath = path_1.join(process.cwd(), jsFile);
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    var jsExports = require(jsPath);
    var argConfig = jsExports[importName];
    if (argConfig == null) {
        console.warn("Could not import ArgumentConfig named '" + importName + "' from jsFile '" + jsFile + "'");
        return undefined;
    }
    return argConfig;
}
exports.loadArgConfig = loadArgConfig;
function isDefined(value) {
    return value != null;
}
