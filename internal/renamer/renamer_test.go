package renamer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile creates a file with the given name and content inside dir, failing
// the test on error.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", name, err)
	}
}

// assertExists asserts whether a name exists inside dir.
func assertExists(t *testing.T, dir, name string, want bool) {
	t.Helper()
	_, err := os.Stat(filepath.Join(dir, name))
	if got := err == nil; got != want {
		t.Errorf("exists(%q) = %v, want %v", name, got, want)
	}
}

func TestRun_RenamesFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Café com Leite.txt", "x")
	writeFile(t, dir, "meu backup.tar.gz", "x")

	var out, errOut bytes.Buffer
	if n := Run([]string{dir}, Options{}, &out, &errOut); n != 0 {
		t.Fatalf("error count = %d, want 0; stderr=%q", n, errOut.String())
	}

	assertExists(t, dir, "cafe_com_leite.txt", true)
	assertExists(t, dir, "meu_backup.tar.gz", true)
	assertExists(t, dir, "Café com Leite.txt", false)
	if !strings.Contains(out.String(), "->") {
		t.Errorf("stdout missing rename lines: %q", out.String())
	}
}

func TestRun_DryRunLeavesDiskUnchanged(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "My File.txt", "x")

	var out, errOut bytes.Buffer
	Run([]string{dir}, Options{DryRun: true}, &out, &errOut)

	// Nothing moved on disk.
	assertExists(t, dir, "My File.txt", true)
	assertExists(t, dir, "my_file.txt", false)

	// But the preview was reported, tagged as a dry run.
	if !strings.Contains(out.String(), "[dry-run]") {
		t.Errorf("stdout missing [dry-run] prefix: %q", out.String())
	}
}

func TestRun_SkipsHiddenByDefault(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".Hidden File.txt", "x")
	writeFile(t, dir, "Visible File.txt", "x")

	var out, errOut bytes.Buffer
	Run([]string{dir}, Options{}, &out, &errOut)

	assertExists(t, dir, "visible_file.txt", true)
	assertExists(t, dir, ".Hidden File.txt", true) // untouched
	assertExists(t, dir, ".hidden_file.txt", false)
}

func TestRun_IncludesHiddenWithAll(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".Hidden File.txt", "x")

	var out, errOut bytes.Buffer
	Run([]string{dir}, Options{All: true}, &out, &errOut)

	assertExists(t, dir, ".hidden_file.txt", true)
	assertExists(t, dir, ".Hidden File.txt", false)
}

func TestRun_Collision(t *testing.T) {
	dir := t.TempDir()
	// Both normalize to "foo_bar.txt". Sorted order processes "Foo Bar.txt"
	// first (uppercase 'F' < lowercase 'f'), so it wins and the other is skipped.
	writeFile(t, dir, "Foo Bar.txt", "first")
	writeFile(t, dir, "foo-bar.txt", "second")

	var out, errOut bytes.Buffer
	Run([]string{dir}, Options{}, &out, &errOut)

	assertExists(t, dir, "foo_bar.txt", true)
	assertExists(t, dir, "Foo Bar.txt", false) // renamed
	assertExists(t, dir, "foo-bar.txt", true)  // skipped, left in place
	if !strings.Contains(errOut.String(), "skipped") {
		t.Errorf("stderr missing collision warning: %q", errOut.String())
	}
}

func TestRun_NeverOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "report.txt", "original") // already normalized, present
	writeFile(t, dir, "Report.txt", "new")      // would normalize onto it

	var out, errOut bytes.Buffer
	Run([]string{dir}, Options{}, &out, &errOut)

	// The pre-existing file is untouched.
	got, err := os.ReadFile(filepath.Join(dir, "report.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original" {
		t.Errorf("report.txt content = %q, want %q (must not be overwritten)", got, "original")
	}
	assertExists(t, dir, "Report.txt", true)
	if !strings.Contains(errOut.String(), "already exists") {
		t.Errorf("stderr missing overwrite warning: %q", errOut.String())
	}
}

func TestRun_AlreadyNormalizedIsNoOp(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "already_normalized.txt", "x")

	var out, errOut bytes.Buffer
	if n := Run([]string{dir}, Options{}, &out, &errOut); n != 0 {
		t.Fatalf("error count = %d, want 0", n)
	}

	assertExists(t, dir, "already_normalized.txt", true)
	if out.String() != "" {
		t.Errorf("expected no output for a no-op, got %q", out.String())
	}
}

func TestRun_Recursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "Sub Dir")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, sub, "Nested File.txt", "x")

	// Without recursion, the nested file is untouched (and the directory is
	// never renamed).
	var out, errOut bytes.Buffer
	Run([]string{dir}, Options{}, &out, &errOut)
	assertExists(t, sub, "Nested File.txt", true)

	// With recursion, the nested file is normalized in place.
	out.Reset()
	errOut.Reset()
	Run([]string{dir}, Options{Recursive: true}, &out, &errOut)
	assertExists(t, sub, "nested_file.txt", true)
	assertExists(t, sub, "Nested File.txt", false)
	assertExists(t, dir, "Sub Dir", true) // directory itself not renamed
}

func TestRun_ExplicitHiddenFileIsProcessed(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".Bash Config", "x")

	// Naming the hidden file explicitly bypasses the hidden filter, even
	// without All.
	var out, errOut bytes.Buffer
	Run([]string{filepath.Join(dir, ".Bash Config")}, Options{}, &out, &errOut)

	assertExists(t, dir, ".bash_config", true)
	assertExists(t, dir, ".Bash Config", false)
}
