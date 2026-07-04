/// <reference types="node" />
import { defineConfig } from '@playwright/test';

// Preview port is overridable (PREVIEW_PORT) so a local run can avoid a port
// already held by another preview/UI session.
const previewPort = Number(process.env.PREVIEW_PORT ?? 4173);

// The e2e build bakes its own API_BASE_URL so the in-process mock backend
// (which listens on this port) never contends with a live dev API / air
// session on 8080. This config is the single source of truth for the e2e
// environment — no env wrapper needed on the npm scripts. Overridable via
// MOCK_API_PORT.
const mockApiPort = Number(process.env.MOCK_API_PORT ?? 8090);
process.env.MOCK_API_PORT = String(mockApiPort);

export default defineConfig({
	webServer: {
		command: `npm run build && npm run preview -- --port ${previewPort}`,
		port: previewPort,
		reuseExistingServer: !process.env.CI,
		env: {
			API_BASE_URL: `http://localhost:${mockApiPort}`,
			COOKIE_SECURE: process.env.COOKIE_SECURE ?? 'false'
		}
	},
	testMatch: '**/*.e2e.{ts,js}',
	// Specs share one mock backend port, so they must not run concurrently
	// and contend for it.
	workers: 1
});
