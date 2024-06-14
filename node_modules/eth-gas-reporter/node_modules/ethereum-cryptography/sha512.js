"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.sha512 = void 0;
const sha512_1 = require("@noble/hashes/sha512");
const utils_1 = require("./utils");
exports.sha512 = (0, utils_1.wrapHash)(sha512_1.sha512);
