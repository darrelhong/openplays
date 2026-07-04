type PlayCapacitySource = {
	confirmed_count?: number;
	added_count?: number;
	confirmed_participants?: { id: number }[] | null;
	added_participants?: { id: number }[] | null;
	participant_preview?: { id: number }[] | null;
};

/**
 * The lowest max_players a host may set when editing: confirmed and added
 * players both hold reserved slots, mirroring the server's
 * CountReservedPlayParticipants guard (409 below the reserved count).
 */
export function minEditableMaxPlayers(play: PlayCapacitySource): number {
	const confirmed =
		play.confirmed_count ??
		play.confirmed_participants?.length ??
		play.participant_preview?.length ??
		0;
	const added = play.added_count ?? play.added_participants?.length ?? 0;
	return Math.max(confirmed + added, 1);
}
