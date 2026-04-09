import { describe, it, expect } from 'vitest';
import {
	capitalize,
	formatDate,
	formatTime,
	formatFee,
	getNumericFee,
	getMetaFee,
	formatPlayFee,
	formatLevel
} from './formatting';

describe('capitalize', () => {
	it('capitalizes first letter', () => {
		expect(capitalize('badminton')).toBe('Badminton');
	});

	it('handles single character', () => {
		expect(capitalize('a')).toBe('A');
	});

	it('handles empty string', () => {
		expect(capitalize('')).toBe('');
	});

	it('preserves rest of string', () => {
		expect(capitalize('mixed_doubles')).toBe('Mixed_doubles');
	});
});

describe('formatDate', () => {
	it('formats a date with timezone', () => {
		const result = formatDate('2026-04-08T09:00:00Z', 'Asia/Singapore');
		expect(result).toContain('Apr');
		expect(result).toContain('8');
	});

	it('respects timezone for date boundary', () => {
		// Midnight UTC = 8am SGT, still same date
		const result = formatDate('2026-04-08T00:00:00Z', 'Asia/Singapore');
		expect(result).toContain('8');
	});
});

describe('formatTime', () => {
	it('formats time with timezone', () => {
		const result = formatTime('2026-04-08T09:00:00Z', 'Asia/Singapore');
		// 9am UTC = 5pm SGT
		expect(result).toContain('5');
	});

	it('omits minutes when on the hour', () => {
		const result = formatTime('2026-04-08T09:00:00Z', 'Asia/Singapore');
		expect(result).not.toContain(':00');
	});

	it('includes minutes when not on the hour', () => {
		const result = formatTime('2026-04-08T09:30:00Z', 'Asia/Singapore');
		expect(result).toContain('30');
	});
});

describe('formatFee', () => {
	it('formats whole dollar amount without decimals', () => {
		const result = formatFee(1000, 'SGD');
		expect(result).toBe('$10');
	});

	it('formats with two decimals when there are cents', () => {
		const result = formatFee(1050, 'SGD');
		expect(result).toBe('$10.50');
	});

	it('formats with two decimals for arbitrary cents', () => {
		const result = formatFee(1055, 'SGD');
		expect(result).toBe('$10.55');
	});

	it('formats whole dollars without decimals', () => {
		const result = formatFee(1500, 'SGD');
		expect(result).toBe('$15');
	});

	it('formats with one decimal when tens place is zero', () => {
		const result = formatFee(1500, 'SGD');
		expect(result).toBe('$15');
	});

	it('handles small amounts', () => {
		const result = formatFee(50, 'SGD');
		expect(result).toBe('$0.50');
	});

	it('handles zero', () => {
		const result = formatFee(0, 'SGD');
		expect(result).toBe('$0');
	});
});

describe('getNumericFee', () => {
	it('returns number for finite numbers', () => {
		expect(getNumericFee(1000)).toBe(1000);
	});

	it('returns null for undefined', () => {
		expect(getNumericFee(undefined)).toBeNull();
	});

	it('returns null for null', () => {
		expect(getNumericFee(null)).toBeNull();
	});

	it('returns null for strings', () => {
		expect(getNumericFee('1000')).toBeNull();
	});

	it('returns null for NaN', () => {
		expect(getNumericFee(NaN)).toBeNull();
	});

	it('returns null for Infinity', () => {
		expect(getNumericFee(Infinity)).toBeNull();
	});
});

describe('getMetaFee', () => {
	it('extracts fee_male from meta', () => {
		expect(getMetaFee({ fee_male: 1500 }, 'fee_male')).toBe(1500);
	});

	it('extracts fee_female from meta', () => {
		expect(getMetaFee({ fee_female: 1200 }, 'fee_female')).toBe(1200);
	});

	it('returns null for missing key', () => {
		expect(getMetaFee({}, 'fee_male')).toBeNull();
	});

	it('returns null for null meta', () => {
		expect(getMetaFee(null, 'fee_male')).toBeNull();
	});

	it('returns null for undefined meta', () => {
		expect(getMetaFee(undefined, 'fee_male')).toBeNull();
	});

	it('returns null for non-numeric values', () => {
		expect(getMetaFee({ fee_male: 'free' }, 'fee_male')).toBeNull();
	});
});

describe('formatPlayFee', () => {
	it('uses primary fee when available', () => {
		const result = formatPlayFee({ fee: 1000, currency: 'SGD', meta: {} });
		expect(result).toBe('$10');
	});

	it('falls back to gendered fees from meta', () => {
		const result = formatPlayFee({
			fee: undefined,
			currency: 'SGD',
			meta: { fee_male: 1500, fee_female: 1200 }
		});
		expect(result).toBe('$15 (M), $12 (F)');
	});

	it('shows only male fee when female is missing', () => {
		const result = formatPlayFee({
			fee: undefined,
			currency: 'SGD',
			meta: { fee_male: 1500 }
		});
		expect(result).toBe('$15 (M)');
	});

	it('shows only female fee when male is missing', () => {
		const result = formatPlayFee({
			fee: undefined,
			currency: 'SGD',
			meta: { fee_female: 1200 }
		});
		expect(result).toBe('$12 (F)');
	});

	it('returns dash when no fee info available', () => {
		const result = formatPlayFee({ fee: undefined, currency: 'SGD', meta: {} });
		expect(result).toBe('-');
	});

	it('prefers primary fee over meta fees', () => {
		const result = formatPlayFee({
			fee: 800,
			currency: 'SGD',
			meta: { fee_male: 1500, fee_female: 1200 }
		});
		expect(result).toBe('$8');
	});
});

describe('formatLevel', () => {
	it('formats range with min and max', () => {
		expect(formatLevel('B+', 'A')).toBe('B+ - A');
	});

	it('formats min only with plus suffix', () => {
		expect(formatLevel('B+', undefined)).toBe('B++');
	});

	it('formats max only with dash prefix', () => {
		expect(formatLevel(undefined, 'A')).toBe('- A');
	});

	it('returns dash when both are undefined', () => {
		expect(formatLevel(undefined, undefined)).toBe('-');
	});

	it('returns dash when both are empty strings', () => {
		expect(formatLevel('', '')).toBe('-');
	});
});
