"use strict";
// This file defines the different config types.
//
// For each possible kind of config value, we have two types:
//
// One that ends with UserConfig, which represent the config as
// written in the user's config file.
//
// The other one, with the same name except without the User part, represents
// the resolved value as used during the hardhat execution.
//
// Note that while many declarations are repeated here (i.e. network types'
// fields), we don't use `extends` as that can interfere with plugin authors
// trying to augment the config types.
Object.defineProperty(exports, "__esModule", { value: true });
//# sourceMappingURL=config.js.map