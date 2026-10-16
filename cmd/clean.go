package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Anigunda/macclean/internal/cleaner"
)

var (
	dryRun        bool
	assumeYes     bool
	only          []string
	cleanAll      bool
	quarantineOff bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove the junk found by 'macclean scan' (asks before deleting)",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		targets := cleaner.Targets()
		if len(only) > 0 {
			targets = filterTargets(targets, only)
			if len(targets) == 0 {
				return fmt.Errorf("no targets matched --only (available: %s)", strings.Join(cleaner.TargetIDs(), ", "))
			}
		}

		fmt.Fprintln(os.Stderr, "Scanning...")
		entries := cleaner.CollectItems(targets)
		if len(entries) == 0 {
			fmt.Println("Nothing to clean - your Mac is already tidy.")
			return nil
		}

		// Summary by category.
		byCat := map[string]int64{}
		var order []string
		var total int64
		for _, e := range entries {
			if _, seen := byCat[e.TargetName]; !seen {
				order = append(order, e.TargetName)
			}
			byCat[e.TargetName] += e.Size
			total += e.Size
		}
		fmt.Println("The following will be cleaned:")
		for _, name := range order {
			fmt.Printf("  %-18s %s\n", name, cleaner.HumanSize(byCat[name]))
		}
		fmt.Printf("\nTotal: %s in %d items\n", cleaner.HumanSize(total), len(entries))

		if dryRun {
			fmt.Println("\nDry run: nothing was deleted.")
			return nil
		}
		if !assumeYes && !confirm("\nProceed?") {
			fmt.Println("Aborted - nothing was deleted.")
			return nil
		}

		// Partition: suggested-keeps are skipped unless --all.
		var deletables, kept []cleaner.ItemEntry
		for _, e := range entries {
			if e.Keep && !cleanAll {
				kept = append(kept, e)
			} else {
				deletables = append(deletables, e)
			}
		}

		mode := "clean"
		var freed, removed, failed int64
		var paths []string
		if quarantineOff {
			mode = "quarantine"
			out := cleaner.QuarantineEntries(deletables)
			freed, removed, failed = out.Bytes, out.Moved, int64(out.Failed)
			paths = out.Paths
			fmt.Printf("quarantined %d items, %s\n", removed, cleaner.HumanSize(freed))
			if removed > 0 {
				fmt.Printf("  -> %s/%s\n", cleaner.QuarantineRoot(), out.BatchID)
				fmt.Printf("  restore with 'macclean undo' - auto-purges after %d days\n",
					cleaner.GetConfig().QuarantineDaysOrDefault())
			}
		} else {
			res := cleaner.CleanEntries(deletables)
			freed, removed, failed = res.Freed, res.Removed, res.Failed
			paths = res.Paths
			fmt.Printf("deleted %d items, freed %s\n", removed, cleaner.HumanSize(freed))
		}
		if failed > 0 {
			fmt.Printf("(%d locked/in-use items skipped)\n", failed)
		}
		for _, k := range kept {
			fmt.Printf("  kept %s (%s)\n", k.Path, k.Reason)
		}
		if removed > 0 || failed > 0 {
			_ = cleaner.AppendHistory(cleaner.Event{
				Time: time.Now(), Mode: mode,
				Freed: freed, Removed: removed, Failed: failed, Paths: paths,
			})
		}
		if len(kept) > 0 && !quarantineOff {
			fmt.Println("\nRun 'macclean clean --all' to also delete the kept items above.")
		}
		fmt.Println("\nDone.")
		return nil
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be deleted without deleting anything")
	cleanCmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "skip the confirmation prompt")
	cleanCmd.Flags().StringSliceVar(&only, "only", nil, "clean only these categories (comma-separated ids, see 'macclean scan')")
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "also delete items suggested to keep (iCloud caches, AI models, ...)")
	cleanCmd.Flags().BoolVarP(&quarantineOff, "quarantine", "q", false, "move items to quarantine instead of deleting (restore with 'macclean undo')")
}

func filterTargets(targets []cleaner.Target, ids []string) []cleaner.Target {
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[strings.TrimSpace(id)] = true
	}
	out := make([]cleaner.Target, 0, len(targets))
	for _, t := range targets {
		if want[t.ID] {
			out = append(out, t)
		}
	}
	return out
}

func confirm(question string) bool {
	fmt.Printf("%s [y/N]: ", question)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}
