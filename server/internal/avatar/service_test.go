package avatar

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"openplays/server/internal/db"
	"openplays/server/internal/objectstore"
)

type fakeObjects struct {
	putKey                    string
	putBody                   string
	putOptions                objectstore.PutOptions
	putErr, deleteErr, urlErr error
	deleted                   []string
}

func (f *fakeObjects) Put(_ context.Context, key string, body io.Reader, options objectstore.PutOptions) error {
	f.putKey, f.putOptions = key, options
	b, _ := io.ReadAll(body)
	f.putBody = string(b)
	return f.putErr
}
func (f *fakeObjects) Delete(_ context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	return f.deleteErr
}
func (f *fakeObjects) PublicURL(key string) (string, error) {
	return "https://images.example/" + key, f.urlErr
}

type fakeUsers struct {
	user                     db.User
	getErr, setErr, clearErr error
	applySetOnError          bool
	set                      *db.SetUserAvatarParams
	cleared                  string
}

func (f *fakeUsers) GetUserByID(context.Context, string) (db.User, error) { return f.user, f.getErr }
func (f *fakeUsers) SetUserAvatar(_ context.Context, p db.SetUserAvatarParams) (db.User, error) {
	f.set = &p
	if f.setErr != nil && !f.applySetOnError {
		return db.User{}, f.setErr
	}
	f.user.PhotoUrl, f.user.AvatarKey = p.PhotoUrl, p.AvatarKey
	if f.setErr != nil {
		return db.User{}, f.setErr
	}
	return f.user, nil
}
func (f *fakeUsers) ClearUserAvatar(_ context.Context, p db.ClearUserAvatarParams) (db.User, error) {
	f.cleared = p.ID
	if f.clearErr != nil {
		return db.User{}, f.clearErr
	}
	f.user.PhotoUrl, f.user.AvatarKey = f.user.OauthPhotoUrl, nil
	return f.user, nil
}

type fakeProcessor struct {
	result ProcessedImage
	err    error
}

func (f fakeProcessor) Process(io.Reader) (ProcessedImage, error) { return f.result, f.err }

func testService(objects *fakeObjects, users *fakeUsers) *Service {
	return NewService(objects, users, fakeProcessor{result: ProcessedImage{Data: []byte("jpeg"), ContentType: "image/jpeg", Extension: ".jpg"}})
}

