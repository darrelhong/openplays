module.exports = {
  canonicalizeExtractorOutput: (output) => {
    // Mirrors cleanJSONResponse() in server/internal/listener/pipeline/extractor.go.
    //
    // This is deliberately a faithful port, not a lenient parser. The eval exists
    // to predict production: if it repairs output that production rejects, it
    // reports green on a run that fails in the listener, and no one finds out
    // until a Telegram message silently produces nothing. If it were stricter, it
    // would penalise models for output production handles fine.
    //
    // So: strip a ``` fence, trim, and hand the rest to JSON.parse. No array
    // slicing, no brace balancing, no trailing-comma repair. If the model emits
    // malformed JSON we want a loud failure here, and the guard belongs in the
    // token cap or the prompt — not in the parser.
    const normalizeJsonText = (rawOutput) => {
      let text = String(rawOutput ?? '').trim();

      if (text.startsWith('```')) {
        const newlineIdx = text.indexOf('\n');
        if (newlineIdx !== -1) {
          text = text.slice(newlineIdx + 1);
        }
        const fenceEnd = text.lastIndexOf('```');
        if (fenceEnd !== -1) {
          text = text.slice(0, fenceEnd);
        }
      }

      return text.trim();
    };

    const canonicalContacts = (contacts) =>
      (Array.isArray(contacts) ? contacts : [])
        .map((contact) => ({
          type: contact?.type ?? null,
          value: contact?.value ?? null,
        }))
        .sort((a, b) => `${a.type}:${a.value}`.localeCompare(`${b.type}:${b.value}`));

    const canonicalItem = (item) => ({
      listing_type: item?.listing_type ?? null,
      host_name: item?.host_name ?? null,
      game_type: item?.game_type ?? null,
      date: item?.date ?? null,
      start_time: item?.start_time ?? null,
      end_time: item?.end_time ?? null,
      venue: item?.venue ? String(item.venue).toLowerCase().trim() : null,
      level_min: item?.level_min ?? null,
      level_max: item?.level_max ?? null,
      level_raw: item?.level_raw ?? null,
      level_male_min: item?.level_male_min ?? null,
      level_male_max: item?.level_male_max ?? null,
      level_female_min: item?.level_female_min ?? null,
      level_female_max: item?.level_female_max ?? null,
      fee_cents: item?.fee_cents ?? null,
      fee_male_cents: item?.fee_male_cents ?? null,
      fee_female_cents: item?.fee_female_cents ?? null,
      currency: item?.currency ?? 'SGD',
      max_players: item?.max_players ?? null,
      slots_left: item?.slots_left ?? null,
      courts: item?.courts ?? null,
      gender_pref: item?.gender_pref ?? null,
      shuttle: item?.shuttle ?? null,
      air_con: item?.air_con ?? null,
      details: item?.details ?? null,
      contacts: canonicalContacts(item?.contacts),
    });

    // Mirrors parseResponse(): array first, then a single-object fallback.
    // Production accepts a bare object when the model returns one listing
    // unwrapped, so the eval must accept it too.
    const normalized = normalizeJsonText(output);
    let parsed;
    try {
      parsed = JSON.parse(normalized);
    } catch (err) {
      throw new Error(`parse LLM JSON output: ${err.message} (raw: ${normalized.slice(0, 200)})`);
    }
    if (!Array.isArray(parsed)) {
      if (parsed === null || typeof parsed !== 'object') {
        throw new Error(`parse LLM JSON output: expected an array or object, got ${typeof parsed}`);
      }
      parsed = [parsed];
    }

    return parsed
      .map(canonicalItem)
      .sort((a, b) =>
        [
          a.date ?? '',
          a.start_time ?? '',
          a.end_time ?? '',
          a.venue ?? '',
          a.level_min ?? '',
          a.level_max ?? '',
          String(a.fee_cents ?? ''),
        ]
          .join('|')
          .localeCompare(
            [
              b.date ?? '',
              b.start_time ?? '',
              b.end_time ?? '',
              b.venue ?? '',
              b.level_min ?? '',
              b.level_max ?? '',
              String(b.fee_cents ?? ''),
            ].join('|'),
          ),
      );
  },
};
