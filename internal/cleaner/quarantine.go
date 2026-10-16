package cleaner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// QuarantineRoot is where quarantined batches live: ~/.macclean/quarantine/<batchID>/
func QuarantineRoot() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".macclean", "quarantine")
}

// BatchItem maps one quarantined path to its original location.
type BatchItem struct {
	From   string `json:"from"`
	Stored string `json:"stored"` // name inside the batch dir
	Size   int64  `json:"size"`
}

// Batch is one quarantine run: a directory with a manifest.json.
type Batch struct {
	ID      string      `json:"id"`
	Created time.Time   `json:"created"`
	Items   []BatchItem `json:"items"`
}

// Size returns the total bytes of the batch.
func (b Batch) Size() int64 {
	var s int64
	for _, it := range b.Items {
		s += it.Size
	}
	return s
}

// QuarantineOutcome reports a quarantine run.
type QuarantineOutcome struct {
	BatchID string
	Moved   int64
	Bytes   int64
	Failed  int
	Paths   []string
}

// QuarantineEntries moves items into a new batch instead of deleting them.
// Anything that cannot be moved is left in place and counted as failed.
func QuarantineEntries(entries []ItemEntry) QuarantineOutcome {
	out := QuarantineOutcome{BatchID: time.Now().Format("20060102-150405")}
	dir := filepath.Join(QuarantineRoot(), out.BatchID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		out.Failed = len(entries)
		return out
	}

	var items []BatchItem
	for i, e := range entries {
		stored := fmt.Sprintf("%04d-%s", i, filepath.Base(e.Path))
		if err := os.Rename(e.Path, filepath.Join(dir, stored)); err != nil {
			out.Failed++
			continue
		}
		items = append(items, BatchItem{From: e.Path, Stored: stored, Size: e.Size})
		out.Moved++
		out.Bytes += e.Size
		out.Paths = append(out.Paths, e.Path)
	}

	if out.Moved == 0 {
		os.Remove(dir)
		return out
	}
	manifest, _ := json.MarshalIndent(Batch{ID: out.BatchID, Created: time.Now(), Items: items}, "", "  ")
	os.WriteFile(filepath.Join(dir, "manifest.json"), manifest, 0o644)
	return out
}

// ListBatches returns all quarantine batches, oldest first.
func ListBatches() []Batch {
	root := QuarantineRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var batches []Batch
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, e.Name(), "manifest.json"))
		if err != nil {
			continue
		}
		var b Batch
		if json.Unmarshal(data, &b) == nil {
			b.ID = e.Name()
			batches = append(batches, b)
		}
	}
	sort.Slice(batches, func(i, j int) bool { return batches[i].Created.Before(batches[j].Created) })
	return batches
}

// RestoreBatch moves every item of a batch back to its original location.
// Existing destinations are skipped (never overwritten). Empty batches are removed.
func RestoreBatch(id string) (restored, skipped, failed int, err error) {
	batches := ListBatches()
	if len(batches) == 0 {
		return 0, 0, 0, fmt.Errorf("quarantine is empty")
	}
	var batch *Batch
	for i := range batches {
		if batches[i].ID == id {
			batch = &batches[i]
		}
	}
	if batch == nil {
		return 0, 0, 0, fmt.Errorf("no such batch %q (run 'macclean undo' to list batches)", id)
	}

	dir := filepath.Join(QuarantineRoot(), batch.ID)
	remaining := []BatchItem{}
	for _, it := range batch.Items {
		if pathExists(it.From) {
			skipped++
			remaining = append(remaining, it)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(it.From), 0o755); err != nil {
			failed++
			remaining = append(remaining, it)
			continue
		}
		if err := os.Rename(filepath.Join(dir, it.Stored), it.From); err != nil {
			failed++
			remaining = append(remaining, it)
			continue
		}
		restored++
	}
	if len(remaining) > 0 {
		manifest, _ := json.MarshalIndent(*batch, "", "  ")
		os.WriteFile(filepath.Join(dir, "manifest.json"), manifest, 0o644)
	} else {
		os.RemoveAll(dir)
	}
	return restored, skipped, failed, nil
}

// PurgeOld permanently deletes batches older than the given number of days.
func PurgeOld(days int) (purged int, freed int64) {
	cutoff := time.Now().AddDate(0, 0, -days)
	for _, b := range ListBatches() {
		if b.Created.Before(cutoff) {
			os.RemoveAll(filepath.Join(QuarantineRoot(), b.ID))
			purged++
			freed += b.Size()
		}
	}
	return purged, freed
}
