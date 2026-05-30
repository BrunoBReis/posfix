package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_Version(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"--version"}, &out, &errOut); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "posfix") {
		t.Errorf("version output = %q, want it to contain %q", out.String(), "posfix")
	}
}

func TestRun_Help(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"--help"}, &out, &errOut); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "Usage") {
		t.Errorf("help output = %q, want it to contain %q", out.String(), "Usage")
	}
}

func TestRun_UnknownFlagExitsTwo(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"--definitely-not-a-flag"}, &out, &errOut); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestRun_DryRunDoesNotRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "My File.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if code := run([]string{"-n", path}, &out, &errOut); code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", code, errOut.String())
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("dry-run must not rename the file: %v", err)
	}
	if !strings.Contains(out.String(), "[dry-run]") {
		t.Errorf("output = %q, want it to contain %q", out.String(), "[dry-run]")
	}
}
