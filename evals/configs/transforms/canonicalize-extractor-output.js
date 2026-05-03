module.exports = {
  canonicalizeExtractorOutput: (output) => {
    const normalizeJsonText = (rawOutput) => {
      let text = String(rawOutput ?? '').trim();

      const fencedMatch = text.match(/^```(?:json)?\s*([\s\S]*?)\s*```$/i);
      if (fencedMatch) {
        text = fencedMatch[1].trim();
      }

      if (!text.startsWith('[')) {
        const arrayStart = text.indexOf('[');
        const arrayEnd = text.lastIndexOf(']');
        if (arrayStart !== -1 && arrayEnd !== -1 && arrayEnd > arrayStart) {
          text = text.slice(arrayStart, arrayEnd + 1);
        }
      }

      return text;
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

    const parsed = JSON.parse(normalizeJsonText(output));
    if (!Array.isArray(parsed)) {
      throw new Error('Expected model output to be a JSON array');
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
