// Package renamer is the filesystem layer of posfix. It discovers target
// files, asks the pure normalize package for each new name, and performs the
// renames, refusing to overwrite anything that already exists.
//
// All output is written through injected io.Writers so the behavior can be
// covered by tests that capture stdout/stderr into buffers and inspect a
// temporary directory.
package renamer

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brunobreis/posfix/internal/normalize"
)

// Options controls a single renaming run.
type Options struct {
	DryRun    bool            // report changes without touching the disk
	Recursive bool            // descend into subdirectories
	All       bool            // include hidden files when scanning directories
	Style     normalize.Style // naming style; defaults to normalize.Snake
}

// Run normalizes the names of the files found under targets and renames them on
// disk (unless Options.DryRun is set). With no targets it processes the current
// directory.
//
// It returns the number of hard errors (unreadable paths, failed renames).
// Skips and collisions are warnings, not errors. Each rename is reported to
// stdout as "old  ->  new" (prefixed with "[dry-run] " in dry-run mode); skips
// and errors go to stderr.
func Run(targets []string, opts Options, stdout, stderr io.Writer) int {
	if len(targets) == 0 {
		targets = []string{"."}
	}
	if opts.Style == nil {
		opts.Style = normalize.Snake{}
	}

	// Group discovered files by their parent directory so that collision
	// detection is scoped to one directory at a time.
	byDir, errs := collect(targets, opts, stderr)

	dirs := make([]string, 0, len(byDir))
	for dir := range byDir {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		errs += renameDir(dir, byDir[dir], opts, stdout, stderr)
	}
	return errs
}

// collect resolves the targets into a map of directory -> file base names.
// A file target is taken as-is (an explicitly named file bypasses the hidden
// filter); a directory target is scanned, honoring the Recursive and All
// options.
func collect(targets []string, opts Options, stderr io.Writer) (map[string][]string, int) {
	byDir := make(map[string][]string)
	errs := 0

	for _, target := range targets {
		info, err := os.Lstat(target)
		if err != nil {
			fmt.Fprintf(stderr, "posfix: %v\n", err)
			errs++
			continue
		}
		if info.IsDir() {
			errs += scanDir(target, opts, byDir, stderr)
			continue
		}
		// A file named explicitly on the command line is processed even if it
		// is hidden, but we still only rename regular files.
		if info.Mode().IsRegular() {
			byDir[filepath.Dir(target)] = append(byDir[filepath.Dir(target)], filepath.Base(target))
		}
	}
	return byDir, errs
}

// scanDir adds the regular files inside dir to byDir. Without Recursive it looks
// only at direct children; with it, it walks the whole tree. Hidden entries are
// skipped unless All is set (and hidden directories are not descended into).
func scanDir(dir string, opts Options, byDir map[string][]string, stderr io.Writer) int {
	if !opts.Recursive {
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintf(stderr, "posfix: %v\n", err)
			return 1
		}
		for _, e := range entries {
			if keepEntry(e, opts.All) {
				byDir[dir] = append(byDir[dir], e.Name())
			}
		}
		return 0
	}

	errs := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(stderr, "posfix: %v\n", err)
			errs++
			return nil
		}
		if d.IsDir() {
			// Never skip the target dir itself; skip hidden subdirectories
			// (and their contents) unless All is set.
			if path != dir && isHidden(d.Name()) && !opts.All {
				return filepath.SkipDir
			}
			return nil
		}
		if keepEntry(d, opts.All) {
			byDir[filepath.Dir(path)] = append(byDir[filepath.Dir(path)], d.Name())
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(stderr, "posfix: %v\n", err)
		errs++
	}
	return errs
}

// keepEntry reports whether a scanned directory entry should be renamed: it must
// be a regular file, and not hidden unless all is set.
func keepEntry(e fs.DirEntry, all bool) bool {
	if !e.Type().IsRegular() {
		return false
	}
	return all || !isHidden(e.Name())
}

// renameDir renames the given files within one directory. Names are processed in
// sorted order for deterministic, reproducible behavior, and a "claimed" set
// guards against two source files normalizing onto the same destination.
func renameDir(dir string, names []string, opts Options, stdout, stderr io.Writer) int {
	sort.Strings(names)

	errs := 0
	claimed := make(map[string]bool)
	last := ""

	for i, name := range names {
		if i > 0 && name == last {
			continue // de-duplicate a target listed more than once
		}
		last = name

		newName := normalize.Normalize(name, opts.Style)
		if newName == name {
			// Already normalized: do nothing, but reserve the name so a sibling
			// cannot collide onto it.
			claimed[name] = true
			continue
		}

		dest := filepath.Join(dir, newName)
		if claimed[newName] || pathExists(dest) {
			fmt.Fprintf(stderr, "posfix: skipped %q: target %q already exists\n", name, newName)
			continue
		}

		fmt.Fprintf(stdout, "%s%s  ->  %s\n", dryRunPrefix(opts.DryRun), name, newName)
		if !opts.DryRun {
			if err := os.Rename(filepath.Join(dir, name), dest); err != nil {
				fmt.Fprintf(stderr, "posfix: %v\n", err)
				errs++
				continue
			}
		}
		claimed[newName] = true
	}
	return errs
}

// pathExists reports whether a path exists, following nothing (a broken symlink
// still counts as existing, so we never clobber it).
func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

func dryRunPrefix(dryRun bool) string {
	if dryRun {
		return "[dry-run] "
	}
	return ""
}
