package cmd

import (
	"errors"
	"fmt"
	"os"
	"runtime"

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
saved to your config file.

It runs in the background, so closing the terminal does not stop it.
Use --foreground to keep it attached to the terminal instead.`,
	Run: func(cmd *cobra.Command, args []string) {
		foreground, _ := cmd.Flags().GetBool("foreground")
		if runtime.GOOS == "darwin" && !foreground && !startedByLaunchd() {
			pid, err := detach("menubar", "--foreground")
			if err != nil {
				die(err)
			}
			fmt.Printf("menu bar timer running in the background (pid %d)\n", pid)
			return
		}

		if err := menubar.Run(config.C); err != nil {
			die(err)
		}
	},
}

// startedByLaunchd reports whether launchd (pid 1) is the parent, as for the
// login agent. launchd must keep supervising the process to restart it after
// a crash, so it stays in the foreground even with older agent plists that
// lack --foreground.
func startedByLaunchd() bool {
	return os.Getppid() == 1
}

var menubarLoginCmd = &cobra.Command{
	Use:       "login [on|off]",
	Args:      cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
	ValidArgs: []string{"on", "off"},
	Short:     "Start the menu bar timer automatically at login",
	Long: `Start the menu bar timer automatically at login.

Without arguments, prints whether it is enabled. "on" registers a
launchd agent for this pomo binary, which takes effect at the next
login. "off" removes it.`,
	Example: `  pomo menubar login      # show status
  pomo menubar login on   # start at login
  pomo menubar login off  # stop starting at login`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runMenubarLogin(args); err != nil {
			die(err)
		}
	},
}

func runMenubarLogin(args []string) error {
	if runtime.GOOS != "darwin" {
		return errors.New("login items are only available on macOS")
	}

	login, err := menubar.DefaultLoginItem()
	if err != nil {
		return err
	}

	if len(args) == 1 {
		if args[0] == "on" {
			err = login.Enable()
		} else {
			err = login.Disable()
		}
		if err != nil {
			return err
		}
	}

	if login.Enabled() {
		fmt.Printf("menu bar starts at login: %s\n", login.Executable)
	} else {
		fmt.Println("menu bar does not start at login")
	}

	return nil
}

func init() {
	menubarCmd.Flags().BoolP("foreground", "f", false, "stay attached to the terminal instead of running in the background")
	menubarCmd.AddCommand(menubarLoginCmd)
	rootCmd.AddCommand(menubarCmd)
}
