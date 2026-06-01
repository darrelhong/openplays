import { defineConfig } from '@playwright/test';

export default defineConfig({
	webServer: { command: 'npm run build && npm run preview', port: 4173 },
	testMatch: '**/*.e2e.{ts,js}',
	// Specs mock the backend by listening on the baked API_BASE_URL port (8080),
	// so they must not run concurrently and contend for it.
	workers: 1
});
