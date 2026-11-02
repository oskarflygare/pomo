package cmd

import (
	"github.com/Bahaaio/pomo/config"
	"github.com/Bahaaio/pomo/menubar"
	"github.com/spf13/cobra"
)

var menubarCmd = &cobra.Command{
	Use:   "menubar",
	Args:  cobra.NoArgs,
	Short: "Run the timer in the macOS menu bar",
	Long: `Show a miniature progress bar timer in the macOS menu bar.

Right-click the bar to start or pause the session. Click it to open
a menu with today's stats and the work and break lengths, which are
saved to your config file.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := menubar.Run(config.C); err != nil {
			die(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(menubarCmd)
}
