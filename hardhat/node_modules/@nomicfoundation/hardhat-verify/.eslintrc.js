const {
  slowImportsCommonIgnoredModules,
} = require("../../config/eslint/constants");

module.exports = {
  extends: [`${__dirname}/../../config/eslint/eslintrc.js`],
  parserOptions: {
    project: `${__dirname}/src/tsconfig.json`,
    sourceType: "module",
  },
  overrides: [
    {
      files: ["src/index.ts"],
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
