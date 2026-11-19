package handler

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanPageImageRelativePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{name: "single filename", in: "logo.png", want: "logo.png", ok: true},
		{name: "nested path", in: "images/logo.png", want: filepath.Join("images", "logo.png"), ok: true},
		{name: "dot prefix", in: "./logo.png", want: "logo.png", ok: true},
		{name: "url escaped slash", in: "images%2Flogo.png", want: filepath.Join("images", "logo.png"), ok: true},
		{name: "parent traversal", in: "../secret.png", ok: false},
		{name: "encoded parent traversal", in: "%2e%2e/secret.png", ok: false},
		{name: "backslash traversal", in: `images\secret.png`, ok: false},
		{name: "absolute path", in: "/etc/passwd", ok: false},
		{name: "encoded absolute path", in: "%2fetc/passwd", ok: false},
		{name: "encoded nul byte", in: "logo.png%00", ok: false},
		{name: "invalid escape", in: "logo.png%zz", ok: false},
		{name: "empty path", in: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := cleanPageImageRelativePath(tt.in)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("path = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOpenPageImageFiles(t *testing.T) {
	pagesDir := t.TempDir()
	base := filepath.Join(pagesDir, "guide")
	if err := os.MkdirAll(filepath.Join(base, "images"), 0755); err != nil {
		t.Fatal(err)
	}
	root, err := openPageImagesDir(pagesDir, "guide")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	for _, name := range []string{"logo.png", filepath.Join("images", "logo.png")} {
		if err := os.WriteFile(filepath.Join(base, name), []byte("fake"), 0644); err != nil {
			t.Fatal(err)
		}
		file, _, err := openRegularPageFile(root, name)
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(file)
		_ = file.Close()
		if err != nil || string(content) != "fake" {
			t.Fatalf("opened %q: contents = %q, err = %v", name, content, err)
		}
	}
	if file, _, err := openRegularPageFile(root, "images"); err == nil {
		_ = file.Close()
		t.Fatal("expected directories to be rejected")
	}
}

func TestOpenPageFilesConfineSymlinks(t *testing.T) {
	parent := t.TempDir()
	pagesDir := filepath.Join(parent, "pages")
	base := filepath.Join(pagesDir, "guide")
	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{filepath.Join(base, "image.png"), filepath.Join(pagesDir, "other.png"), filepath.Join(parent, "other.md")} {
		if err := os.WriteFile(name, []byte("file"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, link := range []struct{ name, target string }{
		{filepath.Join(base, "local.png"), "image.png"},
		{filepath.Join(base, "outside.png"), filepath.Join("..", "other.png")},
		{filepath.Join(pagesDir, "outside.md"), filepath.Join("..", "other.md")},
		{filepath.Join(pagesDir, "outside"), ".."},
	} {
		if err := os.Symlink(link.target, link.name); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}
	images, err := openPageImagesDir(pagesDir, "guide")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = images.Close() }()
	file, _, err := openRegularPageFile(images, "local.png")
	if err != nil {
		t.Fatalf("in-directory image link: %v", err)
	}
	_ = file.Close()
	if file, _, err = openRegularPageFile(images, "outside.png"); err == nil {
		_ = file.Close()
		t.Fatal("expected image link outside slug directory to be rejected")
	}
	pages, err := os.OpenRoot(pagesDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pages.Close() }()
	if file, _, err = openRegularPageFile(pages, "outside.md"); err == nil {
		_ = file.Close()
		t.Fatal("expected markdown link outside pages directory to be rejected")
	}
	if otherDir, err := openPageImagesDir(pagesDir, "outside"); err == nil {
		_ = otherDir.Close()
		t.Fatal("expected slug links outside pages directory to be rejected")
	}
}
