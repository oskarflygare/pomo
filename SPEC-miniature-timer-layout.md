# Spec: Miniature Timer Layout

**Status:** Draft — Phase 1 review required before planning or implementation.

## Objective

Replace the current `t` key behavior, which hides timer text and leaves only the progress bar, with a transient miniature layout for the active Pomodoro session.

When enabled, the layout shows only the countdown timer and its visual progress bar side by side. It is intended to reduce visual noise without changing timer, session, or keyboard-control behavior.

## Scope

### In scope

- `t` toggles between the existing full layout and miniature layout during the current session.
- Miniature layout shows:
  - the timer only (`MM:SS`, or `HH:MM:SS` when the existing formatter includes hours);
  - the visual progress bar, horizontally aligned with the timer when space permits.
- The numeric percentage is hidden in miniature layout.
- Task title, paused/completed/long-break indicators, and key-help text are hidden in miniature layout.
- If ASCII timer art is enabled, it remains enabled. The art may span several terminal rows, with the progress bar horizontally joined beside it.
- On a terminal resize, the progress bar width is recalculated after reserving space for the rendered timer and separator.
- If no progress-bar cells fit, miniature layout shows the timer only; it must not wrap or use a negative bar width.
- Toggling back restores the full layout, including the numeric percentage and help text.

### Out of scope

- New YAML configuration, schema, sample-config, or README options. The layout is session-local and controlled only by `t`.
- Changes to session timing, progress animation, pause/reset/skip/quit behavior, notifications, post-session actions, database storage, statistics, or confirmation dialogs.
- New key bindings, dependencies, or terminal-detection packages.
- A literal one-row guarantee for ASCII art; multiline art is explicitly allowed.

## Tech Stack

- Go 1.25
- Bubble Tea `v1.3.10`
- Bubbles progress `v0.21.0`
- Lip Gloss `v1.1.0`
- `testify` `v1.11.1`

## Documented Implementation Constraints

- Use `lipgloss.JoinHorizontal` to compose the rendered timer and progress bar. It is documented for horizontally joining potentially multiline strings with explicit vertical alignment.
  - https://pkg.go.dev/github.com/charmbracelet/lipgloss@v1.1.0#JoinHorizontal
- Use `lipgloss.Width` to reserve the rendered timer's terminal-cell width; it correctly handles ANSI sequences and wide characters, unlike `len`.
  - https://pkg.go.dev/github.com/charmbracelet/lipgloss@v1.1.0#Width
- Update `progress.Model.Width` on the existing `tea.WindowSizeMsg` path. Bubbles documents both the public `Width` property and its use after a window-size message; its width includes the percentage when shown.
  - https://pkg.go.dev/github.com/charmbracelet/bubbles@v0.21.0/progress#WithWidth
- Do not add an explicit window-size command: Bubble Tea documents that `WindowSizeMsg` is delivered at program start and when dimensions change.
  - https://pkg.go.dev/github.com/charmbracelet/bubbletea@v1.3.10#WindowSize

## Commands

```sh
go fmt ./...
go build ./...
go vet ./...
go test ./...
go test ./ui -run TestTextToggle
```

## Project Structure

- `ui/model.go` — session UI state and defaults; owns the session-local miniature-mode state.
- `ui/handlers.go` — toggles miniature mode for `t` and recalculates the progress-bar width on resize.
- `ui/layout.go` — renders the full and miniature layouts.
- `ui/keys.go` — retains `t` as the documented toggle key; update help text only if it no longer describes the behavior accurately.
- `ui/app_test.go` — behavior-focused regression tests for rendering and toggling.

## Code Style

Follow the existing Bubble Tea dispatch and pointer-handler pattern. Keep layout rendering separate from state mutation.

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, m.handleKeys(msg)
	case tea.WindowSizeMsg:
		return m, m.handleWindowResize(msg)
	default:
		return m, nil
	}
}
```

Use `gofmt`. Keep the new layout logic in the existing `ui` package; do not create a generic layout abstraction for this one view.

## Testing Strategy

Add or replace the existing `t`-toggle test in `ui/app_test.go`. Tests must assert rendered user-visible outcomes, not Lip Gloss internals.

1. **Full → miniature:** pressing `t` hides title, indicators, help, and numeric percentage; it retains a time value and a visible progress bar on the same rendered row at a normal terminal width.
2. **Miniature → full:** pressing `t` again restores title, help, and numeric percentage.
3. **Narrow terminal:** after a resize too narrow for timer plus bar, miniature rendering retains the timer, omits the bar, and does not create a negative width or wrapped layout.
4. **ASCII art:** when enabled, miniature rendering preserves the existing art and places the bar beside it when room exists; multiple timer rows are allowed.

Run the focused UI test during development, then `go build ./...`, `go vet ./...`, and `go test ./...` before review.

## Boundaries

- **Always:** preserve the existing Bubble Tea message/command flow; calculate terminal display widths with Lip Gloss; add regression tests; run formatting, build, vet, and tests.
- **Ask first:** changing the miniature content beyond timer plus bar; adding a persistent configuration option; changing key bindings; adding dependencies; changing normal-layout behavior.
- **Never:** change session accounting or persistence; remove existing tests; bypass the resize path; commit generated binaries, `dist/`, debug logs, or databases.

## Success Criteria

- Pressing `t` toggles a session-local miniature layout and does not alter timer state.
- At ordinary terminal widths, miniature layout contains only the timer and visual bar horizontally; no task metadata, help, or numeric percentage appears.
- ASCII art remains supported in miniature layout, including its permitted multiline rendering.
- At widths without room for bar cells, the timer stays visible and the bar is absent rather than wrapped or rendered with an invalid width.
- A second `t` restores the pre-existing full layout behavior.
- Focused UI tests and the full Go build, vet, and test suite pass.

## Unverified / Platform Notes

- Bubble Tea's source notes that Windows does not report resize events through `SIGWINCH`; initial sizing still occurs, but live-resize behavior should be manually checked on Windows.
  - https://github.com/charmbracelet/bubbletea/blob/v1.3.10/screen.go#L3-L10
- The libraries document width APIs, but do not define this product's timer-only narrow-window fallback; that policy is specified here by user decision.
