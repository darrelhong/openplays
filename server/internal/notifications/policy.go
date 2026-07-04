package notifications

type DeliveryPolicy struct {
	Feed         bool
	Push         bool
	DebounceFeed bool
}

var deliveryPoliciesByKind = map[string]DeliveryPolicy{
	"play.join_requested":    {Feed: true, Push: true},
	"play.player_added":      {Feed: true, Push: true},
	"play.moved_to_waitlist": {Feed: true, Push: true},
	"play.player_joined":     {Feed: true, Push: true},
	"play.player_confirmed":  {Feed: true, Push: true},
	"play.player_left":       {Feed: true, Push: true},
	"chat.message":           {Feed: true, Push: true, DebounceFeed: true},
}

var defaultDeliveryPolicy = DeliveryPolicy{Feed: true, Push: true}

func deliveryPolicyForKind(kind string) DeliveryPolicy {
	if policy, ok := deliveryPoliciesByKind[kind]; ok {
		return policy
	}
	return defaultDeliveryPolicy
}
