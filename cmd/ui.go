package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/anilgoudgunda/macclean/internal/cleaner"
	"github.com/anilgoudgunda/macclean/internal/tui"
)

var uiOnly []string

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the interactive terminal UI (review items, select, clean)",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			return errors.New("the TUI needs an interactive terminal - use 'macclean scan --detail' and 'macclean clean' instead")
		}
		targets := cleaner.Targets()
		if len(uiOnly) > 0 {
			targets = filterTargets(targets, uiOnly)
			if len(targets) == 0 {
				return errors.New("no targets matched --only")
			}
		}
		return tui.Run(targets)
	},
}

func init() {
	uiCmd.Flags().StringSliceVar(&uiOnly, "only", nil, "show only these categories (comma-separated ids)")
}
