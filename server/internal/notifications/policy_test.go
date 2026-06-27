package notifications

import "testing"

func TestDeliveryPolicyForPlayKinds(t *testing.T) {
	tests := []struct {
		name string
		kind string
		want DeliveryPolicy
	}{
		{
			name: "waitlist joined stores feed and push",
			kind: "play.waitlist_joined",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deliveryPolicyForKind(tt.kind); got != tt.want {
				t.Fatalf("deliveryPolicyForKind(%q) = %+v, want %+v", tt.kind, got, tt.want)
			}
		})
	}
}
