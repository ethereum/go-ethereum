"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.sync = sync;
const walker_1 = require("./walker");
function sync(root, options) {
    const walker = new walker_1.Walker(root, options);
    return walker.start();
}
