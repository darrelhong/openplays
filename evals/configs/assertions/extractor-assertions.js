const fs = require('fs');
const path = require('path');

/**
 * Resolve `context.vars.expected` to an array of expected listings.
 *
 * Every failure path throws. This is deliberate and important: previously a
 * missing or malformed `expected` silently became `[]`, which made an empty
 * model response compare equal to an empty fixture and score a *passing* 1.0.
 * A typo'd fixture path could therefore turn the whole suite green while the
 * extractor returned nothing. A test harness that can pass on absent ground
 * truth is worse than no harness — it manufactures false confidence.
 *
 * Known-good inputs are: a JSON array, a JSON array as a string, or a
 * `file://` path to a fixture containing a JSON array.
 */
function loadExpected(expected, assertionName = 'assertion') {
  const fail = (why) => {
    throw new Error(
      `[${assertionName}] cannot resolve expected fixtures: ${why}. ` +
        `Got: ${JSON.stringify(expected)?.slice(0, 200)}`,
    );
  };

  if (expected === undefined || expected === null) {
    fail('`expected` is missing from the test vars');
  }

  let items;
  if (Array.isArray(expected)) {
    items = expected;
  } else if (typeof expected !== 'string') {
    fail(`expected must be an array or file:// string, got ${typeof expected}`);
  } else {
    const trimmed = expected.trim();

    if (trimmed === '') {
      fail('`expected` is an empty string');
    } else if (trimmed.startsWith('[')) {
      try {
        items = JSON.parse(trimmed);
      } catch (err) {
        fail(`inline expected is not valid JSON: ${err.message}`);
      }
    } else if (!trimmed.startsWith('file://')) {
      fail('expected must start with "[" or "file://"');
    } else {
      const relativePath = trimmed.slice('file://'.length);
      const candidatePaths = [
        path.resolve(process.cwd(), relativePath),
        path.resolve(__dirname, '..', '..', relativePath),
        path.resolve(__dirname, '..', '..', path.basename(relativePath)),
      ];

      const found = candidatePaths.find((candidatePath) => fs.existsSync(candidatePath));
      if (!found) {
        fail(`fixture file not found (tried ${candidatePaths.join(', ')})`);
      }
      try {
        items = JSON.parse(fs.readFileSync(found, 'utf8'));
      } catch (err) {
        fail(`fixture ${found} is not valid JSON: ${err.message}`);
      }
    }
  }

  if (!Array.isArray(items)) {
    fail('fixture must contain a JSON array of listings');
  }

  // Sort expected the same way the canonicalizer sorts output
  return (Array.isArray(items) ? items : []).sort((a, b) =>
    [
      a?.date ?? '',
      a?.start_time ?? '',
      a?.end_time ?? '',
      (a?.venue ?? '').toLowerCase(),
      a?.level_min ?? '',
      a?.level_max ?? '',
      String(a?.fee_cents ?? ''),
    ]
      .join('|')
      .localeCompare(
        [
          b?.date ?? '',
          b?.start_time ?? '',
          b?.end_time ?? '',
          (b?.venue ?? '').toLowerCase(),
          b?.level_min ?? '',
          b?.level_max ?? '',
          String(b?.fee_cents ?? ''),
        ].join('|'),
      ),
  );
}

/**
 * Assertions must only ever see the canonical array produced by the transform.
 *
 * When the transform throws (malformed model JSON — the truncation case that
 * occurred in a live run), promptfoo leaves the 
 * raw provider output in place and hands *that* to the assertions. A string
 * silently "works" for a surprising number of these checks because
 * `output.length` and `output.map` behave like an array's, producing nonsense
 * such as "expected 1 listings, got 92" (the string's character count). The
 * resulting "hard structure checks failed (count=0)" sends triage looking for a
 * model bug when the real fault is a transform crash.
 *
 * Failing closed with an explicit reason keeps the two distinct.
 */
function requireCanonicalArray(output, assertionName) {
  if (Array.isArray(output)) {
    return output;
  }
  const seen =
    typeof output === 'string'
      ? `a string of ${output.length} chars (starts ${JSON.stringify(output.slice(0, 60))})`
      : `${output === null ? 'null' : typeof output}`;
  throw new Error(
    `[${assertionName}] expected the transform to produce an array of listings, got ${seen}. ` +
      'This usually means the model returned malformed JSON and the transform threw; ' +
      'check the run log for a parse error rather than reading the metric as a model failure.',
  );
}

