"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.removeAdditionalFormatting = exports.convertChalkStringToMarkdown = void 0;
var chalkStringStyleRegExp = /(?<!\\){([^}]+?)[ \n\r](.+?[^\\])}/gms;
var newLineRegExp = /\n/g;
var highlightModifier = 'highlight';
var codeModifier = 'code';
/**
 * Converts a string with chalk formatting into a string safe for markdown
 *
 * @param input
 */
function convertChalkStringToMarkdown(input) {
    return (input
        .replace(chalkStringStyleRegExp, replaceChalkFormatting)
        //replace new line with 2 spaces then new line
        .replace(newLineRegExp, '  \n')
        .replace(/\\{/g, '{')
        .replace(/\\}/g, '}'));
}
exports.convertChalkStringToMarkdown = convertChalkStringToMarkdown;
function replaceChalkFormatting(_substring) {
    var matches = [];
    for (var _i = 1; _i < arguments.length; _i++) {
        matches[_i - 1] = arguments[_i];
    }
    var modifier = '';
    if (matches[0].indexOf(highlightModifier) >= 0) {
        modifier = '`';
    }
    else if (matches[0].indexOf(codeModifier) >= 0) {
        var codeOptions = matches[0].split('.');
        modifier = '\n```';
        if (codeOptions[1] != null) {
            return "" + modifier + codeOptions[1] + "\n" + matches[1] + modifier + "\n";
        }
        else {
            return modifier + "\n" + matches[1] + modifier + "\n";
        }
    }
    else {
        if (matches[0].indexOf('bold') >= 0) {
            modifier += '**';
        }
        if (matches[0].indexOf('italic') >= 0) {
            modifier += '*';
        }
    }
    return "" + modifier + matches[1] + modifier;
}
/**
 * Removes formatting not supported by chalk
 *
 * @param input
 */
function removeAdditionalFormatting(input) {
    return input.replace(chalkStringStyleRegExp, removeNonChalkFormatting);
}
exports.removeAdditionalFormatting = removeAdditionalFormatting;
function removeNonChalkFormatting(substring) {
    var matches = [];
    for (var _i = 1; _i < arguments.length; _i++) {
        matches[_i - 1] = arguments[_i];
    }
    var nonChalkFormats = [highlightModifier, codeModifier];
    if (nonChalkFormats.some(function (format) { return matches[0].indexOf(format) >= 0; })) {
        return matches[1];
    }
    return substring;
}
