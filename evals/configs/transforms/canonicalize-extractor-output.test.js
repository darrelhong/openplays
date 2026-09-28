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

const test = require("node:test");
const assert = require("node:assert");
const {
  canonicalizeExtractorOutput,
} = require("./canonicalize-extractor-output.js");

// Mirrors cleanJSONResponse() in extractor.go.
function prodClean(s) {
  let text = String(s ?? "").trim();
  if (text.startsWith("```")) {
    const nl = text.indexOf("\n");
    if (nl !== -1) text = text.slice(nl + 1);
    const end = text.lastIndexOf("```");
    if (end !== -1) text = text.slice(0, end);
  }
  return text.trim();
}

// Mirrors jsonValuePrefix() + parseResponse() in extractor.go.
//
// Production now reads ONE JSON value with json.Decoder and ignores trailing
// bytes, so this must too. Reading the first value rather than the whole string
// is what makes "valid array + stray }" acceptable instead of fatal.
function prodFirstValue(text) {
  // Locate the end of the first JSON value the way json.Decoder does: scan for
  // the matching bracket, then let JSON.parse validate. Returns [value, ok].
  const isWs = (c) => c === " " || c === "\t" || c === "\n" || c === "\r";
  let i = 0;
  while (i < text.length && isWs(text[i])) i++;
  if (i >= text.length) return [undefined, false];

  const start = i;
  const first = text[i];

  // json.Decoder accepts any JSON value at top level, including primitives
  // (null, true, 42, "str"). Only {} and [] need bracket matching.
  if (first === '"') {
    let j = i + 1;
    let escaped = false;
    let closed = false;
    while (j < text.length) {
      const ch = text[j];
      if (escaped) escaped = false;
      else if (ch === "\\") escaped = true;
      else if (ch === '"') {
        closed = true;
        j++;
        break;
      }
      j++;
    }
    if (!closed) return [undefined, false];
    try {
      return [JSON.parse(text.slice(start, j)), true];
    } catch {
      return [undefined, false];
    }
  }

  if (first !== "{" && first !== "[") {
    // Number / true / false / null literal.
    let j = i;
    while (j < text.length && /[0-9eE+.\-a-zA-Z]/.test(text[j])) j++;
    try {
      return [JSON.parse(text.slice(start, j)), true];
    } catch {
      return [undefined, false];
    }
  }

  const stack = [first];
  let inString = false;
  let escaped = false;
  i++;
  while (i < text.length && stack.length > 0) {
    const ch = text[i];
    if (inString) {
      if (escaped) escaped = false;
      else if (ch === "\\") escaped = true;
      else if (ch === '"') inString = false;
    } else if (ch === '"') {
      inString = true;
    } else if (ch === "{" || ch === "[") {
      stack.push(ch);
    } else if (ch === "}" || ch === "]") {
      stack.pop();
    }
    i++;
  }
  if (stack.length > 0) return [undefined, false];

  try {
    return [JSON.parse(text.slice(start, i)), true];
  } catch {
    return [undefined, false];
  }
}

function prodAccepts(raw) {
  const cleaned = prodClean(raw);
  const [v, ok] = prodFirstValue(cleaned);
  if (!ok) return false;

  // Top-level null -> empty slice, accepted. (json.Unmarshal semantics.)
  if (v === null) return true;

  if (Array.isArray(v)) {
    // Every element must decode into ParsedPlayCandidate: objects or null.
    return v.every(
      (e) => e === null || (typeof e === "object" && !Array.isArray(e)),
    );
  }

  // single-object fallback accepts any JSON object
  return typeof v === "object";
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
    name: "clean array",
    input: '[{"listing_type":"play","venue":"X"}]',
  },
  {
    name: "empty array",
    input: "[]",
  },
  {
    name: "markdown json fence",
    input: '```json\n[{"listing_type":"play","venue":"X"}]\n```',
  },
  {
    name: "markdown bare fence",
    input: '```\n[{"listing_type":"play","venue":"X"}]\n```',
  },
  {
    name: "multi-line array",
    input: '[\n  {\n    "listing_type": "play",\n    "venue": "X"\n  }\n]',
  },
  {
    name: "truncated mid-object (finish_reason=length)",
    input: '[{"listing_type":"play","venue":"X"},{"listing_type":"pl',
  },
  {
    name: "trailing junk after complete array",
    input: '[{"listing_type":"play","venue":"X"}]}',
  },
  {
    name: "trailing brace with whitespace",
    input: '[{"listing_type":"play","venue":"X"}] }',
  },
  {
    name: "trailing brace on new line",
    input: '[{"listing_type":"play","venue":"X"}]\n}',
  },
  {
    name: "trailing semicolon",
    input: '[{"listing_type":"play","venue":"X"}];',
  },
  {
    name: "brace inside a string value stays balanced",
    input: '[{"listing_type":"play","venue":"Hall {A}"}]}',
  },
  {
    name: "brace inside a string then truncation",
    input: '[{"listing_type":"play","venue":"Hall {A}"},{"venue":"Y"',
  },
  {
    name: "trailing prose after array",
    input: '[{"listing_type":"play","venue":"X"}]\nHope that helps!',
  },
  {
    name: "prose preamble before array",
    input: 'Here are the listings:\n[{"listing_type":"play","venue":"X"}]',
  },
  {
    name: "bare object (no array)",
    input: '{"listing_type":"play","venue":"X"}',
  },
  {
    name: "empty string",
    input: "",
  },
  {
    name: "not json at all",
    input: "I could not find any listings.",
  },
  {
    name: "trailing comma in array",
    input: '[{"listing_type":"play","venue":"X"},]',
  },
  {
    name: "single quotes",
    input: "[{'listing_type':'play'}]",
  },
  // Element types. Go decodes each element into a struct, so these are hard
  // errors in production while `null` is not. Each case was cross-checked
  // against the real parseResponse.
  {
    name: "element is an array",
    input: "[[]]",
  },
  {
    name: "element is a string",
    input: '["a"]',
  },
  {
    name: "element is a number",
    input: "[123]",
  },
  {
    name: "element is null",
    input: "[null]",
  },
  {
    name: "element is a bool",
    input: "[true]",
  },
  {
    name: "object then array element",
    input: '[{"listing_type":"play"},[]]',
  },
  {
    name: "object then null element",
    input: '[{"listing_type":"play"},null]',
  },
  {
    name: "top-level null",
    input: "null",
  },
];

