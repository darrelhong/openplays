// Display labels for review prop slugs. The slugs and their availability
// rules (universal / per-sport / host-only) live in the backend:
// server/internal/reviews/props.go
export const REVIEW_PROP_LABELS: Record<string, string> = {
	// Universal
	great_sport: 'Great sport',
	chill_vibes: 'Chill vibes',
	humble: 'Humble',
	punctual: 'Punctual',
	// Badminton
	powerful_smash: 'Powerful smash',
	great_net_play: 'Great at the net',
	solid_defense: 'Solid defense',
	deceptive_shots: 'Deceptive shots',
	fast_footwork: 'Fast footwork',
	tricky_serve: 'Tricky serve',
	delicate_drops: 'Delicate drop shots',
	great_doubles_partner: 'Great doubles partner',
	// Tennis
	big_forehand: 'Big forehand',
	strong_backhand: 'Strong backhand',
	consistent_rallies: 'Consistent rallies',
	big_serve: 'Big serve',
	great_returns: 'Great returns',
	sharp_volleys: 'Sharp volleys',
	heavy_topspin: 'Heavy topspin',
	precise_placement: 'Pinpoint placement',
	ice_under_pressure: 'Ice cold under pressure',
	// Pickleball
	dink_master: 'Dink master',
	overhead_slams: 'Overhead slams',
	big_drives: 'Big drives',
	soft_hands: 'Soft hands',
	smart_lobs: 'Smart lobs',
	quick_reflexes: 'Quick reflexes',
	consistent_serves: 'Consistent serves',
	// Football
	clinical_finisher: 'Clinical finisher',
	great_touch: 'Great touch',
	great_passing: 'Great passing',
	solid_defending: 'Solid defending',
	tireless_runner: 'Tireless runner',
	creative_playmaker: 'Creative playmaker',
	strong_tackles: 'Strong tackles',
	safe_hands: 'Safe hands',
	reads_the_game: 'Reads the game',
	// Host
	well_organized: 'Well organized',
	quick_replies: 'Quick replies',
	clear_communication: 'Clear communication'
};

/** The reviewer picks at most this many props per co-player (mirrors the backend cap). */
export const MAX_PROPS_PER_REVIEW = 2;

export function reviewPropLabel(slug: string): string {
	return REVIEW_PROP_LABELS[slug] ?? slug.replaceAll('_', ' ');
}
