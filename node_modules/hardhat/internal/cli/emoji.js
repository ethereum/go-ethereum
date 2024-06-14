"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.emoji = exports.enableEmoji = void 0;
let emojiEnabled = false;
function enableEmoji() {
    emojiEnabled = true;
}
exports.enableEmoji = enableEmoji;
function emoji(msgIfEnabled, msgIfDisabled = "") {
    return emojiEnabled ? msgIfEnabled : msgIfDisabled;
}
exports.emoji = emoji;
//# sourceMappingURL=emoji.js.map