// Contract tests for the assertion helpers and the composite gate.
//
// These lock in two failure modes that previously produced false confidence:
//
//  1. A missing or malformed `expected` fixture silently became `[]`, so an
//     empty model response compared equal to an empty fixture and scored a
//     PASSING 1.0. A typo'd fixture path could turn the suite green while the
//     extractor returned nothing.
//
//  2. A hard metric that never reported (assertion threw) was treated as 0 and
//     reported as "hard structure checks failed", pointing triage at the model
//     instead of the harness.
//
// Run: node --test configs/assertions/extractor-assertions.test.js

const test = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');

const assertions = require('./extractor-assertions.js');
const { extractorScoring } = require('./extractor-scoring.js');

const FIXTURE = 'file://tests/extractor/fixtures/simple-single-listing.json';

// --- fixture resolution must fail loudly ------------------------------------

test('assertions throw when `expected` cannot be resolved', async (t) => {
  const badVars = [
    ['missing var', {}],
    ['typo path', { expected: 'file://tests/extractor/fixtures/NOPE.json' }],
    ['empty string', { expected: '' }],
    ['no scheme', { expected: 'tests/extractor/fixtures/simple-single-listing.json' }],
    ['inline invalid json', { expected: '[broken' }],
    ['null', { expected: null }],
  ];

  for (const [name, vars] of badVars) {
    await t.test(name, () => {
      assert.throws(
        () => assertions.listingCountMatches([], { vars }),
        /cannot resolve expected fixtures/,
        'a broken fixture must throw, never silently resolve to an empty expectation',
      );
    });
  }
});

test('an empty response against an unresolvable fixture does not pass', () => {
  // The regression that motivated these guards: [] vs [] used to score 1.0.
  let threw = false;
  try {
    assertions.listingCountMatches([], { vars: { expected: 'file://does/not/exist.json' } });
  } catch {
    threw = true;
  }
  assert.ok(threw, 'empty-vs-missing must not produce a passing count metric');
});

test('a fixture that is not a JSON array is rejected', () => {
  const tmp = path.join(os.tmpdir(), `evals-obj-${process.pid}.json`);
  fs.writeFileSync(tmp, '{"not":"an array"}');
  try {
    assert.throws(
      () => assertions.listingCountMatches([], { vars: { expected: `file://${tmp}` } }),
      /must contain a JSON array/,
    );
  } finally {
    fs.unlinkSync(tmp);
  }
});

test('valid fixtures still resolve and compare correctly', () => {
  const expected = JSON.parse(
    fs.readFileSync('tests/extractor/fixtures/simple-single-listing.json', 'utf8'),
  );
  const pass = assertions.listingCountMatches(expected, { vars: { expected: FIXTURE } });
  assert.strictEqual(pass.pass, true);

  const fail = assertions.listingCountMatches([], { vars: { expected: FIXTURE } });
  assert.strictEqual(fail.pass, false);
  assert.match(fail.reason, /expected 1 listings, got 0/);
});

test('every committed fixture resolves', () => {
  const dir = 'tests/extractor/fixtures';
  const files = fs.readdirSync(dir).filter((f) => f.endsWith('.json'));
  assert.ok(files.length > 0, 'expected fixtures to exist');
  for (const file of files) {
    assert.doesNotThrow(
      () => assertions.listingCountMatches([], { vars: { expected: `file://${dir}/${file}` } }),
      `fixture ${file} should resolve`,
    );
  }
});

// --- non-array output must not be silently interpreted ---------------------

test('assertions reject non-array output from a crashed transform', async (t) => {
  // When the transform throws on malformed model JSON, promptfoo passes the raw
  // provider output through. A string has .length and (via String.prototype)
  // surprising array-ish behaviour, so these checks used to report a character
  // count as a listing count - "expected 1 listings, got 92".
  const FN = [
    'listingCountMatches',
    'structureMatches',
    'courtsMatch',
    'feesMatch',
    'contactsMatch',
    'normalizationMatches',
  ];
  const badOutputs = [
    ['raw string of malformed json', '[{"x":1}]}'],
    ['empty string', ''],
    ['null', null],
    ['undefined', undefined],
    ['plain object', { listing_type: 'play' }],
  ];

  for (const fn of FN) {
    for (const [name, output] of badOutputs) {
      await t.test(`${fn} / ${name}`, () => {
        assert.throws(
          () => assertions[fn](output, { vars: { expected: FIXTURE } }),
          /expected the transform to produce an array/,
          `${fn} must reject non-array output instead of interpreting it`,
        );
      });
    }
  }
});

test('a crashed transform does not look like a model failure', () => {
  // The misleading combination we want to eliminate: a string reported as
  // "count=0" by the gate, which reads as a wrong number of listings.
  let message = '';
  try {
    assertions.listingCountMatches('[{"x":1}]}', { vars: { expected: FIXTURE } });
  } catch (err) {
    message = err.message;
  }
  assert.match(message, /malformed JSON|transform/);
  assert.doesNotMatch(message, /got \d+ listings/);
});

// --- tier-identity comparison (fees/courts must not cascade) ----------------

