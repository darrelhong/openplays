import type { Page } from '@playwright/test';

/**
 * The global "Enable push notifications" prompt (rendered via the layout)
 * auto-opens after an async push-state sync. Wait for it and dismiss it (as a
 * user would) so it isn't mid-open during the click under test — the open
 * animation races the click and swallows it.
 */
export async function dismissPushPrompt(page: Page) {
	await page.getByRole('button', { name: 'Dismiss notification prompt' }).click();
}
