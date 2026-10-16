package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Anigunda/macclean/internal/cleaner"
	"github.com/Anigunda/macclean/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:           "macclean",
	Short:         "Keep your Mac clean",
	Long:          "macclean scans your Mac for junk (caches, logs, temp files, trash, Xcode leftovers,\nfiles left behind by uninstalled apps) and removes it safely.\n\nRun 'macclean' with no arguments (or 'macclean ui') for the interactive UI.\nOtherwise: 'macclean scan' to see the analysis, then 'macclean clean' to free the space.",
	SilenceUsage:  true,
	SilenceErrors: true,
	// Load config and maintenance before any subcommand runs.
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		if err := cleaner.LoadConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
		// Housekeeping: auto-purge expired quarantine batches (silent).
		cleaner.PurgeOld(cleaner.GetConfig().QuarantineDaysOrDefault())
		return nil
	},
	// Bare 'macclean' opens the interactive UI; in a pipe/script show help.
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			return cmd.Help()
		}
		return tui.Run(cleaner.Targets())
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(scanCmd, cleanCmd, uiCmd, configCmd, historyCmd, undoCmd, scheduleCmd)
}
