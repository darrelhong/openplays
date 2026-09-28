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
    // So: strip a ``` fence, trim, read ONE JSON value, and ignore anything
    // after it. No array slicing, no brace balancing, no trailing-comma repair.
    const normalizeJsonText = (rawOutput) => {
      let text = String(rawOutput ?? "").trim();

      if (text.startsWith("```")) {
        const newlineIdx = text.indexOf("\n");
        if (newlineIdx !== -1) {
          text = text.slice(newlineIdx + 1);
        }
        const fenceEnd = text.lastIndexOf("```");
        if (fenceEnd !== -1) {
          text = text.slice(0, fenceEnd);
        }
      }

      return text.trim();
    };

    // Mirrors jsonValuePrefix() in extractor.go, which uses
    // json.Decoder.Decode to read the first complete JSON value and ignores
    // whatever follows.
    //
    // Why: the model sometimes finishes a valid array and then emits a stray
    // "}". Production used to reject the entire response for that
    // ("invalid character '}' after top-level value") and discard every
    // listing, even though the array was complete and correct.
    //
    // Truncated output must STILL fail: a half-written array has no complete
    // leading value, so it throws here exactly as it does in production. The
    // loosening only covers "a valid value plus trailing junk".
    const firstJsonValue = (text) => {
      const isWhitespace = (c) =>
        c === " " || c === "\t" || c === "\n" || c === "\r";
      const n = text.length;
      let i = 0;
      while (i < n && isWhitespace(text[i])) i++;

      if (i >= n) {
        throw new SyntaxError("unexpected end of JSON input");
      }

      const start = i;
      const first = text[i];

      if (first === "{" || first === "[") {
        // Scan to the matching close, honouring strings and escapes so a brace
        // inside a value (e.g. a venue named "Hall {A}") does not unbalance it.
        const stack = [first];
        let inString = false;
        let escaped = false;
        i++;
        while (i < n && stack.length > 0) {
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
            const open = stack.pop();
            if ((ch === "}" && open !== "{") || (ch === "]" && open !== "[")) {
              throw new SyntaxError(`unexpected character '${ch}'`);
            }
          }
          i++;
        }
        if (stack.length > 0) {
          throw new SyntaxError("unexpected end of JSON input");
        }
      } else if (first === '"') {
        i++;
        let escaped = false;
        let closed = false;
        while (i < n) {
          const ch = text[i];
          if (escaped) escaped = false;
          else if (ch === "\\") escaped = true;
          else if (ch === '"') {
            closed = true;
            i++;
            break;
          }
          i++;
        }
        if (!closed) throw new SyntaxError("unexpected end of JSON input");
      } else {
        // Number, true, false, null: read the token only.
        while (i < n && /[0-9eE+.\-a-zA-Z]/.test(text[i])) i++;
      }

      // JSON.parse both validates the slice and produces the value. A balanced
      // but invalid body (e.g. a trailing comma) throws here, as in Go.
      return JSON.parse(text.slice(start, i));
    };

    const canonicalContacts = (contacts) =>
      (Array.isArray(contacts) ? contacts : [])
        .map((contact) => ({
          type: contact?.type ?? null,
          value: contact?.value ?? null,
        }))
        .sort((a, b) =>
          `${a.type}:${a.value}`.localeCompare(`${b.type}:${b.value}`),
        );

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
      currency: item?.currency ?? "SGD",
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
      parsed = firstJsonValue(normalized);
    } catch (err) {
      throw new Error(
        `parse LLM JSON output: ${err.message} (raw: ${normalized.slice(0, 200)})`,
      );
    }
    // Mirror Go's decode into []model.ParsedPlayCandidate exactly.
    //
    // json.Unmarshal accepts a top-level null for a slice and yields an empty
    // slice, so `null` is zero listings - not an error. Element types are
    // stricter: Go refuses to put a non-object into ParsedPlayCandidate, so an
    // array containing [] or "a" or 123 is a hard parse failure in production.
    // The transform must fail there too, or the eval would report a parsed
    // listing for output the listener actually rejects and retries.
    if (parsed === null) {
      return [];
    }
    if (Array.isArray(parsed)) {
      parsed.forEach((element, idx) => {
        // null is allowed: Go decodes it to a zero-value candidate.
        if (element === null) return;
        if (typeof element !== "object" || Array.isArray(element)) {
          const got = Array.isArray(element) ? "array" : typeof element;
          throw new Error(
            `parse LLM JSON output: json: cannot unmarshal ${got} into Go value of type model.ParsedPlayCandidate (element ${idx})`,
          );
        }
      });
    } else {
      // Single-object fallback: production retries the decode as one struct.
      if (typeof parsed !== "object") {
        throw new Error(
          `parse LLM JSON output: expected an array or object, got ${typeof parsed}`,
        );
      }
      parsed = [parsed];
    }

    return parsed
      .map(canonicalItem)
      .sort((a, b) =>
        [
          a.date ?? "",
          a.start_time ?? "",
          a.end_time ?? "",
          a.venue ?? "",
          a.level_min ?? "",
          a.level_max ?? "",
          String(a.fee_cents ?? ""),
        ]
          .join("|")
          .localeCompare(
            [
              b.date ?? "",
              b.start_time ?? "",
              b.end_time ?? "",
              b.venue ?? "",
              b.level_min ?? "",
              b.level_max ?? "",
              String(b.fee_cents ?? ""),
            ].join("|"),
          ),
      );
  },
};
