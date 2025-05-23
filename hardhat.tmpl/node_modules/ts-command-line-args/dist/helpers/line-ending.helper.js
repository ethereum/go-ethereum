"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.filterDoubleBlankLines = exports.splitContent = exports.findEscapeSequence = exports.getEscapeSequence = exports.lineEndings = void 0;
var os_1 = require("os");
exports.lineEndings = ['LF', 'CRLF', 'CR', 'LFCR'];
var multiCharRegExp = /(\r\n)|(\n\r)/g;
var singleCharRegExp = /(\r)|(\n)/g;
// https://en.wikipedia.org/wiki/Newline
/**
 * returns escape sequence for given line ending
 * @param lineEnding
 */
function getEscapeSequence(lineEnding) {
    switch (lineEnding) {
        case 'CR':
            return '\r';
        case 'CRLF':
            return '\r\n';
        case 'LF':
            return '\n';
        case 'LFCR':
            return '\n\r';
        default:
            return handleNever(lineEnding);
    }
}
exports.getEscapeSequence = getEscapeSequence;
function handleNever(lineEnding) {
    throw new Error("Unknown line ending: '" + lineEnding + "'. Line Ending must be one of " + exports.lineEndings.join(', '));
}
function findEscapeSequence(content) {
    var multiCharMatch = multiCharRegExp.exec(content);
    if (multiCharMatch != null) {
        return multiCharMatch[0];
    }
    var singleCharMatch = singleCharRegExp.exec(content);
    if (singleCharMatch != null) {
        return singleCharMatch[0];
    }
    return os_1.EOL;
}
exports.findEscapeSequence = findEscapeSequence;
/**
 * Splits a string into an array of lines.
 * Handles any line endings including a file with a mixture of lineEndings
 *
 * @param content multi line string to be split
 */
function splitContent(content) {
    content = content.replace(multiCharRegExp, '\n');
    content = content.replace(singleCharRegExp, '\n');
    return content.split('\n');
}
exports.splitContent = splitContent;
var nonWhitespaceRegExp = /[^ \t]/;
function filterDoubleBlankLines(line, index, lines) {
    var previousLine = index > 0 ? lines[index - 1] : undefined;
    return nonWhitespaceRegExp.test(line) || previousLine == null || nonWhitespaceRegExp.test(previousLine);
}
exports.filterDoubleBlankLines = filterDoubleBlankLines;
