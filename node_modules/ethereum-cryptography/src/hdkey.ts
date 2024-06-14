import * as hdkeyPure from "./pure/hdkey";

const hdkey: typeof hdkeyPure.HDKey = require("./vendor/hdkey-without-crypto");

export const HDKey = hdkey;
