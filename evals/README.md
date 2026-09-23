# Extractor Prompt Evals

Evaluates how well different LLM models extract structured badminton session listings from raw Telegram messages using [promptfoo](https://promptfoo.dev).

## Setup

```bash
npm install
cp .env.example .env   # then fill in GOOGLE_API_KEY and/or OPENAI_API_KEY
```

## Run

```bash
npm run eval              # both provider groups
npm run eval:google       # Gemini only
npm run eval:openai       # OpenAI only (gpt-6-luna)
npm run eval:view         # open web UI to compare results
```

`npm run eval` passes `--env-file .env`, so the keys never need to be exported
into your shell. To check config without spending anything:

```bash
npm run eval:validate
```

## API keys

Keys live in `evals/.env`, which is gitignored. `.env.example` is committed and
lists the variables:

| Variable | Used by |
| --- | --- |
| `GOOGLE_API_KEY` | `configs/providers/google.yaml` (Gemini) |
| `OPENAI_API_KEY` | `configs/providers/openai.yaml` (gpt-6-luna) |

Set only the ones you need — a provider whose key is missing produces ERROR
results for its rows, which is visible in the output rather than silent.

## Structure

```
evals/
  promptfooconfig.yaml          # root config — wires prompts, providers, tests
  extractor_prompt.yaml         # system + user prompt (chat format)
  configs/
    default-test.yaml           # shared test settings (transform + scoring)
    providers/
      google.yaml               # model definitions (Gemini variants)
      openai.yaml               # model definitions (gpt-6-luna, OpenAI)
    assertions/
      extractor-assertions.js   # 6 named assertion functions
      extractor-scoring.js      # composite pass/fail scoring
    transforms/
      canonicalize-extractor-output.js  # normalizes model JSON output
  tests/
    extractor/
      complex-listings.yaml     # multi-listing expansion tests
      filtering.yaml            # skip/filter logic tests
      fixtures/                 # expected JSON output per test case
```

## How Scoring Works

Six metrics per test case:

| Metric | Type | What it checks |
|--------|------|---------------|
| `count` | hard | Correct number of listings |
| `structure` | hard | Correct composite key (date + time + venue + level_min/max) |
| `courts` | hard | Correct court counts |
| `fees` | hard | Correct fee fields |
| `contacts` | hard | Correct contact values |
| `normalization` | soft | gender_pref, shuttle, details (fuzzy) |

**Hard gate:** All 5 hard metrics must score 1.0 to pass. Score = 0.8 + 0.2 * normalization. If any hard metric fails, score = 0.2 * normalization (max 0.2).

## Adding a Test Case

1. Create a fixture JSON file in `tests/extractor/fixtures/`
2. Add a test entry to an existing YAML file in `tests/extractor/` (or create a new one)
3. Reference the fixture via `expected: file://tests/extractor/fixtures/your-fixture.json` in vars
4. If you created a new test file, add it to `promptfooconfig.yaml` under `tests:`

## Adding a Provider

Add model definitions to an existing file in `configs/providers/` or create a new
one and reference it in `promptfooconfig.yaml` under `providers:`.

### Notes on the OpenAI provider

`configs/providers/openai.yaml` uses the explicit `openai:chat:<model>` endpoint
prefix, matching what `server/internal/listener/pipeline/extractor.go` actually
calls (Chat Completions), rather than a bare model ID which may route to the
Responses API.

`gpt-6-luna` is a reasoning model, so no `temperature` is set — reasoning models
reject it. Reasoning tokens are billed as output, and for a structured extraction
task they are mostly wasted work, so `reasoning_effort: none` is the default
here. Commented entries in that file let you A/B `low` reasoning and the
`gpt-5.4-nano` baseline against it.
