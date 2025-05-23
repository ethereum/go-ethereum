const {
  slowImportsCommonIgnoredModules,
} = require("../../config/eslint/constants");

module.exports = {
  extends: [`${__dirname}/../../config/eslint/eslintrc.js`],
  parserOptions: {
    project: `${__dirname}/src/tsconfig.json`,
    sourceType: "module",
  },
  rules: {
    "@nomicfoundation/hardhat-internal-rules/only-hardhat-error": "error",
  },
  overrides: [
    {
      files: [
        "src/internal/cli/cli.ts",
        "src/register.ts",
        "src/internal/lib/hardhat-lib.ts",
        "src/config.ts",
        "src/plugins.ts",
        "src/types/**/*.ts",
        // used by hh-foundry
        "src/builtin-tasks/task-names.ts",
        "src/internal/core/errors.ts",
        "src/common/index.ts",
        // used by hh-truffle
        "src/internal/core/providers/util.ts",
        "src/utils/contract-names.ts",
      ],
      rules: {
        "@nomicfoundation/slow-imports/no-top-level-external-import": [
          "error",
          {
            ignoreModules: [...slowImportsCommonIgnoredModules],
          },
        ],
      },
    },
  ],
};
