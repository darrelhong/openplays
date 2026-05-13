type PlayRosterSource = {
	listing_type: string;
	created_by?: string;
	max_players?: number;
	slots_left?: number;
	host_name: string;
	creator_display_name?: string;
	creator_photo_url?: string;
};

export type RosterPreviewSlot =
	| {
			kind: 'known';
			name: string;
			photoUrl?: string;
	  }
	| {
			kind: 'occupied';
			label: string;
	  }
	| {
			kind: 'open';
			label: string;
	  };

export type RosterPreview = {
	slots: RosterPreviewSlot[];
	totalSlots: number;
	occupiedSlots: number;
	openSlots: number;
	hiddenSlots: number;
	label: string;
};

export function getPlayRosterPreview(
	play: PlayRosterSource,
	maxVisibleSlots = 8
): RosterPreview | null {
	const maxPlayers = normalizeSlotCount(play.max_players);

	if (play.listing_type !== 'play' || !play.created_by || maxPlayers === null) {
		return null;
	}

	const requestedVisibleSlots = Math.max(1, Math.floor(maxVisibleSlots));
	const openSlots = clampSlotCount(play.slots_left ?? maxPlayers - 1, 0, maxPlayers - 1);
	const occupiedSlots = maxPlayers - openSlots;
	const allSlots: RosterPreviewSlot[] = [
		{
			kind: 'known',
			name: play.creator_display_name ?? play.host_name,
			photoUrl: play.creator_photo_url
		}
	];

	for (let index = 1; index < occupiedSlots; index += 1) {
		allSlots.push({
			kind: 'occupied',
			label: `Confirmed participant ${index + 1}`
		});
	}

	for (let index = 0; index < openSlots; index += 1) {
		allSlots.push({
			kind: 'open',
			label: `Open slot ${index + 1}`
		});
	}

	return {
		slots: allSlots.slice(0, requestedVisibleSlots),
		totalSlots: maxPlayers,
		occupiedSlots,
		openSlots,
		hiddenSlots: Math.max(0, allSlots.length - requestedVisibleSlots),
		label: `${occupiedSlots}/${maxPlayers} joined`
	};
}

function normalizeSlotCount(value: number | undefined): number | null {
	if (value === undefined || !Number.isInteger(value) || value < 1) {
		return null;
	}
	return value;
}

function clampSlotCount(value: number, min: number, max: number): number {
	if (!Number.isFinite(value)) {
		return min;
	}
	return Math.min(max, Math.max(min, Math.floor(value)));
}
