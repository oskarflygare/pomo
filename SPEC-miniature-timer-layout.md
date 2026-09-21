# Spec: Miniature Timer Layout

**Status:** Draft — amended after doubt review; Phase 1 approval is required before planning or implementation.

## Objective

Replace the current `t` key behavior, which hides timer text and leaves only the progress bar, with a miniature layout for the ordinary timer view.

When enabled, the layout shows only the countdown timer and its visual progress bar side by side. It persists for the lifetime of the running program—across work, break, long-break, and short-session transitions—until the user toggles it off or exits. It reduces visual noise without changing timer, session, or keyboard-control behavior.

## Scope

### In scope

- In the ordinary timer view, `t` toggles between the existing full layout and miniature layout. Confirmation-dialog key handling and the waiting-for-post-actions view remain unchanged.
- The selected layout persists for the lifetime of the running program, including work, break, long-break, and short-session transitions, until `t` is pressed again or the program exits.
- Miniature layout shows:
  - the countdown only: `MM:SS`, or the existing `HH:MM:SS` form when one hour or more remains;
  - the visual progress bar, horizontally aligned with the timer when space permits.
- The numeric percentage is hidden in miniature layout.
- Task title, paused/completed/long-break indicators, and key-help text are hidden in miniature layout.
- If ASCII timer art is enabled, it remains enabled. The art may span several terminal rows; vertically center the one-line progress bar against the art.
- Preserve the existing full-layout width budget: `fullBudget = max(0, min(terminalWidth - 2*padding - margin, maxWidth))` (currently `max(0, min(terminalWidth - 8, 80))`). In miniature mode, use `barWidth = max(0, fullBudget - lipgloss.Width(renderedTimer) - 1)`, where the final cell is the single-space timer/bar gap.
- Derive the miniature progress-bar width from the current rendered timer on every miniature render or timer/layout update. This covers toggles, terminal resizes, accepted timer ticks, duration increase/reset, and work/break/long-break/short-session transitions. Restore `fullBudget` when leaving miniature mode.
- Clamp both miniature and full-layout progress-bar widths to zero or greater. If no progress-bar cells fit, miniature layout shows the complete timer only; it does not wrap. At widths narrower than the timer itself, horizontal terminal clipping/overflow is permitted.
- Toggling back restores the full layout, including the numeric percentage, help text, and its non-negative normal progress-bar width.

### Out of scope

- New YAML configuration, schema, sample-config, or README options. The layout is in-memory only and controlled only by `t`.
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
- Use the public `progress.Model.Width` property for the current layout budget. Bubbles documents that it can be set after a window-size message; this feature additionally refreshes it on layout and rendered-timer-width changes. Its width includes the percentage when shown.
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

- `ui/model.go` — long-lived UI state and defaults; owns the in-memory miniature-mode state. Do not use `NewModel` in layout/transition tests because it opens the persistent user database.
- `ui/handlers.go` — toggles miniature mode for `t`, preserves it across session transitions, and refreshes the progress-bar width after relevant state changes.
- `ui/layout.go` — renders the full and miniature layouts and defines the existing `padding`, `margin`, and `maxWidth` constants used by the width budget.
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

1. **Full → miniature:** after a normal-width resize, pressing `t` hides title, indicators, help, and numeric percentage; it retains a formatted time value and a visible progress bar on the same rendered row.
2. **Miniature → full:** pressing `t` again restores title, help, numeric percentage, and a non-negative normal progress-bar width.
3. **State guards:** pressing `t` while showing the confirmation dialog or waiting for post-actions does not alter the selected layout.
4. **Persistent selection:** miniature mode remains selected after work, break, long-break, and short-session transitions; it changes only when `t` is pressed again or the program exits.
5. **Width mutation paths:** miniature rendering recalculates the exact `max(0, fullBudget - timerWidth - 1)` bar budget after an accepted tick, duration increase, reset, and each session transition, including transitions that cross the one-hour formatting boundary.
6. **Narrow terminal:** when the computed bar budget is zero, miniature rendering retains the complete timer and omits the bar without a negative width or wrapping. At widths narrower than the timer, horizontal clipping/overflow is the accepted result. Leaving miniature restores the non-negative `fullBudget`.
7. **ASCII art:** for each configured font (`mono12`, `rebel`, `ansi`, and `ansiShadow`), a digit-changing tick recomputes the bar width; the multiline timer is preserved and the bar is vertically centered beside it when room exists.
8. **Test isolation:** layout and transition tests do not call `NewModel`; they construct a minimal `Model` with `repo: nil`. Tests save and restore the process-global `config.C`, give its work/break tasks deterministic durations and titles consistent with the direct model fixture, and set the fixture's `longBreak` explicitly.

Run the focused UI test during development, then `go build ./...`, `go vet ./...`, and `go test ./...` before review.

## Boundaries

- **Always:** preserve the existing Bubble Tea message/command flow and its confirmation/waiting key behavior; calculate terminal display widths with Lip Gloss; use the specified full and miniature width formulas; use non-negative widths; refresh miniature width from the current timer on every relevant update; construct database-free UI fixtures and isolate global configuration in transition tests; add regression tests; run formatting, build, vet, and tests.
- **Ask first:** changing the miniature content beyond timer plus bar; adding a persistent configuration option; changing key bindings; adding dependencies; changing normal-layout behavior.
- **Never:** change session accounting or persistence; remove existing tests; bypass the resize path; commit generated binaries, `dist/`, debug logs, or databases.

## Success Criteria

- In the ordinary timer view, pressing `t` toggles an in-memory miniature layout without altering timer state; confirmation and waiting-state key behavior is unchanged, and the choice persists across subsequent timer sessions until toggled back or exit.
- At ordinary terminal widths, miniature layout contains only the timer and visual bar horizontally; no task metadata, help, or numeric percentage appears.
- ASCII art remains supported in miniature layout across every configured font, including its permitted multiline rendering with the bar vertically centered beside it.
- The miniature bar width is exactly `max(0, max(0, min(terminalWidth - 8, 80)) - lipgloss.Width(renderedTimer) - 1)`; the full layout restores `max(0, min(terminalWidth - 8, 80))`.
- At widths without room for bar cells, or after any timer-width change, the timer stays visible and the bar is absent or resized; neither layout assigns a negative width or inserts wrapping. Extremely narrow terminals may horizontally clip the complete timer.
- A second `t` restores the pre-existing full layout behavior and its non-negative normal progress-bar width.
- Focused UI tests and the full Go build, vet, and test suite pass.

## Unverified / Platform Notes

- Bubble Tea's source notes that Windows does not report resize events through `SIGWINCH`; initial sizing still occurs, but live-resize behavior should be manually checked on Windows.
  - https://github.com/charmbracelet/bubbletea/blob/v1.3.10/screen.go#L3-L10
- The libraries document width APIs, but do not define this product's timer-only narrow-window fallback, ultra-narrow clipping behavior, or ASCII-art alignment; those policies are specified here by user decision.
