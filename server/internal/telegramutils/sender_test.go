package telegramutils

import "testing"

func TestResolveSender(t *testing.T) {
	tests := []struct {
		name            string
		user            *UserInfo
		userID          int64
		wantUsername    string
		wantDisplayName string
	}{
		{
			name:            "full user with username",
			user:            &UserInfo{Username: "darrel", FirstName: "Darrel", LastName: "Hong"},
			userID:          12345,
			wantUsername:    "darrel",
			wantDisplayName: "Darrel Hong",
		},
		{
			name:            "user with username but no name",
			user:            &UserInfo{Username: "darrel"},
			userID:          12345,
			wantUsername:    "darrel",
			wantDisplayName: "darrel",
		},
		{
			name:            "user with name but no username",
			user:            &UserInfo{FirstName: "Darrel", LastName: "Hong"},
			userID:          12345,
			wantUsername:    "",
			wantDisplayName: "Darrel Hong",
		},
		{
			name:            "user with first name only",
			user:            &UserInfo{FirstName: "Darrel"},
			userID:          12345,
			wantUsername:    "",
			wantDisplayName: "Darrel",
		},
		{
			name:            "user with last name only",
			user:            &UserInfo{LastName: "Hong"},
			userID:          12345,
			wantUsername:    "",
			wantDisplayName: "Hong",
		},
		{
			name:            "empty user falls back to user ID",
			user:            &UserInfo{},
			userID:          12345,
			wantUsername:    "",
			wantDisplayName: "User_12345",
		},
		{
			name:            "nil user falls back to user ID",
			user:            nil,
			userID:          12345,
			wantUsername:    "",
			wantDisplayName: "User_12345",
		},
		{
			name:            "nil user and zero ID falls back to Unknown",
			user:            nil,
			userID:          0,
			wantUsername:    "",
			wantDisplayName: "Unknown",
		},
		{
			name:            "username with empty names and zero ID uses username as display",
			user:            &UserInfo{Username: "bot_user"},
			userID:          0,
			wantUsername:    "bot_user",
			wantDisplayName: "bot_user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUsername, gotDisplayName := ResolveSender(tt.user, tt.userID)
			if gotUsername != tt.wantUsername {
				t.Errorf("username: got %q, want %q", gotUsername, tt.wantUsername)
			}
			if gotDisplayName != tt.wantDisplayName {
				t.Errorf("displayName: got %q, want %q", gotDisplayName, tt.wantDisplayName)
			}
		})
	}
}
