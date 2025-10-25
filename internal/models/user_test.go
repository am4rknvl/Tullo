package models

import (
	"testing"
)

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
	}{
		{
			name: "Valid user",
			user: User{
				Email:       "test@example.com",
				DisplayName: "Test User",
			},
			wantErr: false,
		},
		{
			name: "Empty email",
			user: User{
				Email:       "",
				DisplayName: "Test User",
			},
			wantErr: true,
		},
		{
			name: "Invalid email",
			user: User{
				Email:       "invalid-email",
				DisplayName: "Test User",
			},
			wantErr: true,
		},
		{
			name: "Empty display name",
			user: User{
				Email:       "test@example.com",
				DisplayName: "",
			},
			wantErr: true,
		},
		{
			name: "Display name too short",
			user: User{
				Email:       "test@example.com",
				DisplayName: "A",
			},
			wantErr: true,
		},
		{
			name: "Display name too long",
			user: User{
				Email:       "test@example.com",
				DisplayName: "This is a very long display name that exceeds the maximum allowed length of 100 characters for testing purposes",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("User.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
