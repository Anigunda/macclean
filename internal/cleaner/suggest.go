package cleaner

import "strings"

type keepRule struct {
	pattern string // lowercase substring matched against the item's base name
	reason  string
}

// keepRules are checked against entries of ~/Library/Caches. Deleting any of
// these is safe but costly (slow re-syncs, hours of reindexing, huge downloads),
// so the default recommendation is to keep them.
var keepRules = []keepRule{
	{"com.apple.bird", "iCloud Drive cache - deleting triggers a full re-download"},
	{"cloudkit", "iCloud sync state - let macOS manage it"},
	{"icloud", "iCloud cache - deleting causes a re-sync"},
	{"spotlight", "Spotlight index data - rebuilding takes hours"},
	{"ollama", "may hold AI model data - re-downloading is slow"},
	{"huggingface", "model weights - huge to re-download"},
}

// Suggest reports whether an item is recommended to keep (true) or delete
// (false), with a short reason. User config rules win over built-in heuristics;
// nothing here decides - it advises; --all overrides.
func Suggest(targetID, baseName string) (keep bool, reason string) {
	cfg := GetConfig()
	if cfg.MatchesKeep(baseName) {
		return true, "matches a keep rule in your config"
	}
	if cfg.MatchesDelete(baseName) {
		return false, "matches a delete rule in your config"
	}

	if targetID == "user-caches" {
		lower := strings.ToLower(baseName)
		for _, r := range keepRules {
			if strings.Contains(lower, r.pattern) {
				return true, r.reason
			}
		}
		return false, "cache - apps rebuild it automatically"
	}

	switch targetID {
	case "user-logs":
		return false, "old logs - safe to remove"
	case "temp":
		return false, "temp file - safe to remove"
	case "trash":
		return false, "you already put this in the Trash"
	case "xcode-junk":
		return false, "build data - Xcode regenerates it on next build"
	case "app-leftovers":
		return false, "no installed app matches - review, then remove"
	case "npm-cache":
		return false, "npm re-downloads packages on demand"
	case "gradle-cache":
		return false, "Gradle re-downloads dependencies on the next build"
	case "maven-cache":
		return false, "Maven re-downloads dependencies on the next build"
	case "stale-node-modules":
		return false, "untouched for weeks - 'npm ci' restores it"
	case "ios-backups":
		return true, "full device backup - delete only if outdated or backed up to iCloud"
	}
	return false, "safe to remove"
}
