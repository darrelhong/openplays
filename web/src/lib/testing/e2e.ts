import type { Page } from '@playwright/test';

/**
 * The global "Enable push notifications" prompt (rendered via the layout)
 * auto-opens after an async push-state sync. Wait for it and dismiss it (as a
 * user would) so it isn't mid-open during the click under test — the open
 * animation races the click and swallows it.
 *
 * The prompt is optional UI: it only appears once the async push-state sync
 * settles, which can be slower (or not at all) in some environments (e.g.
 * headed runs). If it never appears, proceed — tests must not depend on it.
 */
export async function dismissPushPrompt(page: Page) {
	const dismiss = page.getByRole('button', { name: 'Dismiss notification prompt' });
	try {
		await dismiss.waitFor({ state: 'visible', timeout: 5000 });
		await dismiss.click();
	} catch {
		// Prompt never appeared; nothing to dismiss.
	}
}
