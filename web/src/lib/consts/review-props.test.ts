import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { REVIEW_PROP_LABELS, reviewPropLabel } from './review-props';

// The prop vocabulary is owned by the backend; this keeps the label map in
// lockstep with it by extracting every quoted slug from the Go source.
function backendPropSlugs(): string[] {
	const source = readFileSync(
		resolve(__dirname, '../../../../server/internal/reviews/props.go'),
		'utf-8'
	);
	return [...source.matchAll(/^\t{2,}"([a-z_]+)",$/gm)].map((match) => match[1]!);
}

describe('review prop labels', () => {
	it('covers every backend prop slug', () => {
		const slugs = backendPropSlugs();
		expect(slugs.length).toBeGreaterThan(20);
		const missing = [...new Set(slugs)].filter((slug) => !(slug in REVIEW_PROP_LABELS));
		expect(missing).toEqual([]);
	});

	it('has no labels for slugs the backend dropped', () => {
		const slugs = new Set(backendPropSlugs());
		const stale = Object.keys(REVIEW_PROP_LABELS).filter((slug) => !slugs.has(slug));
		expect(stale).toEqual([]);
	});

	it('falls back to a readable label for unknown slugs', () => {
		expect(reviewPropLabel('great_sport')).toBe('Great sport');
		expect(reviewPropLabel('brand_new_prop')).toBe('brand new prop');
	});
});
