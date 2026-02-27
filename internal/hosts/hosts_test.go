package hosts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndResolve(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.hosts")
	content := "# comment\n1.2.3.4 Example.COM alias.example.com\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	table, err := Load(p)
	if err != nil {
		t.Fatalf("load err: %v", err)
	}

	if got, ok := table.Resolve("example.com"); !ok || got != "1.2.3.4" {
		t.Fatalf("unexpected resolve: %v %v", got, ok)
	}
	if got, ok := table.Resolve("ALIAS.EXAMPLE.COM"); !ok || got != "1.2.3.4" {
		t.Fatalf("unexpected alias resolve: %v %v", got, ok)
	}
}
