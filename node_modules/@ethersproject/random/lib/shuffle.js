"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.shuffled = void 0;
function shuffled(array) {
    array = array.slice();
    for (var i = array.length - 1; i > 0; i--) {
        var j = Math.floor(Math.random() * (i + 1));
        var tmp = array[i];
        array[i] = array[j];
        array[j] = tmp;
    }
    return array;
}
exports.shuffled = shuffled;
//# sourceMappingURL=shuffle.js.map