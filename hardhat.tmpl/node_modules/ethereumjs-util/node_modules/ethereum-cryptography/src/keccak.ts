import { createHashFunction } from "./hash-utils";

const createKeccakHash = require("keccak");

export const keccak224 = createHashFunction(() =>
  createKeccakHash("keccak224")
);

export const keccak256 = createHashFunction(() =>
  createKeccakHash("keccak256")
);

export const keccak384 = createHashFunction(() =>
  createKeccakHash("keccak384")
);

export const keccak512 = createHashFunction(() =>
  createKeccakHash("keccak512")
);
