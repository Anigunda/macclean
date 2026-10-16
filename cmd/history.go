package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/anilgoudgunda/macclean/internal/cleaner"
)

var (
	historyLimit   int
	historyVerbose bool
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show the journal of past cleans (what was removed, how much was freed)",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		events, err := cleaner.ReadHistory(historyLimit)
		if err != nil {
			fmt.Println("No history yet - run 'macclean clean' first.")
			return nil
		}
		if len(events) == 0 {
			fmt.Println("No history yet - run 'macclean clean' first.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "DATE\tFREED\tITEMS\tFAILED\tMODE")
		var totalFreed int64
		for i := len(events) - 1; i >= 0; i-- { // newest first
			e := events[i]
			totalFreed += e.Freed
			fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n",
				e.Time.Format("2006-01-02 15:04"), cleaner.HumanSize(e.Freed),
				e.Removed, e.Failed, e.Mode)
		}
		_ = w.Flush()
		fmt.Printf("\nTotal freed across %d events: %s\n", len(events), cleaner.HumanSize(totalFreed))

		if historyVerbose {
			last := events[len(events)-1]
			fmt.Printf("\nLast event (%s, %s):\n", last.Time.Format("2006-01-02 15:04"), last.Mode)
			for _, p := range last.Paths {
				fmt.Println("  " + cleaner.ShortPath(p))
			}
		}
		return nil
	},
}

func init() {
	historyCmd.Flags().IntVarP(&historyLimit, "limit", "n", 20, "max events to show")
	historyCmd.Flags().BoolVarP(&historyVerbose, "verbose", "v", false, "list the paths removed in the most recent event")
}
