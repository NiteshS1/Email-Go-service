package config_test

import (
	"os"
	"testing"

	"github.com/emailservice/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setEnv(t *testing.T, vals map[string]string) {
	t.Helper()
	for k, v := range vals {
		t.Setenv(k, v)
	}
}

func validEnv(t *testing.T) {
	t.Helper()
	setEnv(t, map[string]string{
		"DB_USER":     "user",
		"DB_PASSWORD": "pass",
		"DB_NAME":     "testdb",
	})
}

func TestLoad_ValidConfig(t *testing.T) {
	validEnv(t)
	t.Setenv("DB_HOST", "myhost")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("SMTP_PORT", "25")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "user", cfg.DBUser)
	assert.Equal(t, "pass", cfg.DBPassword)
	assert.Equal(t, "testdb", cfg.DBName)
	assert.Equal(t, "myhost", cfg.DBHost)
	assert.Equal(t, "5433", cfg.DBPort)
	assert.Equal(t, "9090", cfg.AppPort)
	assert.Equal(t, 25, cfg.SMTPPort)
}

func TestLoad_Defaults(t *testing.T) {
	validEnv(t)

	cfg, err := config.Load()
	require.NoError(t, err)
	// Defaults should be applied when not set
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, "5432", cfg.DBPort)
	assert.Equal(t, "8080", cfg.AppPort)
	assert.Equal(t, 587, cfg.SMTPPort)
}

func TestLoad_MissingRequired(t *testing.T) {
	tests := []struct {
		name    string
		missing string
	}{
		{"missing DB_USER", "DB_USER"},
		{"missing DB_PASSWORD", "DB_PASSWORD"},
		{"missing DB_NAME", "DB_NAME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validEnv(t)
			os.Unsetenv(tt.missing)

			_, err := config.Load()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.missing)
		})
	}
}

func TestLoad_InvalidSMTPPort(t *testing.T) {
	validEnv(t)
	t.Setenv("SMTP_PORT", "not-a-number")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMTP_PORT")
}
