const { ripemd160: Ripemd160 } = require("hash.js/lib/hash/ripemd");

import { createHashFunction } from "../hash-utils";

export const ripemd160 = createHashFunction(() => new Ripemd160());
