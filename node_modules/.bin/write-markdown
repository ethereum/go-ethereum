#!/usr/bin/env node
"use strict";
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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
var parse_1 = require("./parse");
var path_1 = require("path");
var fs_1 = require("fs");
var helpers_1 = require("./helpers");
var write_markdown_constants_1 = require("./write-markdown.constants");
var string_format_1 = __importDefault(require("string-format"));
var chalk_1 = __importDefault(require("chalk"));
function writeMarkdown() {
    return __awaiter(this, void 0, void 0, function () {
        var args, markdownPath, markdownFileContent, usageGuides, modifiedFileContent, action, contentMatch, relativePath;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0:
                    args = parse_1.parse(write_markdown_constants_1.argumentConfig, write_markdown_constants_1.parseOptions);
                    markdownPath = path_1.resolve(args.markdownPath);
                    console.log("Loading existing file from '" + chalk_1.default.blue(markdownPath) + "'");
                    markdownFileContent = fs_1.readFileSync(markdownPath).toString();
                    usageGuides = helpers_1.generateUsageGuides(args);
                    modifiedFileContent = markdownFileContent;
                    if (usageGuides != null) {
                        modifiedFileContent = helpers_1.addContent(markdownFileContent, usageGuides, args);
                        if (!args.skipFooter) {
                            modifiedFileContent = helpers_1.addCommandLineArgsFooter(modifiedFileContent);
                        }
                    }
                    return [4 /*yield*/, helpers_1.insertCode({ fileContent: modifiedFileContent, filePath: markdownPath }, args)];
                case 1:
                    modifiedFileContent = _a.sent();
                    action = args.verify === true ? "verify" : "write";
                    contentMatch = markdownFileContent === modifiedFileContent ? "match" : "nonMatch";
                    relativePath = path_1.relative(process.cwd(), markdownPath);
                    switch (action + "_" + contentMatch) {
                        case 'verify_match':
                            console.log(chalk_1.default.green("'" + relativePath + "' content as expected. No update required."));
                            break;
                        case 'verify_nonMatch':
                            console.warn(chalk_1.default.yellow(string_format_1.default(args.verifyMessage || "'" + relativePath + "' file out of date. Rerun write-markdown to update.", {
                                fileName: relativePath,
                            })));
                            return [2 /*return*/, process.exit(1)];
                        case 'write_match':
                            console.log(chalk_1.default.blue("'" + relativePath + "' content not modified, not writing to file."));
                            break;
                        case 'write_nonMatch':
                            console.log("Writing modified file to '" + chalk_1.default.blue(relativePath) + "'");
                            fs_1.writeFileSync(relativePath, modifiedFileContent);
                            break;
                    }
                    return [2 /*return*/];
            }
        });
    });
}
writeMarkdown();