module.exports = {
  listingCountMatches: (output, context) => {
    const expected = loadExpected(context.vars.expected, 'listingCountMatches');
    requireCanonicalArray(output, 'listingCountMatches');
    return {
      pass: Array.isArray(output) && Array.isArray(expected) && output.length === expected.length,
      score: Array.isArray(output) && Array.isArray(expected) && output.length === expected.length ? 1 : 0,
      reason: `expected ${expected?.length ?? 0} listings, got ${output?.length ?? 0}`,
    };
  },

  structureMatches: (output, context) => {
    const expected = loadExpected(context.vars.expected, 'structureMatches');
    requireCanonicalArray(output, 'structureMatches');
    const makeKey = (item) =>
      [item?.date ?? '', item?.start_time ?? '', item?.end_time ?? '', (item?.venue ?? '').toLowerCase(), item?.level_min ?? '', item?.level_max ?? ''].join('|');
    const actualKeys = (Array.isArray(output) ? output : []).map(makeKey);
    const expectedKeys = (Array.isArray(expected) ? expected : []).map(makeKey);
    const pass = JSON.stringify(actualKeys) === JSON.stringify(expectedKeys);
    return {
      pass,
      score: pass ? 1 : 0,
      reason: pass ? 'listing structure matches' : `expected ${JSON.stringify(expectedKeys)}, got ${JSON.stringify(actualKeys)}`,
    };
  },

  courtsMatch: (output, context) => {
    const expected = loadExpected(context.vars.expected, 'courtsMatch');
    requireCanonicalArray(output, 'courtsMatch');
    // Same tier-identity reasoning as feesMatch: compare the court count of each
    // tier rather than the tier at each sorted position, so a level_min error in
    // one tier cannot manufacture a courts failure elsewhere.
    const tierKey = (item) =>
      [
        item?.date ?? '',
        item?.start_time ?? '',
        item?.end_time ?? '',
        (item?.venue ?? '').toLowerCase(),
        (item?.level_raw ?? '').trim().toLowerCase(),
      ].join('|');
    const indexCourts = (items) => {
      const map = new Map();
      for (const item of items) map.set(tierKey(item), item?.courts ?? null);
      return map;
    };
    const actual = indexCourts(Array.isArray(output) ? output : []);
    const wanted = indexCourts(Array.isArray(expected) ? expected : []);

    const mismatches = [];
    for (const [key, wantedCourts] of wanted) {
      if (actual.get(key) !== wantedCourts) {
        mismatches.push({ tier: key, expected: wantedCourts, got: actual.get(key) ?? null });
      }
    }
    const pass = mismatches.length === 0;
    return {
      pass,
      score: pass ? 1 : 0,
      reason: pass ? 'courts match' : `courts mismatch on ${mismatches.length} tier(s): ${JSON.stringify(mismatches.slice(0, 3))}`,
      metadata: { mismatches },
    };
  },

  feesMatch: (output, context) => {
    const expected = loadExpected(context.vars.expected, 'feesMatch');
    requireCanonicalArray(output, 'feesMatch');
    const pick = (item) => ({
      fee_cents: item?.fee_cents ?? null,
      fee_male_cents: item?.fee_male_cents ?? null,
      fee_female_cents: item?.fee_female_cents ?? null,
    });
    // Compare fees against the tier they belong to, not against the tier at the
    // same sorted position.
    //
    // The output and fixtures are both sorted by date|time|venue|level_min|...
    // so any error in level_min reorders the arrays. Under a positional compare
    // that single level error also fabricated fee mismatches: a live run reported
    // fees=0 on a response whose 22 fee values were all correct, and fixing the
    // one level code turned both metrics green. A metrics report must not
    // multiply one mistake into three - it misdirects triage.
    //
    // Tier identity is (date, start, end, venue, level_raw). level_raw is the
    // level text as written, so it stays correct even when the model maps it to
    // the wrong code - which is exactly the error structure is there to catch.
    const tierKey = (item) =>
      [
        item?.date ?? '',
        item?.start_time ?? '',
        item?.end_time ?? '',
        (item?.venue ?? '').toLowerCase(),
        (item?.level_raw ?? '').trim().toLowerCase(),
      ].join('|');

    const indexTiers = (items) => {
      const map = new Map();
      for (const item of items) {
        const key = tierKey(item);
        if (!map.has(key)) map.set(key, []);
        map.get(key).push(pick(item));
      }
      return map;
    };

    const actualTiers = indexTiers(Array.isArray(output) ? output : []);
    const wantedTiers = indexTiers(Array.isArray(expected) ? expected : []);

    // Keys present on one side only mean the tier itself is missing or extra.
    // Report that explicitly rather than as a fee difference.
    const missingTiers = [...wantedTiers.keys()].filter((k) => !actualTiers.has(k));
    const extraTiers = [...actualTiers.keys()].filter((k) => !wantedTiers.has(k));
    const unexpectedTiers = [...new Set([...missingTiers, ...extraTiers])].length > 0;

    const feeMismatches = [];
    for (const [key, wanted] of wantedTiers) {
      const actual = actualTiers.get(key);
      if (!actual) continue;
      const norm = (rows) => JSON.stringify(rows.slice().sort((a, b) => JSON.stringify(a).localeCompare(JSON.stringify(b))));
      if (norm(actual) !== norm(wanted)) {
        feeMismatches.push({ tier: key, expected: wanted, got: actual });
      }
    }

    const pass = !unexpectedTiers && feeMismatches.length === 0;
    let reason;
    if (pass) {
      reason = 'fees match';  
    } else if (unexpectedTiers) {
      reason =
        `tier set differs, so fees cannot be compared: ` +
        `${missingTiers.length} expected tier(s) missing, ${extraTiers.length} unexpected. ` +
        `missing=${JSON.stringify(missingTiers.slice(0, 3))} unexpected=${JSON.stringify(extraTiers.slice(0, 3))}`;
    } else {
      reason = `fee mismatch on ${feeMismatches.length} tier(s): ${JSON.stringify(feeMismatches.slice(0, 3))}`;
    }

    return {
      pass,
      score: pass ? 1 : 0,
      reason,
      metadata: { feeMismatches, missingTiers, extraTiers },
    };
  },

  contactsMatch: (output, context) => {
    const expected = loadExpected(context.vars.expected, 'contactsMatch');
    requireCanonicalArray(output, 'contactsMatch');
    const normalizeContacts = (contacts) =>
      [...new Set((Array.isArray(contacts) ? contacts : [])
        .map((contact) => contact?.value ?? null)
        .filter((value) => value !== null)
        .map((value) => String(value).trim()))]
        .sort();

    const actual = (Array.isArray(output) ? output : []).map((item) => normalizeContacts(item?.contacts));
    const wanted = (Array.isArray(expected) ? expected : []).map((item) => normalizeContacts(item?.contacts));
    // Pass if every expected contact value appears in actual (extra contacts are tolerated)
    const pass = wanted.every((expectedContacts, index) => {
      const actualContacts = actual[index] ?? [];
      return expectedContacts.every((value) => actualContacts.includes(value));
    });
    return {
      pass,
      score: pass ? 1 : 0,
      reason: pass ? 'contact values match' : `expected ${JSON.stringify(wanted)}, got ${JSON.stringify(actual)}`,
    };
  },

  normalizationMatches: (output, context) => {
    const expected = loadExpected(context.vars.expected, 'normalizationMatches');
    requireCanonicalArray(output, 'normalizationMatches');
    const equivalentGenderPref = (actual, wanted) => {
      const normalizedActual = actual ?? null;
      const normalizedWanted = wanted ?? null;
      return normalizedActual === normalizedWanted || (
        (normalizedActual === null || normalizedActual === 'all') &&
        (normalizedWanted === null || normalizedWanted === 'all')
      );
    };
    const normalizedDetails = (value) => String(value ?? '').toLowerCase();
    const detailsMatch = (actual, wanted) => {
      const normalizedActual = normalizedDetails(actual);
      const normalizedWanted = normalizedDetails(wanted);
      if (normalizedActual === normalizedWanted) {
        return true;
      }
      if (normalizedWanted === '') {
        return normalizedActual === '';
      }
      return normalizedWanted
        .split(',')
        .map((part) => part.trim().toLowerCase())
        .filter(Boolean)
        .every((part) => normalizedActual.includes(part));
    };
    const pick = (item) => ({
      gender_pref: item?.gender_pref ?? null,
      shuttle: item?.shuttle ?? null,
      details: item?.details ?? null,
    });
    const actual = (Array.isArray(output) ? output : []).map(pick);
    const wanted = (Array.isArray(expected) ? expected : []).map(pick);
    const total = wanted.length * 3;
    const matched = wanted.reduce((sum, expectedItem, index) => {
      const actualItem = actual[index] ?? {};
      return sum +
        (equivalentGenderPref(actualItem.gender_pref, expectedItem.gender_pref) ? 1 : 0) +
        (actualItem.shuttle === expectedItem.shuttle ? 1 : 0) +
        (detailsMatch(actualItem.details, expectedItem.details) ? 1 : 0);
    }, 0);
    const score = total === 0 ? 1 : matched / total;
    const pass = score === 1;
    return {
      pass,
      score,
      reason: pass ? 'normalization fields match' : `expected ${JSON.stringify(wanted)}, got ${JSON.stringify(actual)}`,
    };
  },
};
