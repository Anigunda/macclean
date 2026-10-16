package cleaner

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// MaxHistoryEvents caps the journal; oldest entries are trimmed on write.
const MaxHistoryEvents = 500

// Event is one journal entry describing a clean/quarantine run.
type Event struct {
	Time    time.Time `json:"ts"`
	Mode    string    `json:"mode"` // "clean", "tui", "quarantine", "tui-quarantine"
	Freed   int64     `json:"freed"`
	Removed int64     `json:"removed"`
	Failed  int64     `json:"failed"`
	Paths   []string  `json:"paths,omitempty"`
}

// StateDir is where macclean keeps its own data: history, logs.
func StateDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "macclean")
}

func historyFile() string {
	return filepath.Join(StateDir(), "history.jsonl")
}

// AppendHistory records an event, trimming the journal to MaxHistoryEvents.
func AppendHistory(e Event) error {
	events, err := ReadHistory(0) // all
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	events = append(events, e)
	if len(events) > MaxHistoryEvents {
		events = events[len(events)-MaxHistoryEvents:]
	}

	path := historyFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, ev := range events {
		if err := enc.Encode(ev); err != nil {
			return err
		}
	}
	return nil
}

// ReadHistory reads events oldest-first; limit>0 returns only the last N.
func ReadHistory(limit int) ([]Event, error) {
	f, err := os.Open(historyFile())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var e Event
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			events = append(events, e)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}
	return events, nil
}
