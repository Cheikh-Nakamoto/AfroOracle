const fs = require('fs');
const assert = require('assert');
const source = fs.readFileSync('frontend/app/page.tsx', 'utf8');

assert.match(source, /<p[^>]*role="alert"/, 'error message must have role="alert"');
assert.match(source, /<table[^>]*>[\s\S]*?<caption>[\s\S]*?<\/caption>/, 'table must have a caption');
assert.strictEqual((source.match(/<th[^>]*scope="col"/g) || []).length, 5, 'all five table headers must declare scope="col"');
assert.match(source, /aria-live="polite"/, 'dynamic results must use aria-live="polite"');
assert.match(source, /<button[^>]*aria-label=/, 'buttons must have accessible labels');
