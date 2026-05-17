package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nexspence/nxs/internal/config"
)

func TestLoad_FromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte(`
current_context: local
contexts:
  local:
    url: http://localhost:8080
    token: nxs_abc
`), 0600)
	t.Setenv("NXS_CONFIG", cfgPath)
	t.Setenv("NXS_URL", "")
	t.Setenv("NXS_TOKEN", "")

	cfg, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CurrentURL() != "http://localhost:8080" {
		t.Errorf("url: got %q", cfg.CurrentURL())
	}
	if cfg.CurrentToken() != "nxs_abc" {
		t.Errorf("token: got %q", cfg.CurrentToken())
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte(`
current_context: local
contexts:
  local:
    url: http://localhost:8080
    token: nxs_file_token
`), 0600)
	t.Setenv("NXS_CONFIG", cfgPath)
	t.Setenv("NXS_URL", "http://prod:8080")
	t.Setenv("NXS_TOKEN", "nxs_env_token")

	cfg, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CurrentURL() != "http://prod:8080" {
		t.Errorf("url: got %q", cfg.CurrentURL())
	}
	if cfg.CurrentToken() != "nxs_env_token" {
		t.Errorf("token: got %q", cfg.CurrentToken())
	}
}

func TestLoad_ContextSwitch(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte(`
current_context: local
contexts:
  local:
    url: http://local:8080
    token: nxs_local
  prod:
    url: http://prod:8080
    token: nxs_prod
`), 0600)
	t.Setenv("NXS_CONFIG", cfgPath)
	t.Setenv("NXS_URL", "")
	t.Setenv("NXS_TOKEN", "")

	cfg, err := config.Load("prod")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CurrentURL() != "http://prod:8080" {
		t.Errorf("url: got %q", cfg.CurrentURL())
	}
}
