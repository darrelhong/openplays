import { afterEach, describe, expect, it, vi } from 'vitest';
import { formatDistance } from './format-distance';

describe('formatDistance', () => {
	afterEach(() => {
		vi.useRealTimers();
	});

	it('formats recent notification times as compact relative labels', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-06-27T12:00:00Z'));

		expect(formatDistance('2026-06-27T11:59:40Z')).toBe('now');
		expect(formatDistance('2026-06-27T11:55:00Z')).toBe('5m');
		expect(formatDistance('2026-06-27T09:00:00Z')).toBe('3h');
		expect(formatDistance('2026-06-24T12:00:00Z')).toBe('3d');
	});

	it('appends ago only to relative labels when asked', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-06-27T12:00:00Z'));

		expect(formatDistance('2026-06-27T11:55:00Z', { suffix: true })).toBe('5m ago');
		expect(formatDistance('2026-06-24T12:00:00Z', { suffix: true })).toBe('3d ago');
		// "now ago" and "1 May ago" would be nonsense
		expect(formatDistance('2026-06-27T11:59:40Z', { suffix: true })).toBe('now');
		expect(formatDistance('2026-05-01T12:00:00Z', { suffix: true })).toBe('1 May');
	});

	it('falls back to a date label for older notifications', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-06-27T12:00:00Z'));

		expect(formatDistance('2026-05-01T12:00:00Z')).toBe('1 May');
	});

	it('returns an empty label for invalid timestamps', () => {
		expect(formatDistance('not-a-date')).toBe('');
	});
});
