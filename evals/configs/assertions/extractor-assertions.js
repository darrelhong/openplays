const fs = require('fs');
const path = require('path');

function loadExpected(expected) {
  let items;

  if (Array.isArray(expected)) {
    items = expected;
  } else if (typeof expected !== 'string') {
    return [];
  } else {
    const trimmed = expected.trim();

    if (trimmed.startsWith('[')) {
      items = JSON.parse(trimmed);
    } else if (!trimmed.startsWith('file://')) {
      return [];
    } else {
      const relativePath = trimmed.slice('file://'.length);
      const candidatePaths = [
        path.resolve(process.cwd(), relativePath),
        path.resolve(__dirname, '..', '..', relativePath),
        path.resolve(__dirname, '..', '..', path.basename(relativePath)),
      ];

      items = null;
      for (const candidatePath of candidatePaths) {
        if (fs.existsSync(candidatePath)) {
          items = JSON.parse(fs.readFileSync(candidatePath, 'utf8'));
          break;
        }
      }
      if (!items) {
        throw new Error(`Unable to load expected fixture: ${trimmed}`);
      }
    }
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

module.exports = {
  listingCountMatches: (output, context) => {
    const expected = loadExpected(context.vars.expected);
    return {
      pass: Array.isArray(output) && Array.isArray(expected) && output.length === expected.length,
      score: Array.isArray(output) && Array.isArray(expected) && output.length === expected.length ? 1 : 0,
      reason: `expected ${expected?.length ?? 0} listings, got ${output?.length ?? 0}`,
    };
  },

  structureMatches: (output, context) => {
    const expected = loadExpected(context.vars.expected);
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
    const expected = loadExpected(context.vars.expected);
    const actual = (Array.isArray(output) ? output : []).map((item) => item?.courts ?? null);
    const wanted = (Array.isArray(expected) ? expected : []).map((item) => item?.courts ?? null);
    const pass = JSON.stringify(actual) === JSON.stringify(wanted);
    return {
      pass,
      score: pass ? 1 : 0,
      reason: pass ? 'courts match' : `expected ${JSON.stringify(wanted)}, got ${JSON.stringify(actual)}`,
    };
  },

  feesMatch: (output, context) => {
    const expected = loadExpected(context.vars.expected);
    const pick = (item) => ({
      fee_cents: item?.fee_cents ?? null,
      fee_male_cents: item?.fee_male_cents ?? null,
      fee_female_cents: item?.fee_female_cents ?? null,
    });
    const actual = (Array.isArray(output) ? output : []).map(pick);
    const wanted = (Array.isArray(expected) ? expected : []).map(pick);
    const pass = JSON.stringify(actual) === JSON.stringify(wanted);
    return {
      pass,
      score: pass ? 1 : 0,
      reason: pass ? 'fees match' : `expected ${JSON.stringify(wanted)}, got ${JSON.stringify(actual)}`,
    };
  },

  contactsMatch: (output, context) => {
    const expected = loadExpected(context.vars.expected);
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
    const expected = loadExpected(context.vars.expected);
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
