import crypto from "crypto";

import { createHashFunction } from "./hash-utils";

export const ripemd160 = createHashFunction(() =>
  crypto.createHash("ripemd160")
);
