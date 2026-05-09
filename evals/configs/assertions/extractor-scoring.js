module.exports = {
  extractorScoring: (namedScores, context) => {
    const count = namedScores.count ?? 0;
    const structure = namedScores.structure ?? 0;
    const courts = namedScores.courts ?? 0;
    const fees = namedScores.fees ?? 0;
    const contacts = namedScores.contacts ?? 0;
    const normalization = namedScores.normalization ?? 0;

    const hardPass = structure === 1 && count === 1 && courts === 1 && fees === 1 && contacts === 1;
    const score = hardPass ? 0.8 + 0.2 * normalization : 0.2 * normalization;

    return {
      pass: hardPass,
      score,
      reason: hardPass
        ? normalization === 1
          ? 'hard structure checks passed; normalization matched'
          : 'hard structure checks passed; normalization differences only'
        : 'hard structure checks failed',
    };
  },
};
