package notifications

type DeliveryPolicy struct {
	Feed         bool
	Push         bool
	DebounceFeed bool
}

var deliveryPoliciesByKind = map[string]DeliveryPolicy{
	"play.join_requested":    {Feed: true, Push: true},
	"play.player_added":      {Feed: true, Push: true},
	"play.invited":           {Feed: true, Push: true},
	"play.invite_accepted":   {Feed: true, Push: true},
	"play.invite_declined":   {Feed: true, Push: true},
	"play.moved_to_waitlist": {Feed: true, Push: true},
	"play.player_joined":     {Feed: true, Push: true},
	"play.player_confirmed":  {Feed: true, Push: true},
	"play.player_left":       {Feed: true, Push: true},
	"play.review_prompt":     {Feed: true, Push: true},
	"play.cancelled":         {Feed: true, Push: true},
	"chat.message":           {Feed: true, Push: true, DebounceFeed: true},
	"follow.started":         {Feed: true, Push: true},
	"follow.requested":       {Feed: true, Push: true},
	"follow.accepted":        {Feed: true, Push: true},
	"follow.play_hosted":     {Feed: true, Push: true},
	"follow.play_joined":     {Feed: true, Push: true},
}

var defaultDeliveryPolicy = DeliveryPolicy{Feed: true, Push: true}

func deliveryPolicyForKind(kind string) DeliveryPolicy {
	if policy, ok := deliveryPoliciesByKind[kind]; ok {
		return policy
	}
	return defaultDeliveryPolicy
}
