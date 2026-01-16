module.exports = {
    extends: [
        'stylelint-config-standard',
    ],
    ignoreFiles: [
        'public/build/**',
        'vendor/**',
        'storage/**',
        'node_modules/**',
    ],
    rules: {
        'at-rule-no-unknown': null,
        'import-notation': null,
        'color-function-notation': null,
        'alpha-value-notation': null,
        'selector-class-pattern': null,
        'no-duplicate-selectors': null,
        'property-no-vendor-prefix': null,
        'rule-empty-line-before': null,
        'custom-property-empty-line-before': null,
    },
};
