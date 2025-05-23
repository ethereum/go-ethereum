const browserifyAes = require("browserify-aes");

const SUPPORTED_MODES = ["aes-128-ctr", "aes-128-cbc", "aes-256-cbc"];

function ensureAesMode(mode: string) {
  if (!mode.startsWith("aes-")) {
    throw new Error(`AES submodule doesn't support mode ${mode}`);
  }
}

function warnIfUnsuportedMode(mode: string) {
  if (!SUPPORTED_MODES.includes(mode)) {
    // tslint:disable-next-line no-console
    console.warn("Using an unsupported AES mode. Consider using aes-128-ctr.");
  }
}

export function encrypt(
  msg: Buffer,
  key: Buffer,
  iv: Buffer,
  mode = "aes-128-ctr",
  pkcs7PaddingEnabled = true
): Buffer {
  ensureAesMode(mode);

  const cipher = browserifyAes.createCipheriv(mode, key, iv);
  cipher.setAutoPadding(pkcs7PaddingEnabled);

  const encrypted = cipher.update(msg);
  const final = cipher.final();

  return Buffer.concat([encrypted, final]);
}

export function decrypt(
  cypherText: Buffer,
  key: Buffer,
  iv: Buffer,
  mode = "aes-128-ctr",
  pkcs7PaddingEnabled = true
): Buffer {
  ensureAesMode(mode);

  const decipher = browserifyAes.createDecipheriv(mode, key, iv);
  decipher.setAutoPadding(pkcs7PaddingEnabled);

  const encrypted = decipher.update(cypherText);
  const final = decipher.final();

  return Buffer.concat([encrypted, final]);
}
