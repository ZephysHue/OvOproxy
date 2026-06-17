package hosts

import (
	"os"
	"path/filepath"
	"strings"
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

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "multi-domain line expanded",
			in:   "120.92.124.158 i.wps.cn account.wps.cn api.wps.cn",
			want: "120.92.124.158 i.wps.cn\n120.92.124.158 account.wps.cn\n120.92.124.158 api.wps.cn",
		},
		{
			name: "single domain unchanged",
			in:   "1.2.3.4 example.com",
			want: "1.2.3.4 example.com",
		},
		{
			name: "comments and blanks preserved",
			in:   "# header\n\n1.2.3.4 a.com b.com\n# footer",
			want: "# header\n\n1.2.3.4 a.com\n1.2.3.4 b.com\n# footer",
		},
		{
			name: "inline comment kept on first expanded line",
			in:   "1.2.3.4 a.com b.com # wps",
			want: "1.2.3.4 a.com # wps\n1.2.3.4 b.com",
		},
		{
			name: "CRLF preserved",
			in:   "1.2.3.4 a.com b.com\r\n2.3.4.5 c.com",
			want: "1.2.3.4 a.com\r\n1.2.3.4 b.com\r\n2.3.4.5 c.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeText(tt.in)
			if got != tt.want {
				t.Errorf("NormalizeText:\n got: %s\nwant: %s", strings.ReplaceAll(got, "\r", "\\r"), strings.ReplaceAll(tt.want, "\r", "\\r"))
			}
		})
	}
}
