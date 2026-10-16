package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Anigunda/macclean/internal/cleaner"
)

var undoAll bool

var undoCmd = &cobra.Command{
	Use:   "undo [batch-id]",
	Short: "Restore quarantined items to their original locations",
	Long:  "With no arguments, lists quarantine batches.\nWith a batch id (or --all), restores those items - never overwriting existing paths.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		batches := cleaner.ListBatches()
		if len(batches) == 0 {
			fmt.Println("Quarantine is empty.")
			return nil
		}

		if len(args) == 0 && !undoAll {
			fmt.Printf("Quarantine batches (auto-purged after %d days):\n\n", cleaner.GetConfig().QuarantineDaysOrDefault())
			for _, b := range batches {
				fmt.Printf("  %s   %d items, %s   (quarantined %s)\n",
					b.ID, len(b.Items), cleaner.HumanSize(b.Size()),
					b.Created.Format("2006-01-02 15:04"))
			}
			fmt.Println("\nRestore with: macclean undo <batch-id>   (or --all)")
			return nil
		}

		var toRestore []string
		if undoAll {
			for _, b := range batches {
				toRestore = append(toRestore, b.ID)
			}
		} else {
			toRestore = []string{args[0]}
		}
		for _, id := range toRestore {
			restored, skipped, failed, err := cleaner.RestoreBatch(id)
			if err != nil {
				return err
			}
			fmt.Printf("batch %s: restored %d, skipped %d (already exist), failed %d\n",
				id, restored, skipped, failed)
		}
		return nil
	},
}

func init() {
	undoCmd.Flags().BoolVar(&undoAll, "all", false, "restore every quarantine batch")
}
