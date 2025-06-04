"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.sha256 = void 0;
const sha256_1 = require("@noble/hashes/sha256");
const utils_1 = require("./utils");
exports.sha256 = (0, utils_1.wrapHash)(sha256_1.sha256);
