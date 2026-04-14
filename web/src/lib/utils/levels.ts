// Level definitions per sport.
// Ordered by skill — index position determines ordering for range validation.
// Will be sport-dependent when multi-sport is added.

export type LevelItem = {
	value: string;
	label: string;
};

export const badmintonLevels: LevelItem[] = [
	{ value: 'LB', label: 'LB - Low Beginner' },
	{ value: 'MB', label: 'MB - Mid Beginner' },
	{ value: 'HB', label: 'HB - High Beginner' },
	{ value: 'LI', label: 'LI - Low Intermediate' },
	{ value: 'MI', label: 'MI - Mid Intermediate' },
	{ value: 'HI', label: 'HI - High Intermediate' },
	{ value: 'A', label: 'A - Advanced' }
];

/**
 * Returns the index of a level value in the given levels array.
 * Returns -1 if not found or value is empty.
 */
export function levelIndex(levels: LevelItem[], value: string): number {
	if (!value) return -1;
	return levels.findIndex((l) => l.value === value);
}
