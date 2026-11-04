package menubar

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// LoginLabel identifies the launchd agent that starts the menu bar at login.
const LoginLabel = "com.github.bahaaio.pomo.menubar"

// LoginItem manages a per-user launchd agent that runs `pomo menubar` at login.
type LoginItem struct {
	PlistPath  string // ~/Library/LaunchAgents/<label>.plist
	Executable string // absolute path to the pomo binary
	Home       string // working directory, so the per-user config is used
	PathEnv    string // PATH for post-session commands
}

// DefaultLoginItem returns the login item for the running pomo binary.
func DefaultLoginItem() (LoginItem, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return LoginItem{}, err
	}

	executable, err := os.Executable()
	if err != nil {
		return LoginItem{}, err
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}

	return LoginItem{
		PlistPath:  filepath.Join(home, "Library", "LaunchAgents", LoginLabel+".plist"),
		Executable: executable,
		Home:       home,
		PathEnv:    os.Getenv("PATH"),
	}, nil
}

// Enabled reports whether the login item is installed.
func (l LoginItem) Enabled() bool {
	_, err := os.Stat(l.PlistPath)
	return err == nil
}

// Enable installs the login item. It takes effect at the next login.
func (l LoginItem) Enable() error {
	// binaries from `go run` live in a temporary build directory
	if strings.Contains(l.Executable, string(filepath.Separator)+"go-build") {
		return fmt.Errorf("refusing to register temporary binary %s; install pomo first", l.Executable)
	}

	plist, err := l.plist()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(l.PlistPath), 0o755); err != nil {
		return err
	}

	return os.WriteFile(l.PlistPath, plist, 0o644)
}

// Disable removes the login item. A running menu bar keeps running.
func (l LoginItem) Disable() error {
	if err := os.Remove(l.PlistPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

var plistTemplate = template.Must(template.New("plist").Funcs(template.FuncMap{
	"xml": func(s string) (string, error) {
		var b bytes.Buffer
		err := xml.EscapeText(&b, []byte(s))
		return b.String(), err
	},
}).Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{ .Label }}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{ xml .Executable }}</string>
		<string>menubar</string>
		<string>--foreground</string>
	</array>
	<key>WorkingDirectory</key>
	<string>{{ xml .Home }}</string>
	<key>EnvironmentVariables</key>
	<dict>
		<key>PATH</key>
		<string>{{ xml .PathEnv }}</string>
	</dict>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<dict>
		<key>SuccessfulExit</key>
		<false/>
	</dict>
	<key>ProcessType</key>
	<string>Interactive</string>
</dict>
</plist>
`))

func (l LoginItem) plist() ([]byte, error) {
	var b bytes.Buffer
	err := plistTemplate.Execute(&b, struct {
		LoginItem
		Label string
	}{l, LoginLabel})

	return b.Bytes(), err
}
