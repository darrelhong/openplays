# sv

Everything you need to build a Svelte project, powered by [`sv`](https://github.com/sveltejs/cli).

## Creating a project

If you're seeing this, you've probably already done this step. Congrats!

```sh
# create a new project
npx sv create my-app
```

To recreate this project with the same configuration:

```sh
# recreate this project
pnpm dlx sv@0.14.1 create --template minimal --types ts --add prettier eslint vitest="usages:component,unit" playwright mcp="ide:opencode,vscode+setup:local" --install pnpm web
```

## Developing

Once you've installed dependencies with `pnpm install`, start a development server:

```sh
pnpm dev
```

## API Types

TypeScript types are auto-generated from the Go API's OpenAPI spec. The generated file is `src/lib/api/types.gen.ts`.

To regenerate after changing the API:

```sh
# 1. Start the Go API server (in the server/ directory)
cd ../server && go run ./cmd/api/

# 2. Generate types (in the web/ directory)
pnpm gen:types
```

This fetches `http://localhost:8080/openapi.json` and generates typed request/response interfaces. The config is in `redocly.yaml`.

## Building

To create a production version of your app:

```sh
npm run build
```

You can preview the production build with `npm run preview`.

> To deploy your app, you may need to install an [adapter](https://svelte.dev/docs/kit/adapters) for your target environment.
