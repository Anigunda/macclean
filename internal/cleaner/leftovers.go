package cleaner

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// leftoverLocations are user-level folders where uninstalled apps leave data.
var leftoverLocations = []string{
	"Application Support",
	"Preferences",
	"Containers",
	"Group Containers",
	"HTTPStorages",
	"Saved Application State",
}

// applePrefixes are never flagged: apple. and com.apple. are obvious;
// is.workflow. is Apple's Shortcuts legacy identifier (pre-rebranding).
var applePrefixes = []string{
	"com.apple.", "apple.", "is.workflow.", "group.is.workflow.",
}

// genericTokens carry no information about which app an entry belongs to.
var genericTokens = map[string]bool{
	"com": true, "org": true, "net": true, "io": true, "co": true,
	"app": true, "inc": true, "ltd": true, "llc": true, "corp": true,
	"desktop": true, "helper": true, "mac": true, "osx": true,
	"macos": true, "group": true,
}

// FindLeftovers returns paths that likely belong to apps no longer installed.
// Matching is deliberately conservative: Apple entries and anything sharing a
// name token with an installed app are never flagged, so report - don't trust
// blindly - applies to the user reviewing the results.
func FindLeftovers() []string {
	installed := installedAppTokens()
	home, _ := os.UserHomeDir()
	lib := filepath.Join(home, "Library")

	var found []string
	for _, loc := range leftoverLocations {
		dir := filepath.Join(lib, loc)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // missing or permission-restricted
		}
		for _, e := range entries {
			name := strings.TrimSuffix(e.Name(), ".plist")
			if isOrphanCandidate(name, installed) {
				found = append(found, filepath.Join(dir, e.Name()))
			}
		}
	}
	return found
}

func isOrphanCandidate(name string, installed map[string]bool) bool {
	lower := strings.ToLower(name)

	// Never flag Apple system data.
	for _, prefix := range applePrefixes {
		if strings.HasPrefix(lower, prefix) {
			return false
		}
	}
	// Only judge reverse-DNS-style names (com.vendor.product[...]). Plain-word
	// folders ("Slack", "Adobe") are too often shared or system data.
	if strings.Count(lower, ".") < 2 {
		return false
	}
	for _, tok := range tokenize(name) {
		if genericTokens[tok] || len(tok) < 3 {
			continue
		}
		for itok := range installed {
			// Exact match, or containment for tokens of 4+ chars
			// ("slackmacgap" ~ installed app "Slack").
			if tok == itok {
				return false
			}
			if len(tok) >= 4 && len(itok) >= 4 &&
				(strings.Contains(tok, itok) || strings.Contains(itok, tok)) {
				return false
			}
		}
	}
	return true
}

// installedAppTokens collects lowercase tokens from every installed .app bundle name.
func installedAppTokens() map[string]bool {
	home, _ := os.UserHomeDir()
	appDirs := []string{
		"/Applications",
		"/System/Applications",
		"/System/Applications/Utilities",
		filepath.Join(home, "Applications"),
	}
	tokens := map[string]bool{"apple": true}
	for _, dir := range appDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".app") {
				continue
			}
			for _, tok := range tokenize(strings.TrimSuffix(e.Name(), ".app")) {
				tokens[tok] = true
			}
		}
	}
	return tokens
}

// tokenize splits a name on non-alphanumeric separators into lowercase tokens.
func tokenize(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.ToLower(p))
	}
	return out
}
