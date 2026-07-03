import { describe, expect, it } from 'vitest';
import { formatMessage } from './message-format';

describe('formatMessage', () => {
	it('returns plain text as a single segment', () => {
		expect(formatMessage('see you at the court')).toEqual([
			{ type: 'text', value: 'see you at the court' }
		]);
	});

	it('extracts an http(s) url with surrounding text', () => {
		expect(formatMessage('map here https://maps.app/abc123 see you')).toEqual([
			{ type: 'text', value: 'map here ' },
			{ type: 'link', value: 'https://maps.app/abc123', href: 'https://maps.app/abc123' },
			{ type: 'text', value: ' see you' }
		]);
	});

	it('trims trailing sentence punctuation from urls', () => {
		expect(formatMessage('book at https://example.com/court.')).toEqual([
			{ type: 'text', value: 'book at ' },
			{ type: 'link', value: 'https://example.com/court', href: 'https://example.com/court' },
			{ type: 'text', value: '.' }
		]);
	});

	it('handles multiple urls', () => {
		const segments = formatMessage('http://a.com and https://b.com');
		expect(segments.filter((segment) => segment.type === 'link')).toHaveLength(2);
	});

	it('handles a body that is only a url', () => {
		expect(formatMessage('https://example.com')).toEqual([
			{ type: 'link', value: 'https://example.com', href: 'https://example.com' }
		]);
	});

	it('ignores non-http schemes', () => {
		expect(formatMessage('run javascript:alert(1) or ftp://files')).toEqual([
			{ type: 'text', value: 'run javascript:alert(1) or ftp://files' }
		]);
	});
});
