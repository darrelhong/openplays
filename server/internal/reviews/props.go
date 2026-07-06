package reviews

import (
	"fmt"
	"slices"

	"openplays/server/internal/model"
)

// Prop slugs are stable identifiers stored in play_reviews.props; display
// labels live in the frontend consts (web/src/lib/consts/review-props.ts).
// Every given prop is linked to the play's sport (ratings and shoutouts are
// universal): universal props count toward the sport they were earned in,
// and sport props are only valid on plays of that sport.
var (
	// UniversalPeerProps can be given to any reviewee in any sport. They are
	// about attitude and reliability; skill praise is sport-specific.
	UniversalPeerProps = []string{
		"great_sport",
		"chill_vibes",
		"humble",
		"punctual",
	}

	// SportPeerProps are the skill props per sport.
	SportPeerProps = map[model.Sport][]string{
		model.SportBadminton: {
			"powerful_smash",
			"great_net_play",
			"solid_defense",
			"deceptive_shots",
			"fast_footwork",
			"tricky_serve",
			"delicate_drops",
			"great_doubles_partner",
		},
		model.SportTennis: {
			"big_forehand",
			"strong_backhand",
			"consistent_rallies",
			"big_serve",
			"great_returns",
			"sharp_volleys",
			"heavy_topspin",
			"precise_placement",
			"fast_footwork",
			"great_doubles_partner",
			"ice_under_pressure",
		},
		model.SportPickleball: {
			"dink_master",
			"sharp_volleys",
			"overhead_slams",
			"big_drives",
			"soft_hands",
			"smart_lobs",
			"quick_reflexes",
			"fast_footwork",
			"great_doubles_partner",
			"consistent_serves",
		},
		model.SportFootball: {
			"clinical_finisher",
			"great_touch",
			"great_passing",
			"solid_defending",
			"tireless_runner",
			"creative_playmaker",
			"strong_tackles",
			"safe_hands",
			"reads_the_game",
		},
	}

	// HostProps can additionally be given when the reviewee hosted the play.
	HostProps = []string{
		"well_organized",
		"quick_replies",
		"clear_communication",
	}
)

// MaxPropsPerReview caps how many props one review can carry: picking a
// couple keeps them meaningful.
const MaxPropsPerReview = 2

// PeerPropsFor lists the props available for any reviewee on a play of the
// given sport: the universal set plus the sport's skill props.
func PeerPropsFor(sport model.Sport) []string {
	return append(slices.Clone(UniversalPeerProps), SportPeerProps[sport]...)
}

// ValidateProps dedupes the given prop slugs (preserving order) and rejects
// slugs that are unknown, from another sport, or host-only for a reviewee
// who did not host the play, and more than MaxPropsPerReview of them.
func ValidateProps(props []string, sport model.Sport, revieweeIsHost bool) ([]string, error) {
	peerProps := PeerPropsFor(sport)
	out := make([]string, 0, len(props))
	for _, prop := range props {
		if slices.Contains(out, prop) {
			continue
		}
		switch {
		case slices.Contains(peerProps, prop):
		case slices.Contains(HostProps, prop):
			if !revieweeIsHost {
				return nil, fmt.Errorf("prop %q is only for hosts", prop)
			}
		default:
			return nil, fmt.Errorf("unknown prop %q for %s", prop, sport)
		}
		out = append(out, prop)
	}
	if len(out) > MaxPropsPerReview {
		return nil, fmt.Errorf("at most %d props per review", MaxPropsPerReview)
	}
	return out, nil
}
