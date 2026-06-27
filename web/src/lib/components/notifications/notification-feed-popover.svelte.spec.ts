import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import NotificationFeedPopover from './notification-feed-popover.svelte';

describe('NotificationFeedPopover', () => {
	it('opens the feed and renders unread notification rows', async () => {
		render(NotificationFeedPopover, {
			open: false,
			notifications: [
				{
					id: 'notification-1',
					title: 'Friday Friendly',
					body: 'Seed Advanced joined the game',
					play_id: 'play-1',
					created_at: new Date().toISOString()
				}
			],
			openUnreadIds: new Set<string>(),
			unreadCount: 1,
			canEnablePush: false,
			enablingPush: false,
			onEnablePush: vi.fn()
		});

		await page.getByLabelText('1 unread notification').click();

		await expect.element(page.getByRole('heading', { name: 'Notifications' })).toBeVisible();
		await expect.element(page.getByText('Friday Friendly')).toBeVisible();
		await expect.element(page.getByText('Seed Advanced joined the game')).toBeVisible();
		await expect.element(page.getByRole('link', { name: /Friday Friendly/ })).toBeVisible();
	});

	it('shows the enable button when push can be enabled', async () => {
		const onEnablePush = vi.fn();
		render(NotificationFeedPopover, {
			open: false,
			notifications: [],
			openUnreadIds: new Set<string>(),
			unreadCount: 0,
			canEnablePush: true,
			enablingPush: false,
			onEnablePush
		});

		await page.getByLabelText('Notifications').click();
		await page.getByRole('button', { name: 'Enable' }).click();

		expect(onEnablePush).toHaveBeenCalledOnce();
	});
});
