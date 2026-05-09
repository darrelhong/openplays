# Extractor Prompt Evals

Evaluates how well different LLM models extract structured badminton session listings from raw Telegram messages using [promptfoo](https://promptfoo.dev).

## Setup

```bash
npm install
cp .env.example .env  # add your GOOGLE_API_KEY
```

## Run

```bash
npx promptfoo eval
npx promptfoo view   # open web UI to compare results
```

## Structure

```
evals/
  promptfooconfig.yaml          # root config — wires prompts, providers, tests
  extractor_prompt.yaml         # system + user prompt (chat format)
  configs/
    default-test.yaml           # shared test settings (transform + scoring)
    providers/
      google.yaml               # model definitions (Gemini variants)
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

Add model definitions to `configs/providers/google.yaml` or create a new provider file and reference it in `promptfooconfig.yaml` under `providers:`.
