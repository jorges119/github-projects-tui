# ghtui

[![CI](https://github.com/jhermoso/ghtui/actions/workflows/ci.yml/badge.svg)](https://github.com/jhermoso/ghtui/actions/workflows/ci.yml)

A terminal UI for browsing and managing GitHub Projects, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- Browse all GitHub Projects you have access to (personal and org)
- Filter issues and PRs by iteration, status, type, assignee, label, title, or description
- View full issue detail with metadata (status, sprint, assignees, labels, body)
- Edit issue title, description, and project status without leaving the terminal
- Quick-change project status from the list view
- Open any issue in the browser
- Vim-style keyboard navigation throughout

## Requirements

- Go 1.25+
- A GitHub account with access to one or more GitHub Projects (v2)

## Installation

```bash
git clone https://github.com/jhermoso/ghtui
cd ghtui
go build -o ghtui ./cmd/main.go
```

Or install directly:

```bash
go install github.com/jhermoso/ghtui/cmd/main.go@latest
```

## Authentication

On first launch, `ghtui` will display an authentication screen. Two methods are supported:

### Personal Access Token (PAT)

1. Go to **GitHub → Settings → Developer Settings → Personal access tokens**
2. Generate a token with the following scopes: `repo`, `project`, `read:org`
3. Select **Personal Access Token** in `ghtui` and paste the token

Alternatively, set the `GITHUB_TOKEN` environment variable to skip the auth screen:

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
```

### Device Flow (OAuth App)

Requires a GitHub OAuth App. Set your app's Client ID before launching:

```bash
export GHTUI_CLIENT_ID=<your-oauth-app-client-id>
```

Then select **Device Flow** in the auth screen. Follow the on-screen instructions to authorize in your browser.

Tokens are persisted to `~/.config/ghtui/token` and reused on subsequent runs.

## Configuration

`ghtui` reads `~/.config/ghtui/config.json`. All fields are optional:

```json
{
  "client_id": "your-oauth-app-client-id"
}
```

The `GHTUI_CLIENT_ID` environment variable takes precedence over the config file.

## Usage

```bash
./ghtui
```

### Layout

```
┌─────────────────────────────────────────────────────┐
│  Filter Bar (9 fields across 3 rows)                │
├────────────────────────┬────────────────────────────┤
│  Issue List            │  Detail / Edit Pane        │
│                        │                            │
├────────────────────────┴────────────────────────────┤
│  Hint Bar                                           │
└─────────────────────────────────────────────────────┘
```

### Filter Bar

Press `1`–`9` from anywhere (except edit mode) to jump directly to a filter field.

| Key | Field      | Type     |
|-----|------------|----------|
| `1` | Org        | dropdown |
| `2` | Project    | dropdown |
| `3` | Iteration  | dropdown |
| `4` | Status     | dropdown |
| `5` | Type       | dropdown (all / issue / pr) |
| `6` | Assignee   | text     |
| `7` | Label      | text     |
| `8` | Title      | text     |
| `9` | Desc       | text     |

**In the filter bar:**

| Key | Action |
|-----|--------|
| `tab` / `h` / `l` | Next / previous field |
| `j` / `k` | Cycle dropdown value |
| `enter` | Apply filters and return to list |
| `esc` | Return to list without applying |

### Issue List

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate down / up |
| `gg` | Jump to top |
| `G` | Jump to bottom |
| `ctrl+d` / `ctrl+u` | Half-page down / up |
| `enter` | Open detail pane |
| `e` | Edit issue |
| `s` | Quick-change project status |
| `o` | Open issue in browser |
| `r` | Refresh items from GitHub |
| `tab` / `f` | Jump to filter bar |
| `1`–`9` | Jump to filter field |

### Detail Pane

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll down / up |
| `ctrl+d` / `ctrl+u` | Half-page down / up |
| `g` / `G` | Top / bottom |
| `e` | Edit issue |
| `s` | Quick-change project status |
| `o` | Open issue in browser |
| `esc` / `q` | Return to list |

### Edit Mode

| Key | Action |
|-----|--------|
| `tab` / `shift+tab` | Next / previous field (Title → Body → Status) |
| `h` / `l` | Cycle status (when Status field is focused) |
| `ctrl+s` | Save changes to GitHub |
| `esc` | Cancel and return to detail |

### Quick Status Picker

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate options |
| `enter` | Apply selected status |
| `esc` / `q` | Cancel |

### Global

| Key | Action |
|-----|--------|
| `?` | Toggle help overlay |
| `ctrl+c` | Quit |

## Token Storage

The authenticated token is stored at `~/.config/ghtui/token` with `0600` permissions. To log out and re-authenticate, delete this file or set an invalid token — `ghtui` will prompt for authentication on next launch.

## License

MIT
