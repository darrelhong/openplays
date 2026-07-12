package me

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/auth"
	"openplays/server/internal/avatar"
	"openplays/server/internal/db"
)

const maxAvatarMultipartBytes = avatar.MaxInputBytes + 64<<10

type AvatarService interface {
	Upload(context.Context, string, io.Reader) (db.User, error)
	Remove(context.Context, string) (db.User, error)
}

type avatarUploadInput struct {
	RawBody huma.MultipartFormFiles[struct {
		Avatar huma.FormFile `form:"avatar" contentType:"image/jpeg,image/png" required:"true"`
	}]
}

type avatarOutput struct {
	Body auth.User
}

func RegisterAvatar(api huma.API, service AvatarService) {
	uploadOperation := huma.Operation{
		OperationID: "upload-my-avatar",
		Summary:     "Upload profile photo",
		Method:      http.MethodPut,
		Path:        "/avatar",
		Tags:        []string{"Me"},
		Middlewares: huma.Middlewares{prepareAvatarUpload(api, service)},
	}
	huma.Register(api, uploadOperation, func(ctx context.Context, input *avatarUploadInput) (*avatarOutput, error) {
		if service == nil {
			return nil, huma.Error503ServiceUnavailable("avatar uploads are not configured")
		}
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		file := input.RawBody.Data().Avatar
		if file.Size > avatar.MaxInputBytes {
			return nil, huma.Error413RequestEntityTooLarge(fmt.Sprintf(
				"profile photo exceeds %d MB", avatar.MaxInputBytes/(1<<20),
			))
		}
		updated, err := service.Upload(ctx, user.ID, file)
		if err != nil {
			return nil, avatarHTTPError(err)
		}
		return &avatarOutput{Body: auth.MapUser(updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "remove-my-avatar",
		Summary:     "Remove profile photo",
		Method:      http.MethodDelete,
		Path:        "/avatar",
		Tags:        []string{"Me"},
	}, func(ctx context.Context, _ *struct{}) (*avatarOutput, error) {
		if service == nil {
			return nil, huma.Error503ServiceUnavailable("avatar uploads are not configured")
		}
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		updated, err := service.Remove(ctx, user.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to remove profile photo")
		}
		return &avatarOutput{Body: auth.MapUser(updated)}, nil
	})
}

func prepareAvatarUpload(api huma.API, service AvatarService) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if service == nil {
			_ = huma.WriteErr(api, ctx, http.StatusServiceUnavailable, "avatar uploads are not configured")
			return
		}
		request, writer := humachi.Unwrap(ctx)
		if request.ContentLength > maxAvatarMultipartBytes {
			_ = huma.WriteErr(api, ctx, http.StatusRequestEntityTooLarge, "request body exceeds upload limit")
			return
		}
		request.Body = http.MaxBytesReader(writer, request.Body, maxAvatarMultipartBytes)
		if err := request.ParseMultipartForm(humachi.MultipartMaxMemory); err != nil {
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				_ = huma.WriteErr(api, ctx, http.StatusRequestEntityTooLarge, "request body exceeds upload limit")
				return
			}
			_ = huma.WriteErr(api, ctx, http.StatusBadRequest, "cannot read multipart form")
			return
		}
		next(ctx)
	}
}

func avatarHTTPError(err error) error {
	switch {
	case errors.Is(err, avatar.ErrInputTooLarge):
		return huma.Error413RequestEntityTooLarge(err.Error())
	case errors.Is(err, avatar.ErrInvalidImage),
		errors.Is(err, avatar.ErrUnsupportedFormat),
		errors.Is(err, avatar.ErrDimensionsTooLarge):
		return huma.Error422UnprocessableEntity(err.Error())
	default:
		return huma.Error500InternalServerError("failed to upload profile photo")
	}
}
