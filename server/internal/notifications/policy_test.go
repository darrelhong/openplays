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
