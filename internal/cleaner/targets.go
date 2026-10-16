package cleaner

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Target is a category of junk that can be scanned and cleaned.
// By default only the *contents* of Paths are removed, never the directories
// themselves. If RemoveWhole is true (used for app leftovers), each path is
// deleted entirely.
type Target struct {
	ID          string
	Name        string
	Description string
	Paths       []string
	RemoveWhole bool
}

// Targets returns the junk categories for this machine (Phase 1: user-level only, no sudo).
func Targets() []Target {
	home, _ := os.UserHomeDir()
	lib := filepath.Join(home, "Library")

	targets := []Target{
		{
			ID:          "user-caches",
			Name:        "User Caches",
			Description: "App cache files in ~/Library/Caches. Safe: apps rebuild them.",
			Paths:       []string{filepath.Join(lib, "Caches")},
		},
		{
			ID:          "user-logs",
			Name:        "User Logs",
			Description: "Log files in ~/Library/Logs.",
			Paths:       []string{filepath.Join(lib, "Logs")},
		},
		{
			ID:          "temp",
			Name:        "Temporary Files",
			Description: "Files in your per-user temp directory ($TMPDIR).",
			Paths:       []string{os.TempDir()},
		},
		{
			ID:          "trash",
			Name:        "Trash",
			Description: "Items sitting in ~/.Trash.",
			Paths:       []string{filepath.Join(home, ".Trash")},
		},
	}

	xcode := filepath.Join(lib, "Developer", "Xcode")
	if dirExists(xcode) {
		targets = append(targets, Target{
			ID:          "xcode-junk",
			Name:        "Xcode Junk",
			Description: "DerivedData and iOS DeviceSupport left behind by Xcode builds.",
			Paths: []string{
				filepath.Join(xcode, "DerivedData"),
				filepath.Join(xcode, "iOS DeviceSupport"),
			},
		})
	}

	// Developer ecosystem caches (only added when present).
	if d := filepath.Join(home, ".npm"); dirExists(d) {
		targets = append(targets, Target{
			ID: "npm-cache", Name: "npm Cache",
			Description: "Downloaded npm packages. Re-downloaded on demand.",
			Paths:       []string{d},
		})
	}
	if d := filepath.Join(home, ".gradle", "caches"); dirExists(d) {
		targets = append(targets, Target{
			ID: "gradle-cache", Name: "Gradle Cache",
			Description: "Gradle dependency cache. Re-downloaded on the next build.",
			Paths:       []string{d},
		})
	}
	if d := filepath.Join(home, ".m2", "repository"); dirExists(d) {
		targets = append(targets, Target{
			ID: "maven-cache", Name: "Maven Cache",
			Description: "Maven dependency cache. Re-downloaded on the next build.",
			Paths:       []string{d},
		})
	}

	// Stale node_modules (dependencies of projects untouched for config.stale_days).
	if dirs := staleNodeModules(GetConfig().StaleDaysOrDefault()); len(dirs) > 0 {
		targets = append(targets, Target{
			ID:          "stale-node-modules",
			Name:        "Stale node_modules",
			Description: "node_modules of projects untouched for a while. 'npm ci' restores them.",
			Paths:       dirs,
			RemoveWhole: true,
		})
	}

	// iOS/iPadOS device backups (huge, but user data: default suggestion is KEEP).
	if d := filepath.Join(lib, "Application Support", "MobileSync", "Backup"); dirExists(d) {
		targets = append(targets, Target{
			ID: "ios-backups", Name: "iOS Backups",
			Description: "Full device backups. Kept by default - review before deleting.",
			Paths:       []string{d},
		})
	}

	// User-defined categories from the config file.
	for _, et := range GetConfig().ExtraTargets {
		var paths []string
		for _, p := range et.Paths {
			expanded := expandPath(p)
			if dirExists(expanded) {
				paths = append(paths, expanded)
			}
		}
		if len(paths) > 0 && et.ID != "" {
			name := et.Name
			if name == "" {
				name = et.ID
			}
			targets = append(targets, Target{
				ID:          et.ID,
				Name:        name,
				Description: "Custom category from your macclean config.",
				Paths:       paths,
			})
		}
	}

	if leftovers := FindLeftovers(); len(leftovers) > 0 {
		targets = append(targets, Target{
			ID:          "app-leftovers",
			Name:        "App Leftovers",
			Description: "Files from apps that no longer exist on this Mac. Review with 'macclean scan --detail' before cleaning.",
			Paths:       leftovers,
			RemoveWhole: true,
		})
	}

	return targets
}

// TargetIDs lists the IDs of all targets on this machine (for --only flag help).
func TargetIDs() []string {
	ids := make([]string, 0, len(Targets()))
	for _, t := range Targets() {
		ids = append(ids, t.ID)
	}
	return ids
}

// expandPath expands a leading ~ to the user's home directory.
func expandPath(p string) string {
	if p == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

// staleNodeModules finds node_modules directories (1-2 levels under common
// project roots) whose directory mtime is older than the given number of days.
func staleNodeModules(days int) []string {
	home, _ := os.UserHomeDir()
	roots := []string{"Code", "Projects", "Developer", "dev", "work", "src", "repos"}
	cutoff := time.Now().AddDate(0, 0, -days)

	seen := map[string]bool{}
	var out []string
	for _, r := range roots {
		root := filepath.Join(home, r)
		if !dirExists(root) {
			continue
		}
		for _, pattern := range []string{"*/node_modules", "*/*/node_modules"} {
			matches, _ := filepath.Glob(filepath.Join(root, pattern))
			for _, m := range matches {
				if seen[m] {
					continue
				}
				if st, err := os.Stat(m); err == nil && st.IsDir() && st.ModTime().Before(cutoff) {
					seen[m] = true
					out = append(out, m)
				}
			}
		}
	}
	return out
}
