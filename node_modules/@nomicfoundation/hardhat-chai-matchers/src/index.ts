import "@nomicfoundation/hardhat-ethers";

import "./types";

import { hardhatWaffleIncompatibilityCheck } from "./internal/hardhatWaffleIncompatibilityCheck";
import "./internal/add-chai-matchers";

hardhatWaffleIncompatibilityCheck();
