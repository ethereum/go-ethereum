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
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g;
    return g = { next: verb(0), "throw": verb(1), "return": verb(2) }, typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (_) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
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
exports.insertCode = void 0;
var line_ending_helper_1 = require("./line-ending.helper");
var path_1 = require("path");
var util_1 = require("util");
var fs_1 = require("fs");
var chalk_1 = __importDefault(require("chalk"));
var asyncReadFile = util_1.promisify(fs_1.readFile);
var asyncWriteFile = util_1.promisify(fs_1.writeFile);
/**
 * Loads content from other files and inserts it into the target file
 * @param input - if a string is provided the target file is loaded from that path AND saved to that path once content has been inserted. If a `FileDetails` object is provided the content is not saved when done.
 * @param partialOptions - optional. changes the default tokens
 */
function insertCode(input, partialOptions) {
    return __awaiter(this, void 0, void 0, function () {
        var options, fileDetails, filePath, content, lineBreak, lines, modifiedContent;
        var _a;
        return __generator(this, function (_b) {
            switch (_b.label) {
                case 0:
                    options = __assign({ removeDoubleBlankLines: false }, partialOptions);
                    if (!(typeof input === 'string')) return [3 /*break*/, 2];
                    filePath = path_1.resolve(input);
                    console.log("Loading existing file from '" + chalk_1.default.blue(filePath) + "'");
                    _a = { filePath: filePath };
                    return [4 /*yield*/, asyncReadFile(filePath)];
                case 1:
                    fileDetails = (_a.fileContent = (_b.sent()).toString(), _a);
                    return [3 /*break*/, 3];
                case 2:
                    fileDetails = input;
                    _b.label = 3;
                case 3:
                    content = fileDetails.fileContent;
                    lineBreak = line_ending_helper_1.findEscapeSequence(content);
                    lines = line_ending_helper_1.splitContent(content);
                    return [4 /*yield*/, insertCodeImpl(fileDetails.filePath, lines, options, 0)];
                case 4:
                    lines = _b.sent();
                    if (options.removeDoubleBlankLines) {
                        lines = lines.filter(function (line, index, lines) { return line_ending_helper_1.filterDoubleBlankLines(line, index, lines); });
                    }
                    modifiedContent = lines.join(lineBreak);
                    if (!(typeof input === 'string')) return [3 /*break*/, 6];
                    console.log("Saving modified content to '" + chalk_1.default.blue(fileDetails.filePath) + "'");
                    return [4 /*yield*/, asyncWriteFile(fileDetails.filePath, modifiedContent)];
                case 5:
                    _b.sent();
                    _b.label = 6;
                case 6: return [2 /*return*/, modifiedContent];
            }
        });
    });
}
exports.insertCode = insertCode;
function insertCodeImpl(filePath, lines, options, startLine) {
    return __awaiter(this, void 0, void 0, function () {
        var insertCodeBelow, insertCodeAbove, insertCodeBelowResult, insertCodeAboveResult, linesFromFile, linesBefore, linesAfter;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0:
                    insertCodeBelow = options === null || options === void 0 ? void 0 : options.insertCodeBelow;
                    insertCodeAbove = options === null || options === void 0 ? void 0 : options.insertCodeAbove;
                    if (insertCodeBelow == null) {
                        return [2 /*return*/, Promise.resolve(lines)];
                    }
                    insertCodeBelowResult = insertCodeBelow != null
                        ? findIndex(lines, function (line) { return line.indexOf(insertCodeBelow) === 0; }, startLine)
                        : undefined;
                    if (insertCodeBelowResult == null) {
                        return [2 /*return*/, Promise.resolve(lines)];
                    }
                    insertCodeAboveResult = insertCodeAbove != null
                        ? findIndex(lines, function (line) { return line.indexOf(insertCodeAbove) === 0; }, insertCodeBelowResult.lineIndex)
                        : undefined;
                    return [4 /*yield*/, loadLines(filePath, options, insertCodeBelowResult)];
                case 1:
                    linesFromFile = _a.sent();
                    linesBefore = lines.slice(0, insertCodeBelowResult.lineIndex + 1);
                    linesAfter = insertCodeAboveResult != null ? lines.slice(insertCodeAboveResult.lineIndex) : [];
                    lines = __spreadArray(__spreadArray(__spreadArray([], linesBefore), linesFromFile), linesAfter);
                    return [2 /*return*/, insertCodeAboveResult == null
                            ? lines
                            : insertCodeImpl(filePath, lines, options, insertCodeAboveResult.lineIndex)];
            }
        });
    });
}
var fileRegExp = /file="([^"]+)"/;
var codeCommentRegExp = /codeComment(="([^"]+)")?/; //https://regex101.com/r/3MVdBO/1
var snippetRegExp = /snippetName="([^"]+)"/;
function loadLines(targetFilePath, options, result) {
    var _a;
    return __awaiter(this, void 0, void 0, function () {
        var partialPathResult, codeCommentResult, snippetResult, partialPath, filePath, fileBuffer, contentLines, copyBelowMarker, copyAboveMarker, copyBelowIndex, copyAboveIndex;
        return __generator(this, function (_b) {
            switch (_b.label) {
                case 0:
                    partialPathResult = fileRegExp.exec(result.line);
                    if (partialPathResult == null) {
                        throw new Error("insert code token (" + options.insertCodeBelow + ") found in file but file path not specified (file=\"relativePath/from/markdown/toFile.whatever\")");
                    }
                    codeCommentResult = codeCommentRegExp.exec(result.line);
                    snippetResult = snippetRegExp.exec(result.line);
                    partialPath = partialPathResult[1];
                    filePath = path_1.isAbsolute(partialPath) ? partialPath : path_1.join(path_1.dirname(targetFilePath), partialPathResult[1]);
                    console.log("Inserting code from '" + chalk_1.default.blue(filePath) + "' into '" + chalk_1.default.blue(targetFilePath) + "'");
                    return [4 /*yield*/, asyncReadFile(filePath)];
                case 1:
                    fileBuffer = _b.sent();
                    contentLines = line_ending_helper_1.splitContent(fileBuffer.toString());
                    copyBelowMarker = options.copyCodeBelow;
                    copyAboveMarker = options.copyCodeAbove;
                    copyBelowIndex = copyBelowMarker != null ? contentLines.findIndex(findLine(copyBelowMarker, snippetResult === null || snippetResult === void 0 ? void 0 : snippetResult[1])) : -1;
                    copyAboveIndex = copyAboveMarker != null
                        ? contentLines.findIndex(function (line, index) { return line.indexOf(copyAboveMarker) === 0 && index > copyBelowIndex; })
                        : -1;
                    if (snippetResult != null && copyBelowIndex < 0) {
                        throw new Error("The copyCodeBelow marker '" + options.copyCodeBelow + "' was not found with the requested snippet: '" + snippetResult[1] + "'");
                    }
                    contentLines = contentLines.slice(copyBelowIndex + 1, copyAboveIndex > 0 ? copyAboveIndex : undefined);
                    if (codeCommentResult != null) {
                        contentLines = __spreadArray(__spreadArray(['```' + ((_a = codeCommentResult[2]) !== null && _a !== void 0 ? _a : '')], contentLines), ['```']);
                    }
                    return [2 /*return*/, contentLines];
            }
        });
    });
}
function findLine(copyBelowMarker, snippetName) {
    return function (line) {
        return line.indexOf(copyBelowMarker) === 0 && (snippetName == null || line.indexOf(snippetName) > 0);
    };
}
function findIndex(lines, predicate, startLine) {
    for (var lineIndex = startLine; lineIndex < lines.length; lineIndex++) {
        var line = lines[lineIndex];
        if (predicate(line)) {
            return { lineIndex: lineIndex, line: line };
        }
    }
    return undefined;
}
