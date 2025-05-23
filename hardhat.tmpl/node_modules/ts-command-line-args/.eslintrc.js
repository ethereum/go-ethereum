module.exports = {
    env: {
        node: true,
        jest: true,
    },
    extends: [
        'eslint:recommended',
        'plugin:@typescript-eslint/eslint-recommended',
        'plugin:@typescript-eslint/recommended',
        'prettier',
        'prettier/@typescript-eslint',
        'plugin:prettier/recommended',
    ],
    parser: '@typescript-eslint/parser',
    parserOptions: {
        ecmaVersion: 11,
        sourceType: 'module',
    },
    plugins: ['@typescript-eslint'],
    rules: {
        indent: 'off',
        semi: ['error', 'always'],
        '@typescript-eslint/no-explicit-any': 'off',
        'prettier/prettier': ['error', { endOfLine: 'auto' }],
    },
    ignorePatterns: ['dist/'],
};
