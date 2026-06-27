import { afterEach, describe, expect, it, vi } from 'vitest';
import { formatNotificationTime } from './notification-time';

describe('formatNotificationTime', () => {
	afterEach(() => {
		vi.useRealTimers();
	});

	it('formats recent notification times as compact relative labels', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-06-27T12:00:00Z'));

		expect(formatNotificationTime('2026-06-27T11:59:40Z')).toBe('now');
		expect(formatNotificationTime('2026-06-27T11:55:00Z')).toBe('5m');
		expect(formatNotificationTime('2026-06-27T09:00:00Z')).toBe('3h');
		expect(formatNotificationTime('2026-06-24T12:00:00Z')).toBe('3d');
	});

	it('falls back to a date label for older notifications', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-06-27T12:00:00Z'));

		expect(formatNotificationTime('2026-05-01T12:00:00Z')).toBe('1 May');
	});

	it('returns an empty label for invalid timestamps', () => {
		expect(formatNotificationTime('not-a-date')).toBe('');
	});
});
