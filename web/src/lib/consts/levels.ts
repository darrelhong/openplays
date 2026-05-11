import type { SelectItem } from './sports';

// Level definitions per sport.
// Ordered by skill — index position determines ordering for range validation.

export const BADMINTON_LEVELS: SelectItem[] = [
	{ value: 'LB', label: 'LB - Low Beginner' },
	{ value: 'MB', label: 'MB - Mid Beginner' },
	{ value: 'HB', label: 'HB - High Beginner' },
	{ value: 'LI', label: 'LI - Low Intermediate' },
	{ value: 'MI', label: 'MI - Mid Intermediate' },
	{ value: 'HI', label: 'HI - High Intermediate' },
	{ value: 'A', label: 'A - Advanced' }
];

export const TENNIS_LEVELS: SelectItem[] = [
	{ value: '1.0', label: '1.0' },
	{ value: '1.5', label: '1.5' },
	{ value: '2.0', label: '2.0' },
	{ value: '2.5', label: '2.5' },
	{ value: '3.0', label: '3.0' },
	{ value: '3.5', label: '3.5' },
	{ value: '4.0', label: '4.0' },
	{ value: '4.5', label: '4.5' },
	{ value: '5.0', label: '5.0' },
	{ value: '5.5', label: '5.5' },
	{ value: '6.0', label: '6.0' },
	{ value: '6.5', label: '6.5' },
	{ value: '7.0', label: '7.0' }
];

/**
 * Returns the index of a level value in the given levels array.
 * Returns -1 if not found or value is empty.
 */
export function levelIndex(levels: SelectItem[], value: string): number {
	if (!value) return -1;
	return levels.findIndex((l) => l.value === value);
}
