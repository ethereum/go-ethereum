"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.promise = promise;
exports.callback = callback;
const walker_1 = require("./walker");
function promise(root, options) {
    return new Promise((resolve, reject) => {
        callback(root, options, (err, output) => {
            if (err)
                return reject(err);
            resolve(output);
        });
    });
}
function callback(root, options, callback) {
    let walker = new walker_1.Walker(root, options, callback);
    walker.start();
}
