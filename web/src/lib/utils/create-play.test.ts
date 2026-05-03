import { describe, it, expect } from 'vitest';
import { feeToCents, buildRFC3339, optionalNumber } from './create-play';

describe('feeToCents', () => {
	it('converts dollars to cents', () => {
		expect(feeToCents('10')).toBe(1000);
		expect(feeToCents('5.50')).toBe(550);
		expect(feeToCents('0.99')).toBe(99);
	});

	it('rounds to nearest cent', () => {
		expect(feeToCents('10.555')).toBe(1056);
		expect(feeToCents('10.554')).toBe(1055);
	});

	it('returns undefined for empty/null', () => {
		expect(feeToCents(null)).toBeUndefined();
		expect(feeToCents('')).toBeUndefined();
	});

	it('returns undefined for non-numeric', () => {
		expect(feeToCents('abc')).toBeUndefined();
	});

	it('handles zero', () => {
		expect(feeToCents('0')).toBe(0);
	});
});

describe('buildRFC3339', () => {
	it('combines date + time + offset into RFC3339', () => {
		expect(buildRFC3339('2026-05-02', '10:00', '+08:00')).toBe('2026-05-02T10:00:00+08:00');
	});

	it('works with different timezones', () => {
		expect(buildRFC3339('2026-01-15', '09:30', '+00:00')).toBe('2026-01-15T09:30:00+00:00');
		expect(buildRFC3339('2026-12-31', '23:59', '-05:00')).toBe('2026-12-31T23:59:00-05:00');
	});

	it('returns empty string if date missing', () => {
		expect(buildRFC3339('', '10:00', '+08:00')).toBe('');
	});

	it('returns empty string if time missing', () => {
		expect(buildRFC3339('2026-05-02', '', '+08:00')).toBe('');
	});
});

describe('optionalNumber', () => {
	it('converts string to number', () => {
		expect(optionalNumber('5')).toBe(5);
		expect(optionalNumber('120')).toBe(120);
	});

	it('returns undefined for empty/null', () => {
		expect(optionalNumber(null)).toBeUndefined();
		expect(optionalNumber('')).toBeUndefined();
	});

	it('returns undefined for non-numeric', () => {
		expect(optionalNumber('abc')).toBeUndefined();
	});

	it('handles decimals', () => {
		expect(optionalNumber('3.5')).toBe(3.5);
	});
});
