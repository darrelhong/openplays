// Contract test: the eval transform must accept exactly what production accepts,
// and reject exactly what production rejects.
//
// Production side (server/internal/listener/pipeline/extractor.go):
//   cleanJSONResponse()  strips a ```json fence, then trims
//   parseResponse()      tries array, then single object, else hard-fails
//
// Why this matters: if the transform is *more* lenient than production, the eval
// goes green on output that fails in prod — a false negative that hides a real
// bug. If it's *stricter*, the eval goes red on output prod handles fine, and
// models get judged unfairly. Both directions are wrong; the harness must
// predict production, so the guards live in config (token caps, prompts) rather
// than in the parser.
//
// Run: node --test configs/transforms/canonicalize-extractor-output.test.js

const test = require('node:test');
const assert = require('node:assert');
const { canonicalizeExtractorOutput } = require('./canonicalize-extractor-output.js');

// Mirrors cleanJSONResponse() in extractor.go.
function prodClean(s) {
  let text = String(s ?? '').trim();
  if (text.startsWith('```')) {
    const nl = text.indexOf('\n');
    if (nl !== -1) text = text.slice(nl + 1);
    const end = text.lastIndexOf('```');
    if (end !== -1) text = text.slice(0, end);
  }
  return text.trim();
}

// Mirrors parseResponse() in extractor.go: array first, then single object.
function prodAccepts(raw) {
  const cleaned = prodClean(raw);
  try {
    const v = JSON.parse(cleaned);
    if (Array.isArray(v)) return true;
    // single-object fallback accepts any JSON object
    return typeof v === 'object' && v !== null;
  } catch {
    return false;
  }
}

/** Does the transform accept this (vs throwing)? */
function transformAccepts(raw) {
  try {
    canonicalizeExtractorOutput(raw);
    return true;
  } catch {
    return false;
  }
}

const CASES = [
  {
    name: 'clean array',
    input: '[{"listing_type":"play","venue":"X"}]',
  },
  {
    name: 'empty array',
    input: '[]',
  },
  {
    name: 'markdown json fence',
    input: '```json\n[{"listing_type":"play","venue":"X"}]\n```',
  },
  {
    name: 'markdown bare fence',
    input: '```\n[{"listing_type":"play","venue":"X"}]\n```',
  },
  {
    name: 'multi-line array',
    input: '[\n  {\n    "listing_type": "play",\n    "venue": "X"\n  }\n]',
  },
  {
    name: 'truncated mid-object (finish_reason=length)',
    input: '[{"listing_type":"play","venue":"X"},{"listing_type":"pl',
  },
  {
    name: 'trailing junk after complete array',
    input: '[{"listing_type":"play","venue":"X"}]}',
  },
  {
    name: 'trailing prose after array',
    input: '[{"listing_type":"play","venue":"X"}]\nHope that helps!',
  },
  {
    name: 'prose preamble before array',
    input: 'Here are the listings:\n[{"listing_type":"play","venue":"X"}]',
  },
  {
    name: 'bare object (no array)',
    input: '{"listing_type":"play","venue":"X"}',
  },
  {
    name: 'empty string',
    input: '',
  },
  {
    name: 'not json at all',
    input: 'I could not find any listings.',
  },
  {
    name: 'trailing comma in array',
    input: '[{"listing_type":"play","venue":"X"},]',
  },
  {
    name: 'single quotes',
    input: "[{'listing_type':'play'}]",
  },
];

test('transform acceptance matches production acceptance', async (t) => {
  for (const c of CASES) {
    await t.test(c.name, () => {
      const prod = prodAccepts(c.input);
      const tf = transformAccepts(c.input);
      assert.strictEqual(
        tf,
        prod,
        `production ${prod ? 'accepts' : 'rejects'} this, but the transform ${tf ? 'accepts' : 'rejects'} it.\n` +
          `  input: ${JSON.stringify(c.input.slice(0, 90))}\n` +
          `  Fix the transform to mirror extractor.go, not to be lenient.`,
      );
    });
  }
});

test('malformed JSON fails loudly rather than silently degrading', () => {
  // The specific class that bit us in a live run: a truncated response.
  const truncated = '[{"listing_type":"play","venue":"X"},{"listing_type":"pl';
  assert.throws(
    () => canonicalizeExtractorOutput(truncated),
    (err) => err instanceof SyntaxError || err instanceof Error,
    'truncated output must throw so the run reports an error, not a silent empty result',
  );
});

test('canonicalisation still normalises accepted output', () => {
  const out = canonicalizeExtractorOutput(
    '[{"listing_type":"play","venue":"  Big Venue  ","contacts":[{"type":"phone","value":"123"}]}]',
  );
  assert.strictEqual(out.length, 1);
  // Venue is lowercased and trimmed. Interior whitespace is NOT collapsed —
  // that mirrors the transform as written; don't assert normalisation it
  // doesn't do.
  assert.strictEqual(out[0].venue, 'big venue', 'venue lowercased+trimmed');
  assert.strictEqual(out[0].currency, 'SGD', 'currency defaulted');
  assert.deepStrictEqual(out[0].contacts, [{ type: 'phone', value: '123' }]);
  // Missing fields are defaulted to null, not dropped.
  assert.strictEqual(out[0].fee_cents, null);
  assert.strictEqual(out[0].date, null);
});
