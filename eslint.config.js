import js from '@eslint/js';
import globals from 'globals';

export default [
    {
        ignores: [
            'public/build/**',
            'vendor/**',
            'storage/**',
            'node_modules/**',
        ],
    },
    js.configs.recommended,
    {
        files: ['resources/js/**/*.{js,mjs}'],
        languageOptions: {
            ecmaVersion: 'latest',
            sourceType: 'module',
            globals: {
                ...globals.browser,
            },
        },
        rules: {
            'no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
            'no-console': 'off',
        },
    },
];
