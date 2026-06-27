package notifications

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	push "openplays/server/internal/notifications"
)

type VAPIDPublicKeyOutput struct {
	Body struct {
		PublicKey string `json:"public_key"`
	}
}

type SubscribeInput struct {
	Body push.PushSubscription
}

type ListInput struct {
	Limit int `query:"limit" minimum:"1" maximum:"50" default:"50" doc:"Maximum notifications to return"`
}

type ListOutput struct {
	Body struct {
		Notifications []push.UserNotification `json:"notifications"`
	}
}

type MarkReadInput struct {
	Body struct {
		IDs []string `json:"ids,omitempty" doc:"Notification IDs to mark read. Empty marks all notifications read."`
	}
}

func Register(api huma.API, service *push.WebPushService, authMiddleware func(huma.Context, func(huma.Context))) {
	grp := huma.NewGroup(api, "/notifications")

	huma.Register(grp, huma.Operation{
		OperationID: "list-notifications",
		Summary:     "List notifications",
		Description: "Returns the current user's latest notifications.",
		Method:      http.MethodGet,
		Path:        "/",
		Tags:        []string{"Notifications"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *ListInput) (*ListOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		notifications, err := service.ListNotifications(ctx, user.ID, limit)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list notifications")
		}
		out := &ListOutput{}
		out.Body.Notifications = notifications
		return out, nil
	})

	huma.Register(grp, huma.Operation{
		OperationID: "mark-notifications-read",
		Summary:     "Mark notifications read",
		Description: "Marks notifications as read for the current user.",
		Method:      http.MethodPost,
		Path:        "/read",
		Tags:        []string{"Notifications"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *MarkReadInput) (*struct{}, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if err := service.MarkNotificationsRead(ctx, user.ID, input.Body.IDs); err != nil {
			return nil, huma.Error500InternalServerError("failed to mark notifications read")
		}
		return &struct{}{}, nil
	})

	huma.Register(grp, huma.Operation{
		OperationID: "get-web-push-vapid-public-key",
		Summary:     "Get Web Push public key",
		Description: "Returns the VAPID public key for Web Push subscriptions.",
		Method:      http.MethodGet,
		Path:        "/push/vapid-public-key",
		Tags:        []string{"Notifications"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, _ *struct{}) (*VAPIDPublicKeyOutput, error) {
		if authmw.UserFromContext(ctx) == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		out := &VAPIDPublicKeyOutput{}
		out.Body.PublicKey = service.PublicKey()
		return out, nil
	})

	huma.Register(grp, huma.Operation{
		OperationID: "subscribe-web-push",
		Summary:     "Subscribe to Web Push",
		Description: "Stores a Web Push subscription for the current user.",
		Method:      http.MethodPost,
		Path:        "/push/subscriptions",
		Tags:        []string{"Notifications"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *SubscribeInput) (*struct{}, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if err := service.Subscribe(ctx, user.ID, input.Body); err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return &struct{}{}, nil
	})
}
