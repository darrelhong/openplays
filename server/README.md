# OpenPlays Server

Listens to Telegram group messages, extracts structured sports session listings using a local LLM, and outputs parsed play data.

## Prerequisites

- Go 1.26+
- [LM Studio](https://lmstudio.ai/) (or any OpenAI-compatible API endpoint)
- Telegram API credentials from [my.telegram.org](https://my.telegram.org)

## Setup

```bash
cp .env.example .env
```

Fill in `.env`:

`LLM_MODEL` can be left empty to use whatever model is loaded in LM Studio. Set `LLM_API_KEY` if using a cloud provider like OpenAI.

## Listener

Connects to Telegram and parses incoming messages in real-time.

```bash
go run ./cmd/listener/
```

Log to file while watching output:

```bash
go run ./cmd/listener/ 2>&1 | tee -a listener_output.log
```

On first run, Telegram will prompt for a verification code.

## Test parsing

Pipe a message through the LLM pipeline without needing Telegram. Useful for testing different LLM providers or prompt changes.

```bash

# From file with sender name
SENDER_NAME="Daniel" go run ./tools/parsetest/ < example_messages.txt

## With a different LLM provider
LLM_BASE_URL=https://api.openai.com/v1 \
LLM_MODEL=gpt-4o-mini \
LLM_API_KEY=sk-... \
go run ./tools/parsetest/ < example_messages.txt
```
