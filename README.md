# Go Remind Me!

A terminal-based reminder app with live markdown parsing and a beautiful TUI.

![Go Remind Me!](https://img.shields.io/badge/go-%3E%3D1.21-blue)

## Usage

### Method 1: Live Markdown Parsing

Point Go Remind Me! at a directory and leave it running. As you go about your day editing markdown files, any `[remind_me]` tags you add will automatically appear in the TUI:

```bash
# Watch an entire directory (recursively finds .md files)
./go_remind ~/notes/

# Or watch a single file
./go_remind notes.md
```

Then, in any markdown file within that directory, embed reminders inline as you take notes:

```markdown
# Meeting Notes

Don't forget to follow up with the team [remind_me +2h Send meeting summary]

## Action Items

- Review PR [remind_me tomorrow 9am Review PR #123]
- Call mom [remind_me Jan 15 3pm Call mom]
```

Go Remind Me! watches for file changes in real-time—save your file and the reminder instantly appears.

### Method 2: Create Reminders in the TUI

Run Go Remind Me! without arguments to use it standalone:

```bash
./go_remind
```

Press `n` to create a new reminder directly in the app.

### Datetime Formats

Go Remind supports flexible datetime parsing:

| Format | Example |
|--------|---------|
| Relative | `+30m`, `+2h`, `+1d`, `+1h30m` |
| Natural | `tomorrow`, `tomorrow 9am`, `in 3 days`, `in 2 hours` |
| Time only (today) | `3pm`, `3:30pm`, `15:30` |
| Date + time | `Jan 15 3pm`, `January 15 3:30pm` |
| Full date | `Jan 15 2025 3pm`, `2025-01-15 15:30` |

## Keybindings

| Key | Action |
|-----|--------|
| `↑/k` | Move up |
| `↓/j` | Move down |
| `←/h` | Move left (card view) |
| `→/l` | Move right (card view) |
| `Enter/Space` | Acknowledge (mark done) |
| `u` | Unacknowledge (reopen) |
| `dd` | Delete reminder |
| `1` | Snooze 5 minutes |
| `2` | Snooze 1 hour |
| `3` | Snooze 1 day |
| `/` | Filter reminders |
| `n` | New reminder |
| `t` | Change theme |
| `v` | Toggle view (compact/card) |
| `?` | Toggle help |
| `q` | Quit |

## Views

Press `v` to toggle between views:

- **Compact**: Single-line items, dense list
- **Card**: Bordered cards in a responsive grid layout

## Themes

Press `t` to open the theme picker. Available themes:

- Everforest (default)
- Kiro Purple
- Dracula
- Nord
- Solarized
- Monokai

Navigate with `↑/k` and `↓/j` to preview themes live, then press `Enter` to select or `Esc` to cancel.

## Reminder States

| State | Icon | Description |
|-------|------|-------------|
| Pending | `○` | Waiting for trigger time |
| Triggered | `🔔` | Time reached, needs attention |
| Acknowledged | `✓` | Marked as done (strikethrough) |

## State Persistence

Reminders are automatically saved to `~/.go_remind/reminders_state.json`. Your reminder states (acknowledged, snoozed times, etc.) persist across sessions.

## Dependencies

Go Remind is built with these excellent libraries:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - The Elm-inspired TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components (list, text input, help)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Style definitions for terminal layouts
- [fsnotify](https://github.com/fsnotify/fsnotify) - Cross-platform filesystem notifications

## Architecture

```
go_remind/
├── main.go           # Entry point, CLI handling, watcher setup
├── tui/
│   ├── tui.go        # Bubble Tea model, views, and update logic
│   ├── theme.go      # Color theme definitions
│   └── layout.go     # Layout mode (compact/card)
├── reminder/
│   └── reminder.go   # Reminder struct, status enum, sorting, merging
├── parser/
│   └── parser.go     # Markdown [remind_me] tag extraction
├── datetime/
│   └── datetime.go   # Flexible datetime parsing (relative, absolute)
├── watcher/
│   └── watcher.go    # Filesystem watching with fsnotify
└── state/
    └── state.go      # JSON persistence to ~/.go_remind/
```

### Data Flow

1. **Startup**: Load saved state from disk, optionally parse markdown files
2. **File Watching**: fsnotify detects changes → parser extracts reminders → merge with existing state
3. **TUI Loop**: Bubble Tea handles input → updates model → renders view
4. **Tick**: Every second, check for newly triggered reminders
5. **Persistence**: State saved on every change (acknowledge, snooze, delete, add)

### Key Design Decisions

- **Merge Strategy**: File-parsed reminders are matched by description + source file to preserve user state (acknowledged, snoozed) across file edits
- **Dual Input Modes**: Supports both embedded markdown workflow and standalone TUI creation
- **Theme/Layout Separation**: Colors and layout density are independent settings
- **Grid Navigation**: Card view calculates columns dynamically based on terminal width

## Building

```bash
go build .
./go_remind
```

### Global Access

Add an alias to your shell config (`~/.zshrc` or `~/.bashrc`) to run from anywhere:

```bash
alias remind="/path/to/go_remind"
```

Then reload your shell and use it from any directory:

```bash
remind ~/notes/
remind
```

## License

MIT
