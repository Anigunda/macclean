# macclean

**Know what you're deleting.** macclean is a small Go CLI that scans your Mac for junk — caches, logs, temp files, trash, Xcode leftovers, and files from apps you already uninstalled — then cleans it **without guesswork**: every item gets a keep/delete recommendation with a reason, and anything you remove can go to a restorable quarantine instead of the void.

<video src="docs/launch.mp4" controls="controls" muted="muted" playsinline style="max-width: 100%;"></video>

[Can't see the video? Download it here (46s, 4.2 MB).](docs/launch.mp4)

## Why

- **Read-only analysis first.** `macclean scan` shows exactly what's taking space before anything is touched.
- **Judgment built in.** Every item is tagged `delete` or `keep` *with a reason* — iCloud Drive caches, Spotlight indexes and AI model downloads are kept by default (deleting them costs you hours of re-syncing).
- **Undo for deletes.** `macclean clean --quarantine` moves instead of deletes; `macclean undo` restores. Batches auto-purge after 7 days.
- **Interactive TUI.** Checkbox table, keep/delete tags, WHY column, live totals — you approve every file.

## Install

**Option 1 — download from this repo (no Go needed)** — binaries live in [`release/`](release/):

```bash
# Apple Silicon
curl -L -o macclean https://raw.githubusercontent.com/anilgoudgunda/macclean/main/release/macclean-darwin-arm64
# Intel Mac? use macclean-darwin-amd64 instead

chmod +x macclean && sudo mv macclean /usr/local/bin/
macclean scan
```

(Also available on the [Releases page](https://github.com/Anigunda/macclean/releases) when tagged — same files.)

**Option 2 — with Go** (Go >= 1.26):

```bash
go install github.com/Anigunda/macclean@latest         # lands in ~/go/bin

# or from source
git clone https://github.com/Anigunda/macclean && cd macclean && go install .
```

Binaries are static (no dependencies) — for macOS only: Apple Silicon and Intel.

## Use it

```bash
macclean               # interactive UI: review items, select, clean or quarantine
macclean scan          # read-only system analysis (add --detail for per-item tags)
macclean clean         # clean with confirmation (respects keep advice)
macclean undo          # restore quarantined items (never overwrites)
```

Handy flags for `clean`:

```bash
--dry-run            # preview, delete nothing
--only=temp,trash    # limit to specific categories (ids shown by 'macclean scan')
--all                # also delete items suggested to keep
--quarantine, -q     # move to quarantine instead of deleting
```

Config & automation:

```bash
macclean config --init   # ~/.config/macclean/config.yaml — your keep/delete rules + custom categories
macclean schedule on     # weekly auto-clean (launchd), respects your keep rules
macclean history         # journal of every clean: what, when, bytes freed
```

## What it cleans (user-level, no sudo)

| Category              | What                                                                         |
| --------------------- | ---------------------------------------------------------------------------- |
| User Caches           | `~/Library/Caches` — rebuilt automatically by apps                            |
| User Logs             | `~/Library/Logs`                                                             |
| Temporary Files       | your per-session `$TMPDIR`                                                    |
| Trash                 | `~/.Trash`                                                                   |
| npm / Gradle / Maven  | package caches (re-downloaded on demand)                                      |
| Stale node_modules    | projects untouched 30+ days — `npm ci` restores them                          |
| Xcode Junk            | DerivedData + iOS DeviceSupport (only when Xcode exists)                      |
| App Leftovers         | Application Support / Preferences / Containers with **no installed app** — never flags Apple items |
| iOS Backups           | device backups — kept by default, review before deleting                      |

## Safety by design

- Scan is 100% read-only; `clean` always asks unless you pass `-y`.
- Only the *contents* of cache/temp folders are removed, never the folders themselves.
- Locked/in-use files are skipped, not fatal.
- App leftovers conservative-match: `com.apple.*` never matches, and anything sharing a name token with an installed app is excluded.
- Quarantine restores never overwrite existing paths.
- A journal of every clean lives at `~/.local/state/macclean/history.jsonl`.

## Tech

Pure Go 1.26 · [Cobra] commands · [Bubble Tea] + [Lipgloss] TUI · YAML config · launchd scheduler. No daemons, no accounts, no telemetry.

[Cobra]: https://github.com/spf13/cobra
[Bubble Tea]: https://github.com/charmbracelet/bubbletea
[Lipgloss]: https://github.com/charmbracelet/lipgloss

## License

MIT — see [LICENSE](LICENSE).
