// Composite gate for the extractor eval.
//
// A test passes only when every hard metric scores exactly 1.0. Normalization is
// soft and contributes 20% on top of the 0.8 floor.
//
// Two robustness rules applied here:
//
//  1. A metric that is *missing* from namedScores is reported differently from
//     one that scored 0. Missing almost always means the assertion threw (for
//     example a fixture failed to load), and calling that "hard structure
//     checks failed" sends you looking for a model bug instead of a harness bug.
//
//  2. The failure reason names the specific failing metrics and their scores, so
//     the CLI/web output is actionable without opening results.json.

const HARD_METRICS = ['count', 'structure', 'courts', 'fees', 'contacts'];
const SOFT_METRICS = ['normalization'];

module.exports = {
  extractorScoring: (namedScores, context) => {
    const scores = namedScores ?? {};

    const hardScores = {};
    const missingHard = [];
    for (const metric of HARD_METRICS) {
      if (typeof scores[metric] === 'number') {
        hardScores[metric] = scores[metric];
      } else {
        missingHard.push(metric);
      }
    }

    const normalization = typeof scores.normalization === 'number' ? scores.normalization : 0;
    const missingSoft = SOFT_METRICS.filter((metric) => typeof scores[metric] !== 'number');

    // A hard metric that never reported cannot be assumed to have passed.
    // Failing closed is correct: an unevaluated gate is not a green gate.
    const failedHard = Object.entries(hardScores)
      .filter(([, value]) => value !== 1)
      .map(([metric]) => metric);

    const hardPass = missingHard.length === 0 && failedHard.length === 0;
    const score = hardPass ? 0.8 + 0.2 * normalization : 0.2 * normalization;

    let reason;
    if (missingHard.length > 0) {
      reason =
        `hard metric(s) did not report: ${missingHard.join(', ')} — ` +
        'likely an assertion error rather than a model failure; check the run log';
    } else if (!hardPass) {
      const detail = failedHard.map((metric) => `${metric}=${hardScores[metric]}`).join(', ');
      reason = `hard structure checks failed (${detail})`;
    } else if (missingSoft.length > 0) {
      reason = `hard structure checks passed; ${missingSoft.join(', ')} did not report`;
    } else {
      reason =
        normalization === 1
          ? 'hard structure checks passed; normalization matched'
          : 'hard structure checks passed; normalization differences only';
    }

    return {
      pass: hardPass,
      score,
      reason,
      // Surfaced in the results file for triage.
      namedScores: { ...hardScores, normalization },
      metadata: {
        failedHard,
        missingHard,
        hardPass,
        normalization,
      },
    };
  },
};
