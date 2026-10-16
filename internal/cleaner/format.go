package cleaner

import (
	"fmt"
	"os"
	"strings"
)

// HumanSize formats bytes as a human-readable string (e.g. 1.4 GB).
func HumanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// ShortPath replaces the home directory prefix with ~.
func ShortPath(p string) string {
	home, err := os.UserHomeDir()
	if err == nil {
		if p == home {
			return "~"
		}
		if strings.HasPrefix(p, home+string(os.PathSeparator)) {
			return "~" + p[len(home):]
		}
	}
	return p
}
