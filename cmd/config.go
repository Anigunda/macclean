package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Anigunda/macclean/internal/cleaner"
)

var configInit bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or initialize the macclean config file",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		path := cleaner.ConfigPath()
		if configInit {
			p, err := cleaner.WriteSampleConfig()
			if err != nil {
				return err
			}
			fmt.Printf("Wrote %s - edit it, then re-run macclean.\n", p)
			return nil
		}

		cfg := cleaner.GetConfig()
		fmt.Printf("Config file: %s\n\n", path)
		fmt.Printf("keep rules:        %v\n", noneIfEmpty(cfg.Keep))
		fmt.Printf("delete rules:      %v\n", noneIfEmpty(cfg.Delete))
		fmt.Printf("quarantine days:   %d\n", cfg.QuarantineDaysOrDefault())
		fmt.Printf("stale days:        %d\n", cfg.StaleDaysOrDefault())
		fmt.Printf("extra targets:     %d\n", len(cfg.ExtraTargets))
		fmt.Println("\nRun 'macclean config --init' to create a commented sample config.")
		return nil
	},
}

func init() {
	configCmd.Flags().BoolVar(&configInit, "init", false, "write a commented sample config (refuses to overwrite)")
}

func noneIfEmpty(s []string) string {
	if len(s) == 0 {
		return "(none)"
	}
	return fmt.Sprintf("%v", s)
}
