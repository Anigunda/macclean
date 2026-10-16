package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Anigunda/macclean/internal/cleaner"
)

const launchAgentLabel = "com.macclean.weekly"

func launchAgentPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist")
}

func stateLogPath() string {
	return filepath.Join(cleaner.StateDir(), "scheduler.log")
}

var scheduleCmd = &cobra.Command{
	Use:       "schedule [on|off]",
	Short:     "Manage a weekly automatic clean (launchd agent)",
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"on", "off"},
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return scheduleStatus()
		}
		switch args[0] {
		case "on":
			return scheduleOn()
		case "off":
			return scheduleOff()
		default:
			return fmt.Errorf("expected 'on' or 'off'")
		}
	},
}

func scheduleStatus() error {
	path := launchAgentPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("Scheduler is OFF. Enable with: macclean schedule on")
		return nil
	}
	fmt.Printf("Scheduler is ON\n  plist: %s\n  runs:  every Monday 09:00 ('macclean clean --yes')\n  log:   %s\n",
		path, stateLogPath())
	return nil
}

func scheduleOn() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.EvalSymlinks(exe)
	log := stateLogPath()
	if err := os.MkdirAll(filepath.Dir(log), 0o755); err != nil {
		return err
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>clean</string>
		<string>--yes</string>
	</array>
	<key>StartCalendarInterval</key>
	<dict>
		<key>Weekday</key><integer>1</integer>
		<key>Hour</key><integer>9</integer>
		<key>Minute</key><integer>0</integer>
	</dict>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
</dict>
</plist>
`, launchAgentLabel, exe, log, log)

	path := launchAgentPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		return err
	}
	// Reload in case an older agent is already registered.
	_ = exec.Command("launchctl", "unload", path).Run()
	if out, err := exec.Command("launchctl", "load", path).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load failed: %v (%s)", err, out)
	}
	fmt.Printf("Scheduler ON - 'macclean clean --yes' runs every Monday 09:00.\n  plist: %s\n  log:   %s\n", path, log)
	fmt.Println("\nTip: it respects your keep rules and only deletes suggested-deletes.")
	return nil
}

func scheduleOff() error {
	path := launchAgentPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("Scheduler is already OFF.")
		return nil
	}
	_ = exec.Command("launchctl", "unload", path).Run()
	if err := os.Remove(path); err != nil {
		return err
	}
	fmt.Println("Scheduler OFF - launch agent removed.")
	return nil
}