test('a level_min error does not fabricate fee or court failures', () => {
  // Reproduced from a live run: 22 listings, every fee value correct, but one
  // level code wrong ("Mid-high beginners & above" mapped to HB instead of MB).
  // Under the old positional comparison that single error made fees and courts
  // fail too, because sorting by level_min reordered the arrays. Fixing the one
  // level code turned all three green.
  const tier = (over = {}) => ({
    listing_type: 'play',
    host_name: 'H',
    date: '2026-03-31',
    start_time: '19:30',
    end_time: '21:30',
    venue: 'hwi yoh cc',
    level_min: 'HB',
    level_max: null,
    level_raw: 'Mid-high beginners & above',
    fee_cents: 1000,
    courts: 2,
    ...over,
  });

  const expected = [
    tier({ level_min: 'MB', level_raw: 'Mid-high beginners & above', fee_cents: 1000, courts: 2 }),
    tier({ level_min: 'HB', level_max: 'LI', level_raw: 'High Beginners-LI', fee_cents: 1200, courts: 2 }),
    tier({ level_min: 'LI', level_raw: 'Low Intermediate & above', fee_cents: 1300, courts: 2 }),
  ];
  // Same tiers, same fees, same courts - only the MB tier's code is wrong.
  const actual = [
    tier({ level_min: 'HB', level_raw: 'Mid-high beginners & above', fee_cents: 1000, courts: 2 }),
    tier({ level_min: 'HB', level_max: 'LI', level_raw: 'High Beginners-LI', fee_cents: 1200, courts: 2 }),
    tier({ level_min: 'LI', level_raw: 'Low Intermediate & above', fee_cents: 1300, courts: 2 }),
  ];

  const expectedJson = JSON.stringify(expected);
  const ctx = (expectedArr) => ({ vars: { expected: expectedArr } });

  const fees = assertions.feesMatch(actual, ctx(expected));
  assert.strictEqual(fees.pass, true, `fees are all correct; must not fail. reason: ${fees.reason}`);

  const courts = assertions.courtsMatch(actual, ctx(expected));
  assert.strictEqual(courts.pass, true, `courts are all correct; must not fail. reason: ${courts.reason}`);

  // structure is the metric that SHOULD catch it, and still does.
  const structure = assertions.structureMatches(actual, ctx(expected));
  assert.strictEqual(structure.pass, false, 'structure must still fail on the wrong level code');

  assert.ok(expectedJson.length > 0);
});

test('fees still catch a genuine fee error', () => {
  const base = {
    listing_type: 'play',
    date: '2026-03-31',
    start_time: '19:30',
    end_time: '21:30',
    venue: 'hwi yoh cc',
    level_min: 'MB',
    level_max: null,
    level_raw: 'Mid-high beginners & above',
    fee_cents: 1000,
  };
  const expected = [base];
  const actual = [{ ...base, fee_cents: 1300 }];
  const result = assertions.feesMatch(actual, { vars: { expected } });
  assert.strictEqual(result.pass, false, 'a wrong fee must fail');
  assert.match(result.reason, /fee mismatch/);
});

test('fees report a wrong tier set rather than a fee difference', () => {
  const base = {
    listing_type: 'play',
    date: '2026-03-31',
    start_time: '19:30',
    end_time: '21:30',
    venue: 'hwi yoh cc',
    level_min: 'MB',
    level_max: null,
    level_raw: 'Mid-high beginners & above',
    fee_cents: 1000,
  };
  const expected = [base];
  const actual = [{ ...base, level_raw: 'something else entirely' }];
  const result = assertions.feesMatch(actual, { vars: { expected } });
  assert.strictEqual(result.pass, false);
  assert.match(result.reason, /tier set differs/);
});

// --- composite gate ---------------------------------------------------------

test('gate passes only when all hard metrics are 1', () => {
  const allPass = extractorScoring(
    { count: 1, structure: 1, courts: 1, fees: 1, contacts: 1, normalization: 1 },
    {},
  );
  assert.strictEqual(allPass.pass, true);
  assert.strictEqual(allPass.score, 1);

  for (const metric of ['count', 'structure', 'courts', 'fees', 'contacts']) {
    const scores = { count: 1, structure: 1, courts: 1, fees: 1, contacts: 1, normalization: 1 };
    scores[metric] = 0;
    const result = extractorScoring(scores, {});
    assert.strictEqual(result.pass, false, `${metric}=0 must fail the gate`);
    assert.match(result.reason, new RegExp(metric), `reason should name ${metric}`);
  }
});

test('gate fails closed when a hard metric never reported', () => {
  // Simulates an assertion that threw before returning a score.
  const result = extractorScoring(
    { count: 1, structure: 1, fees: 1, contacts: 1, normalization: 1 },
    {},
  );
  assert.strictEqual(result.pass, false, 'a missing hard metric must not be assumed to pass');
  assert.match(result.reason, /courts/);
  assert.match(result.reason, /did not report/);
  assert.deepStrictEqual(result.metadata.missingHard, ['courts']);
});

test('gate distinguishes harness failure from model failure in the reason', () => {
  const harnessBug = extractorScoring({ normalization: 1 }, {});
  assert.match(harnessBug.reason, /did not report/);
  assert.doesNotMatch(harnessBug.reason, /hard structure checks failed/);

  const modelBug = extractorScoring(
    { count: 1, structure: 1, courts: 1, fees: 0, contacts: 1, normalization: 1 },
    {},
  );
  assert.match(modelBug.reason, /hard structure checks failed/);
  assert.match(modelBug.reason, /fees=0/);
});

test('normalization is soft and contributes at most 0.2', () => {
  const withSoft = extractorScoring(
    { count: 1, structure: 1, courts: 1, fees: 1, contacts: 1, normalization: 0.5 },
    {},
  );
  assert.strictEqual(withSoft.pass, true, 'normalization must not gate a pass');
  assert.strictEqual(withSoft.score, 0.9);

  const hardFail = extractorScoring(
    { count: 1, structure: 0, courts: 1, fees: 1, contacts: 1, normalization: 1 },
    {},
  );
  assert.strictEqual(hardFail.score, 0.2, 'a hard failure caps the score at the normalization share');
});

test('gate tolerates a null/undefined namedScores object', () => {
  assert.doesNotThrow(() => extractorScoring(null, {}));
  assert.strictEqual(extractorScoring(undefined, {}).pass, false);
});
