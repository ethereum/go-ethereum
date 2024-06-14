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
    "@typescript-eslint/no-non-null-assertion": "error",
  },
  overrides: [
    {
      files: ["src/index.ts"],
      rules: {
        "@nomicfoundation/slow-imports/no-top-level-external-import": [
          "error",
          {
            ignoreModules: [
              ...slowImportsCommonIgnoredModules,
              "chai",
              "chai-as-promised",
            ],
          },
        ],
      },
    },
  ],
};
