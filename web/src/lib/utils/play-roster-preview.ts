type PlayRosterSource = {
	listing_type: string;
	created_by?: string;
	max_players?: number;
	slots_left?: number;
	host_name: string;
	creator_display_name?: string;
	creator_photo_url?: string;
	participant_preview?: PlayRosterParticipantPreview[] | null;
};

type PlayRosterParticipantPreview = {
	id: number;
	display_name?: string;
	photo_url?: string;
	rating_code?: string;
	is_guest: boolean;
};

export type RosterPreviewSlot =
	| {
			kind: 'known';
			name: string;
			photoUrl?: string;
			ratingCode?: string;
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
	const participantSlots = getParticipantSlots(play);
	const hasParticipantPreview = participantSlots.length > 0;
	const allSlots: RosterPreviewSlot[] = hasParticipantPreview
		? participantSlots.slice(0, maxPlayers)
		: [
				{
					kind: 'known',
					name: play.creator_display_name ?? play.host_name,
					photoUrl: play.creator_photo_url
				}
			];

	// slots_left is the server's slot accounting (confirmed and added players
	// both reserve slots); prefer it over inferring from the preview, which
	// can omit reserved players
	const inferredOpenSlots = maxPlayers - allSlots.length;
	const openSlots = clampSlotCount(play.slots_left ?? inferredOpenSlots, 0, inferredOpenSlots);
	const occupiedSlots = maxPlayers - openSlots;

	for (let index = allSlots.length; index < occupiedSlots; index += 1) {
		allSlots.push({
			kind: 'occupied',
			label: `Reserved spot ${index + 1}`
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

function getParticipantSlots(play: PlayRosterSource): RosterPreviewSlot[] {
	if (!Array.isArray(play.participant_preview)) {
		return [];
	}

	return play.participant_preview.map<RosterPreviewSlot>((participant) => ({
		kind: 'known',
		name: participant.display_name ?? 'Player',
		photoUrl: participant.photo_url,
		ratingCode: participant.rating_code
	}));
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
