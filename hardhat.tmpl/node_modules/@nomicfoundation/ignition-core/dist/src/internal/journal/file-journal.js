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
Object.defineProperty(exports, "__esModule", { value: true });
exports.FileJournal = void 0;
/* eslint-disable no-bitwise */
const fs_1 = __importStar(require("fs"));
const ndjson_1 = require("ndjson");
const deserialize_replacer_1 = require("./utils/deserialize-replacer");
const emitExecutionEvent_1 = require("./utils/emitExecutionEvent");
const serialize_replacer_1 = require("./utils/serialize-replacer");
/**
 * A file-based journal.
 *
 * @beta
 */
class FileJournal {
    _filePath;
    _executionEventListener;
    constructor(_filePath, _executionEventListener) {
        this._filePath = _filePath;
        this._executionEventListener = _executionEventListener;
    }
    record(message) {
        this._log(message);
        this._appendJsonLine(this._filePath, message);
    }
    async *read() {
        if (!fs_1.default.existsSync(this._filePath)) {
            return;
        }
        const stream = fs_1.default.createReadStream(this._filePath).pipe((0, ndjson_1.parse)());
        for await (const chunk of stream) {
            const json = JSON.stringify(chunk);
            const deserializedChunk = JSON.parse(json, deserialize_replacer_1.deserializeReplacer.bind(this));
            yield deserializedChunk;
        }
    }
    _appendJsonLine(path, value) {
        const flags = fs_1.constants.O_CREAT |
            fs_1.constants.O_WRONLY | // Write only
            fs_1.constants.O_APPEND | // Append
            fs_1.constants.O_DSYNC; // Synchronous I/O waiting for writes of content and metadata
        const fd = (0, fs_1.openSync)(path, flags);
        (0, fs_1.writeFileSync)(fd, `\n${JSON.stringify(value, serialize_replacer_1.serializeReplacer.bind(this))}`, "utf-8");
        (0, fs_1.closeSync)(fd);
    }
    _log(message) {
        if (this._executionEventListener !== undefined) {
            (0, emitExecutionEvent_1.emitExecutionEvent)(message, this._executionEventListener);
        }
    }
}
exports.FileJournal = FileJournal;
//# sourceMappingURL=file-journal.js.map