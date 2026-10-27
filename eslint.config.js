const js = require('@eslint/js');
const ts = require('typescript-eslint');
module.exports = ts.config(
  { ignores: ['**/dist/**', '**/coverage/**', '**/schema.d.ts'] },
  js.configs.recommended,
  ...ts.configs.recommended,
  { files: ['examples/consumer/**/*.mjs'], languageOptions: { globals: {
    process: 'readonly', Buffer: 'readonly', fetch: 'readonly', Response: 'readonly', URL: 'readonly',
  } } },
  { files: ['examples/consumer/public/*.js'], languageOptions: { globals: {
    document: 'readonly', Intl: 'readonly', fetch: 'readonly', AbortSignal: 'readonly',
  } } },
);
