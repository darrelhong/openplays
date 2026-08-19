package notifications

import "testing"

func TestDeliveryPolicyForPlayKinds(t *testing.T) {
	tests := []struct {
		name string
		kind string
		want DeliveryPolicy
	}{
		{
			name: "join requested stores feed and push",
			kind: "play.join_requested",
			want: DeliveryPolicy{Feed: true, Push: true},
		},
		{
			name: "player added stores feed and push",
			kind: "play.player_added",
			want: DeliveryPolicy{Feed: true, Push: true},
		},
		{
			name: "play cancelled stores feed and push",
			kind: "play.cancelled",
			want: DeliveryPolicy{Feed: true, Push: true},
		},
		{
			name: "unknown kinds default to feed and push",
			kind: "custom.event",
			want: DeliveryPolicy{Feed: true, Push: true},
		},
		{
			name: "chat message stores debounced feed and push",
			kind: "chat.message",
			want: DeliveryPolicy{Feed: true, Push: true, DebounceFeed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deliveryPolicyForKind(tt.kind); got != tt.want {
				t.Fatalf("deliveryPolicyForKind(%q) = %+v, want %+v", tt.kind, got, tt.want)
			}
		})
	}
}

func TestFollowDeliveryPoliciesAreExplicit(t *testing.T) {
	for _, kind := range []string{
		"follow.started",
		"follow.requested",
		"follow.accepted",
		"follow.play_hosted",
		"follow.play_joined",
	} {
		t.Run(kind, func(t *testing.T) {
			policy, ok := deliveryPoliciesByKind[kind]
			if !ok {
				t.Fatalf("delivery policy for %q is not explicitly registered", kind)
			}
			if policy != (DeliveryPolicy{Feed: true, Push: true}) {
				t.Fatalf("delivery policy for %q = %+v, want feed and push", kind, policy)
			}
		})
	}
}
