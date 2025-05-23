"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.InterpreterDataReader = void 0;
const fs = require("fs");
const util = require("util");
const VocabularyImpl_1 = require("../VocabularyImpl");
const ATNDeserializer_1 = require("../atn/ATNDeserializer");
function splitToLines(buffer) {
    let lines = [];
    let index = 0;
    while (index < buffer.length) {
        let lineStart = index;
        let lineEndLF = buffer.indexOf("\n".charCodeAt(0), index);
        let lineEndCR = buffer.indexOf("\r".charCodeAt(0), index);
        let lineEnd;
        if (lineEndCR >= 0 && (lineEndCR < lineEndLF || lineEndLF === -1)) {
            lineEnd = lineEndCR;
        }
        else if (lineEndLF >= 0) {
            lineEnd = lineEndLF;
        }
        else {
            lineEnd = buffer.length;
        }
        lines.push(buffer.toString("utf-8", lineStart, lineEnd));
        if (lineEnd === lineEndCR && lineEnd + 1 === lineEndLF) {
            index = lineEnd + 2;
        }
        else {
            index = lineEnd + 1;
        }
    }
    return lines;
}
// A class to read plain text interpreter data produced by ANTLR.
var InterpreterDataReader;
(function (InterpreterDataReader) {
    /**
     * The structure of the data file is very simple. Everything is line based with empty lines
     * separating the different parts. For lexers the layout is:
     * token literal names:
     * ...
     *
     * token symbolic names:
     * ...
     *
     * rule names:
     * ...
     *
     * channel names:
     * ...
     *
     * mode names:
     * ...
     *
     * atn:
     * <a single line with comma separated int values> enclosed in a pair of squared brackets.
     *
     * Data for a parser does not contain channel and mode names.
     */
    function parseFile(fileName) {
        return __awaiter(this, void 0, void 0, function* () {
            let result = new InterpreterDataReader.InterpreterData();
            let input = yield util.promisify(fs.readFile)(fileName);
            let lines = splitToLines(input);
            try {
                let line;
                let lineIndex = 0;
                let literalNames = [];
                let symbolicNames = [];
                line = lines[lineIndex++];
                if (line !== "token literal names:") {
                    throw new RangeError("Unexpected data entry");
                }
                for (line = lines[lineIndex++]; line !== undefined; line = lines[lineIndex++]) {
                    if (line.length === 0) {
                        break;
                    }
                    literalNames.push(line === "null" ? "" : line);
                }
                line = lines[lineIndex++];
                if (line !== "token symbolic names:") {
                    throw new RangeError("Unexpected data entry");
                }
                for (line = lines[lineIndex++]; line !== undefined; line = lines[lineIndex++]) {
                    if (line.length === 0) {
                        break;
                    }
                    symbolicNames.push(line === "null" ? "" : line);
                }
                let displayNames = [];
                result.vocabulary = new VocabularyImpl_1.VocabularyImpl(literalNames, symbolicNames, displayNames);
                line = lines[lineIndex++];
                if (line !== "rule names:") {
                    throw new RangeError("Unexpected data entry");
                }
                for (line = lines[lineIndex++]; line !== undefined; line = lines[lineIndex++]) {
                    if (line.length === 0) {
                        break;
                    }
                    result.ruleNames.push(line);
                }
                line = lines[lineIndex++];
                if (line === "channel names:") { // Additional lexer data.
                    result.channels = [];
                    for (line = lines[lineIndex++]; line !== undefined; line = lines[lineIndex++]) {
                        if (line.length === 0) {
                            break;
                        }
                        result.channels.push(line);
                    }
                    line = lines[lineIndex++];
                    if (line !== "mode names:") {
                        throw new RangeError("Unexpected data entry");
                    }
                    result.modes = [];
                    for (line = lines[lineIndex++]; line !== undefined; line = lines[lineIndex++]) {
                        if (line.length === 0) {
                            break;
                        }
                        result.modes.push(line);
                    }
                }
                line = lines[lineIndex++];
                if (line !== "atn:") {
                    throw new RangeError("Unexpected data entry");
                }
                line = lines[lineIndex++];
                let elements = line.split(",");
                let serializedATN = new Uint16Array(elements.length);
                for (let i = 0; i < elements.length; ++i) {
                    let value;
                    let element = elements[i];
                    if (element.startsWith("[")) {
                        value = parseInt(element.substring(1).trim(), 10);
                    }
                    else if (element.endsWith("]")) {
                        value = parseInt(element.substring(0, element.length - 1).trim(), 10);
                    }
                    else {
                        value = parseInt(element.trim(), 10);
                    }
                    serializedATN[i] = value;
                }
                let deserializer = new ATNDeserializer_1.ATNDeserializer();
                result.atn = deserializer.deserialize(serializedATN);
            }
            catch (e) {
                // We just swallow the error and return empty objects instead.
            }
            return result;
        });
    }
    InterpreterDataReader.parseFile = parseFile;
    class InterpreterData {
        constructor() {
            this.vocabulary = VocabularyImpl_1.VocabularyImpl.EMPTY_VOCABULARY;
            this.ruleNames = [];
        }
    }
    InterpreterDataReader.InterpreterData = InterpreterData;
})(InterpreterDataReader = exports.InterpreterDataReader || (exports.InterpreterDataReader = {}));
//# sourceMappingURL=InterpreterDataReader.js.map