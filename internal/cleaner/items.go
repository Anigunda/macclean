package cleaner

import (
	"os"
	"path/filepath"
	"sort"
)

// ItemEntry is one cleanable item with its owning category and advice.
type ItemEntry struct {
	Item
	TargetID   string
	TargetName string
	Keep       bool
	Reason     string
}

// CollectAllItems is CollectItems over all targets of this machine.
func CollectAllItems() []ItemEntry {
	return CollectItems(Targets())
}

// CollectItems flattens the given targets into individual items:
// direct children of cache/temp directories, or the paths themselves for
// RemoveWhole targets (app leftovers, stale node_modules). Empty items are
// dropped. Result is sorted biggest first.
func CollectItems(targets []Target) []ItemEntry {
	var paths []string
	var owners []Target
	for _, t := range targets {
		for _, p := range collectPaths(t) {
			paths = append(paths, p)
			owners = append(owners, t)
		}
	}

	measured := measurePaths(paths)

	var out []ItemEntry
	for i, it := range measured {
		if it.Size == 0 {
			continue
		}
		keep, reason := Suggest(owners[i].ID, filepath.Base(it.Path))
		out = append(out, ItemEntry{
			Item:       it,
			TargetID:   owners[i].ID,
			TargetName: owners[i].Name,
			Keep:       keep,
			Reason:     reason,
		})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Size > out[b].Size })
	return out
}

// collectPaths returns the item-level paths for a target: the target paths
// themselves when they are removed whole, else their direct children.
func collectPaths(t Target) []string {
	if t.RemoveWhole {
		var out []string
		for _, p := range t.Paths {
			if pathExists(p) {
				out = append(out, p)
			}
		}
		return out
	}
	var out []string
	for _, dir := range t.Paths {
		if !dirExists(dir) {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out
}

// EntriesResult reports the outcome of CleanEntries.
type EntriesResult struct {
	Freed   int64
	Removed int64
	Failed  int64
	Paths   []string // original paths that were removed
}

// CleanEntries deletes the given items (whichever is selected) and reports
// the bytes freed. Locked/in-use files are skipped, not fatal.
func CleanEntries(entries []ItemEntry) EntriesResult {
	var res EntriesResult
	for _, e := range entries {
		// RemoveAll on a symlink removes the link only - never followed.
		if err := os.RemoveAll(e.Path); err != nil {
			res.Failed++
			continue
		}
		res.Freed += e.Size
		res.Removed++
		res.Paths = append(res.Paths, e.Path)
	}
	return res
}