func TestServiceUploadReplacesAvatar(t *testing.T) {
	old := "avatars/user/old.jpg"
	objects := &fakeObjects{}
	users := &fakeUsers{user: db.User{ID: "user", AvatarKey: &old}}
	updated, err := testService(objects, users).Upload(context.Background(), "user", strings.NewReader("input"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(objects.putKey, "avatars/user/") || !strings.HasSuffix(objects.putKey, ".jpg") {
		t.Fatalf("key = %q", objects.putKey)
	}
	if objects.putBody != "jpeg" || objects.putOptions.ContentType != "image/jpeg" || objects.putOptions.CacheControl != avatarCacheControl {
		t.Fatalf("put = %#v body %q", objects.putOptions, objects.putBody)
	}
	if users.set == nil || users.set.ID != "user" || users.set.AvatarKey == nil || *users.set.AvatarKey != objects.putKey {
		t.Fatalf("set = %#v", users.set)
	}
	if users.set.ExpectedAvatarKey == nil || *users.set.ExpectedAvatarKey != old {
		t.Fatalf("expected avatar key = %v, want %q", users.set.ExpectedAvatarKey, old)
	}
	if updated.PhotoUrl == nil || *updated.PhotoUrl != "https://images.example/"+objects.putKey {
		t.Fatalf("photo = %v", updated.PhotoUrl)
	}
	if len(objects.deleted) != 1 || objects.deleted[0] != old {
		t.Fatalf("deleted = %v", objects.deleted)
	}
}

func TestServiceUploadReconcilesCommittedDatabaseError(t *testing.T) {
	o := &fakeObjects{}
	u := &fakeUsers{
		user: db.User{ID: "user"}, setErr: errors.New("context canceled"),
		applySetOnError: true,
	}
	got, err := testService(o, u).Upload(context.Background(), "user", strings.NewReader("input"))
	if err != nil {
		t.Fatal(err)
	}
	if got.AvatarKey == nil || *got.AvatarKey != o.putKey {
		t.Fatalf("avatar key = %v, want %q", got.AvatarKey, o.putKey)
	}
	if len(o.deleted) != 0 {
		t.Fatalf("deleted referenced object: %v", o.deleted)
	}
}

func TestServiceUploadFailureOrdering(t *testing.T) {
	tests := []struct {
		name        string
		configure   func(*fakeObjects, *fakeUsers, *fakeProcessor)
		wantDeletes int
		wantSet     bool
	}{
		{"process", func(_ *fakeObjects, _ *fakeUsers, p *fakeProcessor) { p.err = errors.New("bad image") }, 0, false},
		{"public URL", func(o *fakeObjects, _ *fakeUsers, _ *fakeProcessor) { o.urlErr = errors.New("bad URL") }, 0, false},
		{"put", func(o *fakeObjects, _ *fakeUsers, _ *fakeProcessor) { o.putErr = errors.New("put") }, 0, false},
		{"database", func(_ *fakeObjects, u *fakeUsers, _ *fakeProcessor) { u.setErr = errors.New("db") }, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, u := &fakeObjects{}, &fakeUsers{user: db.User{ID: "user"}}
			p := &fakeProcessor{result: ProcessedImage{Data: []byte("jpeg"), ContentType: "image/jpeg", Extension: ".jpg"}}
			tt.configure(o, u, p)
			_, err := NewService(o, u, p).Upload(context.Background(), "user", strings.NewReader("input"))
			if err == nil {
				t.Fatal("expected error")
			}
			if len(o.deleted) != tt.wantDeletes {
				t.Fatalf("deletes = %v", o.deleted)
			}
			if (u.set != nil) != tt.wantSet {
				t.Fatalf("set called = %v", u.set != nil)
			}
		})
	}
}

func TestServiceUploadIgnoresOldDeleteFailure(t *testing.T) {
	old := "avatars/user/old.jpg"
	o := &fakeObjects{deleteErr: errors.New("delete")}
	u := &fakeUsers{user: db.User{ID: "user", AvatarKey: &old}}
	if _, err := testService(o, u).Upload(context.Background(), "user", strings.NewReader("input")); err != nil {
		t.Fatal(err)
	}
}

func TestServiceRemove(t *testing.T) {
	providerPhoto, old := "https://provider/photo.jpg", "avatars/user/old.jpg"
	o := &fakeObjects{}
	u := &fakeUsers{user: db.User{ID: "user", PhotoUrl: strptr("https://custom/photo.jpg"), OauthPhotoUrl: &providerPhoto, AvatarKey: &old}}
	got, err := testService(o, u).Remove(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	if u.cleared != "user" || got.PhotoUrl == nil || *got.PhotoUrl != providerPhoto {
		t.Fatalf("clear result = %#v", got)
	}
	if len(o.deleted) != 1 || o.deleted[0] != old {
		t.Fatalf("deleted = %v", o.deleted)
	}
}

func TestServiceRemoveWithoutAvatarIsNoOp(t *testing.T) {
	o, u := &fakeObjects{}, &fakeUsers{user: db.User{ID: "user"}}
	if _, err := testService(o, u).Remove(context.Background(), "user"); err != nil {
		t.Fatal(err)
	}
	if u.cleared != "" || len(o.deleted) != 0 {
		t.Fatalf("clear = %q, deleted = %v", u.cleared, o.deleted)
	}
}

func TestServiceRemoveDoesNotDeleteWhenClearFails(t *testing.T) {
	old := "avatars/user/old.jpg"
	o, u := &fakeObjects{}, &fakeUsers{user: db.User{ID: "user", AvatarKey: &old}, clearErr: errors.New("db")}
	if _, err := testService(o, u).Remove(context.Background(), "user"); err == nil {
		t.Fatal("expected error")
	}
	if len(o.deleted) != 0 {
		t.Fatalf("deleted = %v", o.deleted)
	}
}

func strptr(value string) *string { return &value }