test("transform acceptance matches production acceptance", async (t) => {
  for (const c of CASES) {
    await t.test(c.name, () => {
      const prod = prodAccepts(c.input);
      const tf = transformAccepts(c.input);
      assert.strictEqual(
        tf,
        prod,
        `production ${prod ? "accepts" : "rejects"} this, but the transform ${tf ? "accepts" : "rejects"} it.\n` +
          `  input: ${JSON.stringify(c.input.slice(0, 90))}\n` +
          `  Fix the transform to mirror extractor.go, not to be lenient.`,
      );
    });
  }
});

test("malformed JSON fails loudly rather than silently degrading", () => {
  // The specific class that bit us in a live run: a truncated response.
  const truncated = '[{"listing_type":"play","venue":"X"},{"listing_type":"pl';
  assert.throws(
    () => canonicalizeExtractorOutput(truncated),
    (err) => err instanceof SyntaxError || err instanceof Error,
    "truncated output must throw so the run reports an error, not a silent empty result",
  );
});

test("top-level null is zero listings, not an error", () => {
  // json.Unmarshal into a slice accepts null and yields an empty slice, so
  // production treats it as "nothing found". The transform must not throw.
  assert.deepStrictEqual(canonicalizeExtractorOutput("null"), []);
});

test("non-object array elements are rejected, like Go", () => {
  // Go cannot decode [] / "a" / 123 into ParsedPlayCandidate, so these fail in
  // the listener. The transform must fail too, otherwise the eval scores a
  // parsed listing for output production rejects.
  for (const bad of [
    "[[]]",
    '["a"]',
    "[123]",
    "[true]",
    '[{"venue":"X"},[]]',
  ]) {
    assert.throws(
      () => canonicalizeExtractorOutput(bad),
      (err) => err instanceof Error,
      `expected ${bad} to throw`,
    );
  }
});

// Guards the fix: a complete array followed by a stray character used to fail
// the whole response in production, discarding every listing in it.
test("trailing junk after a complete array keeps the listings", () => {
  const out = canonicalizeExtractorOutput(
    '[{"listing_type":"play","venue":"X"}]}',
  );
  assert.strictEqual(out.length, 1);
  assert.strictEqual(out[0].listing_type, "play");
});

test("canonicalisation still normalises accepted output", () => {
  const out = canonicalizeExtractorOutput(
    '[{"listing_type":"play","venue":"  Big Venue  ","contacts":[{"type":"phone","value":"123"}]}]',
  );
  assert.strictEqual(out.length, 1);
  // Venue is lowercased and trimmed. Interior whitespace is NOT collapsed —
  // that mirrors the transform as written; don't assert normalisation it
  // doesn't do.
  assert.strictEqual(out[0].venue, "big venue", "venue lowercased+trimmed");
  assert.strictEqual(out[0].currency, "SGD", "currency defaulted");
  assert.deepStrictEqual(out[0].contacts, [{ type: "phone", value: "123" }]);
  // Missing fields are defaulted to null, not dropped.
  assert.strictEqual(out[0].fee_cents, null);
  assert.strictEqual(out[0].date, null);
});
