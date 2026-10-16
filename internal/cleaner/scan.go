package cleaner

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Result holds the scan outcome for one Target.
type Result struct {
	Target Target
	Paths  []string // target paths that actually exist on disk
	Size   int64    // total bytes
	Files  int64    // number of regular files
}

// Scan measures all targets concurrently. It is read-only.
func Scan(targets []Target) []Result {
	results := make([]Result, len(targets))
	var wg sync.WaitGroup
	for i, t := range targets {
		wg.Add(1)
		go func(i int, t Target) {
			defer wg.Done()
			r := Result{Target: t}
			for _, p := range t.Paths {
				if !pathExists(p) { // leftover paths can be files (.plist) too
					continue
				}
				r.Paths = append(r.Paths, p)
				size, files := measure(p)
				r.Size += size
				r.Files += files
			}
			results[i] = r
		}(i, t)
	}
	wg.Wait()
	return results
}

func measure(root string) (size int64, files int64) {
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				size += info.Size()
				files++
			}
		}
		return nil
	})
	return size, files
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
