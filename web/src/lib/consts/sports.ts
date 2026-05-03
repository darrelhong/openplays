export type SelectItem = {
	value: string;
	label: string;
	disabled?: boolean;
};

export const SPORTS: SelectItem[] = [
	{ value: 'badminton', label: 'Badminton' },
	{ value: 'tennis', label: 'Tennis' },
	{ value: 'football', label: 'Football' },
	{ value: 'pickleball', label: 'Pickleball' }
];

export const GAME_TYPES: SelectItem[] = [
	{ value: 'doubles', label: 'Doubles' },
	{ value: 'singles', label: 'Singles' },
	{ value: 'mixed_doubles', label: 'Mixed Doubles' }
];

export const DURATIONS: SelectItem[] = [
	{ value: '15', label: '15 min' },
	{ value: '30', label: '30 min' },
	{ value: '45', label: '45 min' },
	{ value: '60', label: '1 hour' },
	{ value: '75', label: '1h 15m' },
	{ value: '90', label: '1.5 hours' },
	{ value: '105', label: '1h 45m' },
	{ value: '120', label: '2 hours' },
	{ value: '135', label: '2h 15m' },
	{ value: '150', label: '2.5 hours' },
	{ value: '165', label: '2h 45m' },
	{ value: '180', label: '3 hours' },
	{ value: '210', label: '3.5 hours' },
	{ value: '240', label: '4 hours' },
	{ value: '270', label: '4.5 hours' },
	{ value: '300', label: '5 hours' }
];
