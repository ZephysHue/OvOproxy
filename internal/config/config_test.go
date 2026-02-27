package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.json")
	json := `{"profiles":[{"name":"a","listen_ip":"127.0.0.1","port":8080,"hosts_file":"a.hosts"}]}`
	if err := os.WriteFile(p, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("load err: %v", err)
	}
	if len(cfg.Profiles) != 1 {
		t.Fatalf("want 1 profile")
	}
}
