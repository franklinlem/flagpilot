const js = require('@eslint/js');
const ts = require('typescript-eslint');
module.exports = ts.config(
  { ignores: ['**/dist/**', '**/coverage/**', '**/schema.d.ts'] },
  js.configs.recommended,
  ...ts.configs.recommended,
);
