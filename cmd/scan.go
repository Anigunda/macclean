package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"macclean/internal/cleaner"
)

var (
	scanDetail bool
	scanTop    int
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Analyze your system and show how much space can be reclaimed",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		fmt.Fprintln(os.Stderr, "Scanning your Mac...")
		results := cleaner.Scan(cleaner.Targets())

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "CATEGORY\tSIZE\tFILES\tID")
		var total int64
		for _, r := range results {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", r.Target.Name, cleaner.HumanSize(r.Size), r.Files, r.Target.ID)
			total += r.Size
		}
		fmt.Fprintf(w, "TOTAL\t%s\t\t\n", cleaner.HumanSize(total))
		_ = w.Flush()

		if scanDetail {
			for _, r := range results {
				if r.Size == 0 {
					continue
				}
				fmt.Printf("\n%s - %s\n", r.Target.Name, cleaner.HumanSize(r.Size))
				var items []cleaner.Item
				if r.Target.RemoveWhole {
					items = cleaner.ItemsFromPaths(r, scanTop)
				} else {
					items = cleaner.TopItems(r, scanTop)
				}
				var shown int64
				for _, it := range items {
					shown += it.Size
					keep, reason := cleaner.Suggest(r.Target.ID, filepath.Base(it.Path))
					tag := "delete"
					if keep {
						tag = "keep  "
					}
					fmt.Printf("  %9s  [%s] %s (%d files)\n          %s\n",
						cleaner.HumanSize(it.Size), tag, cleaner.ShortPath(it.Path), it.Files, reason)
				}
				if rest := r.Size - shown; rest > 0 {
					fmt.Printf("  %9s  everything else\n", cleaner.HumanSize(rest))
				}
			}
		} else {
			fmt.Println("\nTip: run 'macclean scan --detail' to see what is inside each category.")
		}

		fmt.Println("\nRun 'macclean clean' to free this space.")
		return nil
	},
}

func init() {
	scanCmd.Flags().BoolVarP(&scanDetail, "detail", "d", false, "show the largest items inside each category")
	scanCmd.Flags().IntVar(&scanTop, "top", 10, "number of largest items shown per category (with --detail)")
}
