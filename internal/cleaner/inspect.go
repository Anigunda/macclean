package cleaner

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Item is one entry with its measured size.
type Item struct {
	Path  string
	Size  int64
	Files int64
}

// TopItems measures the direct children of every path in a scanned target
// and returns the n largest, biggest first.
func TopItems(r Result, n int) []Item {
	var children []string
	for _, dir := range r.Paths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			children = append(children, filepath.Join(dir, e.Name()))
		}
	}
	return limitItems(measurePaths(children), n)
}

// ItemsFromPaths measures the target's paths directly - used when the paths
// themselves are the items (e.g. app leftovers, where each path is deleted whole).
func ItemsFromPaths(r Result, n int) []Item {
	items := make([]Item, 0, len(r.Paths))
	for _, p := range r.Paths {
		size, files := measure(p)
		items = append(items, Item{Path: p, Size: size, Files: files})
	}
	return limitItems(items, n)
}

// measurePaths measures every path concurrently (bounded) and returns one Item per path.
func measurePaths(paths []string) []Item {
	items := make([]Item, len(paths))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8) // bound concurrency on big directory trees
	for i, p := range paths {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, p string) {
			defer wg.Done()
			defer func() { <-sem }()
			items[i].Path = p
			size, files := measure(p)
			items[i].Size = size
			items[i].Files = files
		}(i, p)
	}
	wg.Wait()
	return items
}

func limitItems(items []Item, n int) []Item {
	sort.Slice(items, func(i, j int) bool { return items[i].Size > items[j].Size })
	if len(items) > n {
		items = items[:n]
	}
	return items
}
