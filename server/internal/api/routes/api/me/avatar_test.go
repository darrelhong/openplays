package me_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/routes/api/me"
	"openplays/server/internal/auth"
	"openplays/server/internal/avatar"
	"openplays/server/internal/db"
)

type fakeAvatarService struct {
	result                 db.User
	uploadErr, removeErr   error
	uploadUser, removeUser string
	uploaded               []byte
}

func (f *fakeAvatarService) Upload(_ context.Context, userID string, body io.Reader) (db.User, error) {
	f.uploadUser = userID
	f.uploaded, _ = io.ReadAll(body)
	return f.result, f.uploadErr
}

func (f *fakeAvatarService) Remove(_ context.Context, userID string) (db.User, error) {
	f.removeUser = userID
	return f.result, f.removeErr
}

func setupAvatar(authStore *fakeAuthStore, avatarService me.AvatarService) *httptest.Server {
	router := chi.NewRouter()
	api := humachi.New(router, huma.DefaultConfig("test", "0.0.1"))
	group := huma.NewGroup(api, "/api/me")
	group.UseMiddleware(authmw.RequireAuth(api, auth.NewService(authStore)))
	me.RegisterAvatar(group, avatarService)
	return httptest.NewServer(router)
}

func avatarResultUser() db.User {
	now := time.Now()
	photo := "https://images.example/avatar.jpg"
	return db.User{
		ID: "user-1", Email: "test@test.com", DisplayName: "Test User",
		PhotoUrl: &photo, Status: "active", CreatedAt: now, UpdatedAt: now,
	}
}

func avatarRequest(t *testing.T, method, url string, size int) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, "avatar", "avatar.jpg"))
	header.Set("Content-Type", "image/jpeg")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(bytes.Repeat([]byte{0xff}, size)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(method, url, &body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func TestUploadAvatar(t *testing.T) {
	service := &fakeAvatarService{result: avatarResultUser()}
	server := setupAvatar(activeSession(), service)
	defer server.Close()
	request := avatarRequest(t, http.MethodPut, server.URL+"/api/me/avatar", 128)
	request.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if service.uploadUser != "user-1" || len(service.uploaded) != 128 {
		t.Fatalf("upload user/body = %q/%d", service.uploadUser, len(service.uploaded))
	}
}

func TestUploadAvatarRequiresAuthentication(t *testing.T) {
	service := &fakeAvatarService{result: avatarResultUser()}
	server := setupAvatar(activeSession(), service)
	defer server.Close()
	response, err := http.DefaultClient.Do(avatarRequest(t, http.MethodPut, server.URL+"/api/me/avatar", 10))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized || service.uploadUser != "" {
		t.Fatalf("status/user = %d/%q", response.StatusCode, service.uploadUser)
	}
}

func TestUploadAvatarRejectsOversizedFile(t *testing.T) {
	service := &fakeAvatarService{result: avatarResultUser()}
	server := setupAvatar(activeSession(), service)
	defer server.Close()
	request := avatarRequest(t, http.MethodPut, server.URL+"/api/me/avatar", avatar.MaxInputBytes+1)
	request.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusRequestEntityTooLarge || service.uploadUser != "" {
		t.Fatalf("status/user = %d/%q", response.StatusCode, service.uploadUser)
	}
}

func TestUploadAvatarRejectsOversizedMultipartBody(t *testing.T) {
	service := &fakeAvatarService{result: avatarResultUser()}
	server := setupAvatar(activeSession(), service)
	defer server.Close()
	request := avatarRequest(t, http.MethodPut, server.URL+"/api/me/avatar", avatar.MaxInputBytes+128<<10)
	request.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusRequestEntityTooLarge || service.uploadUser != "" {
		t.Fatalf("status/user = %d/%q", response.StatusCode, service.uploadUser)
	}
}

func TestUploadAvatarRejectsOversizedChunkedBody(t *testing.T) {
	service := &fakeAvatarService{result: avatarResultUser()}
	server := setupAvatar(activeSession(), service)
	defer server.Close()
	request := avatarRequest(t, http.MethodPut, server.URL+"/api/me/avatar", avatar.MaxInputBytes+128<<10)
	request.ContentLength = -1
	request.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusRequestEntityTooLarge || service.uploadUser != "" {
		t.Fatalf("status/user = %d/%q", response.StatusCode, service.uploadUser)
	}
}

func TestUploadAvatarMapsValidationErrors(t *testing.T) {
	service := &fakeAvatarService{uploadErr: avatar.ErrInvalidImage}
	server := setupAvatar(activeSession(), service)
	defer server.Close()
	request := avatarRequest(t, http.MethodPut, server.URL+"/api/me/avatar", 10)
	request.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", response.StatusCode)
	}
}

func TestRemoveAvatar(t *testing.T) {
	service := &fakeAvatarService{result: avatarResultUser()}
	server := setupAvatar(activeSession(), service)
	defer server.Close()
	request, err := http.NewRequest(http.MethodDelete, server.URL+"/api/me/avatar", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || service.removeUser != "user-1" {
		t.Fatalf("status/user = %d/%q", response.StatusCode, service.removeUser)
	}
}

func TestAvatarRoutesUnavailableWithoutObjectStore(t *testing.T) {
	server := setupAvatar(activeSession(), nil)
	defer server.Close()
	request := avatarRequest(t, http.MethodPut, server.URL+"/api/me/avatar", 10)
	request.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.StatusCode)
	}
}

func TestUnavailableAvatarUploadDoesNotParseBody(t *testing.T) {
	server := setupAvatar(activeSession(), nil)
	defer server.Close()
	request, err := http.NewRequest(http.MethodPut, server.URL+"/api/me/avatar", bytes.NewBufferString("not multipart"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "multipart/form-data; boundary=broken")
	request.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.StatusCode)
	}
}
