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
exports.Anonymizer = void 0;
const find_up_1 = __importDefault(require("find-up"));
const fp_ts_1 = require("fp-ts");
const path = __importStar(require("path"));
const ANONYMIZED_FILE = "<user-file>";
const ANONYMIZED_MNEMONIC = "<mnemonic>";
const MNEMONIC_PHRASE_LENGTH_THRESHOLD = 7;
const MINIMUM_AMOUNT_OF_WORDS_TO_ANONYMIZE = 4;
class Anonymizer {
    constructor(_configPath) {
        this._configPath = _configPath;
    }
    /**
     * Given a sentry serialized exception
     * (https://develop.sentry.dev/sdk/event-payloads/exception/), return an
     * anonymized version of the event.
     */
    anonymize(event) {
        if (event === null || event === undefined) {
            return fp_ts_1.either.left("event is null or undefined");
        }
        if (typeof event !== "object") {
            return fp_ts_1.either.left("event is not an object");
        }
        const result = {
            event_id: event.event_id,
            platform: event.platform,
            timestamp: event.timestamp,
            extra: event.extra,
        };
        if (event.exception !== undefined && event.exception.values !== undefined) {
            const anonymizededExceptions = this._anonymizeExceptions(event.exception.values);
            result.exception = {
                values: anonymizededExceptions,
            };
        }
        return fp_ts_1.either.right(result);
    }
    /**
     * Return the anonymized filename and a boolean indicating if the content of
     * the file should be anonymized
     */
    anonymizeFilename(filename) {
        if (filename === this._configPath) {
            const packageJsonPath = this._getFilePackageJsonPath(filename);
            if (packageJsonPath === null) {
                // if we can't find a package.json, we just return the basename
                return {
                    anonymizedFilename: path.basename(filename),
                    anonymizeContent: true,
                };
            }
            return {
                anonymizedFilename: path.relative(path.dirname(packageJsonPath), filename),
                anonymizeContent: true,
            };
        }
        const parts = filename.split(path.sep);
        const nodeModulesIndex = parts.indexOf("node_modules");
        if (nodeModulesIndex === -1) {
            if (filename.startsWith("internal")) {
                // show internal parts of the stack trace
                return {
                    anonymizedFilename: filename,
                    anonymizeContent: false,
                };
            }
            // if the file isn't inside node_modules and it's a user file, we hide it completely
            return {
                anonymizedFilename: ANONYMIZED_FILE,
                anonymizeContent: true,
            };
        }
        return {
            anonymizedFilename: parts.slice(nodeModulesIndex).join(path.sep),
            anonymizeContent: false,
        };
    }
    anonymizeErrorMessage(errorMessage) {
        errorMessage = this._anonymizeMnemonic(errorMessage);
        // the \\ before path.sep is necessary for this to work on windows
        const pathRegex = new RegExp(`\\S+\\${path.sep}\\S+`, "g");
        // for files that don't have a path separator
        const fileRegex = new RegExp("\\S+\\.(js|ts)\\S*", "g");
        // hide hex strings of 20 chars or more
        const hexRegex = /(0x)?[0-9A-Fa-f]{20,}/g;
        return errorMessage
            .replace(pathRegex, ANONYMIZED_FILE)
            .replace(fileRegex, ANONYMIZED_FILE)
            .replace(hexRegex, (match) => match.replace(/./g, "x"));
    }
    raisedByHardhat(event) {
        const exceptions = event?.exception?.values;
        if (exceptions === undefined) {
            // if we can't prove that the exception doesn't come from hardhat,
            // we err on the side of reporting the error
            return true;
        }
        const originalException = exceptions[exceptions.length - 1];
        const frames = originalException?.stacktrace?.frames;
        if (frames === undefined) {
            return true;
        }
        for (const frame of frames.slice().reverse()) {
            if (frame.filename === undefined) {
                continue;
            }
            // we stop after finding either a hardhat file or a file from the user's
            // project
            if (this._isHardhatFile(frame.filename)) {
                return true;
            }
            if (frame.filename === ANONYMIZED_FILE) {
                return false;
            }
            if (this._configPath !== undefined &&
                this._configPath.includes(frame.filename)) {
                return false;
            }
        }
        // if we didn't find any hardhat frame, we don't report the error
        return false;
    }
    _getFilePackageJsonPath(filename) {
        return find_up_1.default.sync("package.json", {
            cwd: path.dirname(filename),
        });
    }
    _isHardhatFile(filename) {
        const nomiclabsPath = path.join("node_modules", "@nomiclabs");
        const nomicFoundationPath = path.join("node_modules", "@nomicfoundation");
        const truffleContractPath = path.join(nomiclabsPath, "truffle-contract");
        const isHardhatFile = (filename.startsWith(nomiclabsPath) ||
            filename.startsWith(nomicFoundationPath)) &&
            !filename.startsWith(truffleContractPath);
        return isHardhatFile;
    }
    _anonymizeExceptions(exceptions) {
        return exceptions.map((exception) => this._anonymizeException(exception));
    }
    _anonymizeException(value) {
        const result = {
            type: value.type,
        };
        if (value.value !== undefined) {
            result.value = this.anonymizeErrorMessage(value.value);
        }
        if (value.stacktrace !== undefined) {
            result.stacktrace = this._anonymizeStacktrace(value.stacktrace);
        }
        return result;
    }
    _anonymizeStacktrace(stacktrace) {
        if (stacktrace.frames !== undefined) {
            const anonymizededFrames = this._anonymizeFrames(stacktrace.frames);
            return {
                frames: anonymizededFrames,
            };
        }
        return {};
    }
    _anonymizeFrames(frames) {
        return frames.map((frame) => this._anonymizeFrame(frame));
    }
    _anonymizeFrame(frame) {
        const result = {
            lineno: frame.lineno,
            colno: frame.colno,
            function: frame.function,
        };
        let anonymizeContent = true;
        if (frame.filename !== undefined) {
            const anonymizationResult = this.anonymizeFilename(frame.filename);
            result.filename = anonymizationResult.anonymizedFilename;
            anonymizeContent = anonymizationResult.anonymizeContent;
        }
        if (!anonymizeContent) {
            result.context_line = frame.context_line;
            result.pre_context = frame.pre_context;
            result.post_context = frame.post_context;
            result.vars = frame.vars;
        }
        return result;
    }
    _anonymizeMnemonic(errorMessage) {
        const matches = getAllWordMatches(errorMessage);
        // If there are enough consecutive words, there's a good chance of there being a mnemonic phrase
        if (matches.length < MNEMONIC_PHRASE_LENGTH_THRESHOLD) {
            return errorMessage;
        }
        const mnemonicWordlist = [].concat(...[
            require("ethereum-cryptography/bip39/wordlists/czech"),
            require("ethereum-cryptography/bip39/wordlists/english"),
            require("ethereum-cryptography/bip39/wordlists/french"),
            require("ethereum-cryptography/bip39/wordlists/italian"),
            require("ethereum-cryptography/bip39/wordlists/japanese"),
            require("ethereum-cryptography/bip39/wordlists/korean"),
            require("ethereum-cryptography/bip39/wordlists/simplified-chinese"),
            require("ethereum-cryptography/bip39/wordlists/spanish"),
            require("ethereum-cryptography/bip39/wordlists/traditional-chinese"),
        ].map((wordlistModule) => wordlistModule.wordlist));
        let anonymizedMessage = errorMessage.slice(0, matches[0].index);
        // Determine all mnemonic phrase maximal fragments.
        // We check sequences of n consecutive words just in case there is a typo
        let wordIndex = 0;
        while (wordIndex < matches.length) {
            const maximalPhrase = getMaximalMnemonicPhrase(matches, errorMessage, wordIndex, mnemonicWordlist);
            if (maximalPhrase.length >= MINIMUM_AMOUNT_OF_WORDS_TO_ANONYMIZE) {
                const lastAnonymizedWord = maximalPhrase[maximalPhrase.length - 1];
                const nextWordIndex = wordIndex + maximalPhrase.length < matches.length
                    ? matches[wordIndex + maximalPhrase.length].index
                    : errorMessage.length;
                const sliceUntilNextWord = errorMessage.slice(lastAnonymizedWord.index + lastAnonymizedWord.word.length, nextWordIndex);
                anonymizedMessage += `${ANONYMIZED_MNEMONIC}${sliceUntilNextWord}`;
                wordIndex += maximalPhrase.length;
            }
            else {
                const thisWord = matches[wordIndex];
                const nextWordIndex = wordIndex + 1 < matches.length
                    ? matches[wordIndex + 1].index
                    : errorMessage.length;
                const sliceUntilNextWord = errorMessage.slice(thisWord.index, nextWordIndex);
                anonymizedMessage += sliceUntilNextWord;
                wordIndex++;
            }
        }
        return anonymizedMessage;
    }
}
exports.Anonymizer = Anonymizer;
function getMaximalMnemonicPhrase(matches, originalMessage, startIndex, mnemonicWordlist) {
    const maximalPhrase = [];
    for (let i = startIndex; i < matches.length; i++) {
        const thisMatch = matches[i];
        if (!mnemonicWordlist.includes(thisMatch.word)) {
            break;
        }
        if (maximalPhrase.length > 0) {
            // Check that there's only whitespace until this word.
            const lastMatch = maximalPhrase[maximalPhrase.length - 1];
            const lastIndex = lastMatch.index + lastMatch.word.length;
            const sliceBetweenWords = originalMessage.slice(lastIndex, thisMatch.index);
            if (!/\s+/u.test(sliceBetweenWords)) {
                break;
            }
        }
        maximalPhrase.push(thisMatch);
    }
    return maximalPhrase;
}
function getAllWordMatches(errorMessage) {
    const matches = [];
    const re = /\p{Letter}+/gu;
    let match = re.exec(errorMessage);
    while (match !== null) {
        matches.push({
            word: match[0],
            index: match.index,
        });
        match = re.exec(errorMessage);
    }
    return matches;
}
//# sourceMappingURL=anonymizer.js.map