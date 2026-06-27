/// <reference types="node" />
import { defineConfig } from '@playwright/test';

// Preview port is overridable (PREVIEW_PORT) so a local run can avoid a port
// already held by another preview/UI session.
const previewPort = Number(process.env.PREVIEW_PORT ?? 4173);

export default defineConfig({
	webServer: {
		command: `npm run build && npm run preview -- --port ${previewPort}`,
		port: previewPort,
		reuseExistingServer: !process.env.CI
	},
	testMatch: '**/*.e2e.{ts,js}',
	// Specs mock the backend by listening on the baked API_BASE_URL port (8080),
	// so they must not run concurrently and contend for it.
	workers: 1
});
