"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.build = build;
const getArray = (paths) => {
    return paths;
};
const getArrayGroup = () => {
    return [""].slice(0, 0);
};
function build(options) {
    return options.group ? getArrayGroup : getArray;
}
