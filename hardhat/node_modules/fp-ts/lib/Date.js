"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var IO_1 = require("./IO");
/**
 * Returns the current `Date`
 *
 * @since 1.10.0
 */
exports.create = new IO_1.IO(function () { return new Date(); });
/**
 * Returns the number of milliseconds elapsed since January 1, 1970, 00:00:00 UTC
 *
 * @since 1.10.0
 */
exports.now = new IO_1.IO(function () { return new Date().getTime(); });
