import crypto from "crypto";

import { createHashFunction } from "./hash-utils";

export const sha256 = createHashFunction(() => crypto.createHash("sha256"));
