package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aqasim81/database-migration-engine/internal/config"
)

func TestRedactURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "full URL with password",
			raw:  "postgres://admin:s3cret@db.example.com:5432/mydb?sslmode=require",
			want: "postgres://admin:***@db.example.com:5432/mydb?sslmode=require",
		},
		{
			name: "URL without password",
			raw:  "postgres://admin@localhost:5432/mydb",
			want: "postgres://admin@localhost:5432/mydb",
		},
		{
			name: "URL without userinfo",
			raw:  "postgres://localhost:5432/mydb",
			want: "postgres://localhost:5432/mydb",
		},
		{
			name: "empty string",
			raw:  "",
			want: "",
		},
		{
			name: "unparseable string",
			raw:  "://not-a-url",
			want: "://not-a-url",
		},
		{
			name: "password with special characters",
			raw:  "postgres://user:p%40ss%23word@host:5432/db",
			want: "postgres://user:***@host:5432/db",
		},
		{
			name: "empty password",
			raw:  "postgres://user:@host:5432/db",
			want: "postgres://user:***@host:5432/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := config.RedactURL(tt.raw)
			assert.Equal(t, tt.want, got)
		})
	}
}
