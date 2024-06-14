let emojiEnabled = false;

export function enableEmoji() {
  emojiEnabled = true;
}

export function emoji(msgIfEnabled: string, msgIfDisabled: string = "") {
  return emojiEnabled ? msgIfEnabled : msgIfDisabled;
}
