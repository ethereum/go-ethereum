const Sha256Hash = require("hash.js/lib/hash/sha/256");

import { createHashFunction } from "../hash-utils";

export const sha256 = createHashFunction(() => new Sha256Hash());
