# jira-tui

A terminal UI for Jira Cloud. Browse your issues, read and write comments,
transition tickets, log work and manage estimates without leaving the terminal.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Vim-style
keys throughout.

## Features

- **Tabbed boards** — several queries open at once, each with its own cursor,
  filter and drill-down state. Saved boards for *My Issues*, *Reported by me*
  and *Updated recently*, plus per-project and per-epic boards.
- **Issue detail** — description, comments, worklogs, links and sub-tasks, each
  a navigable section.
- **Write operations** — transition, assign, set priority, edit summary and
  description, comment (create, edit, delete), link issues, set estimates and
  start/due dates, and create issues and sub-tasks.
- **Markdown authoring** — comments and descriptions are written in Markdown and
  converted to Atlassian's ADF on the way out. `@[Display Name]` mentions are
  resolved against your instance's users.
- **Local tags** — private, local-only labels on any issue, stored on your
  machine and never sent to Jira. Filter by them with `/#tag`.
- **Time tracking** — optional, via [Tempo](https://www.tempo.io/).
- **Yank to clipboard** — issue key, URL, summary, or a formatted block.

## Install

### Homebrew

```bash
brew install --cask oliverjhernandez/tap/jira-tui
```

### From source

Requires Go 1.25 or newer.

```bash
go install github.com/oliverjhernandez/jira-tui/cmd/jira-tui@latest
```

### Prebuilt binaries

Download a tarball for macOS or Linux (amd64 or arm64) from the
[releases page](https://github.com/oliverjhernandez/jira-tui/releases).

## Configure

jira-tui reads its configuration from the environment. Three variables are
required:

| Variable | Description |
| --- | --- |
| `JIRA_URL` | Your Jira base URL, e.g. `https://your-domain.atlassian.net` (no trailing slash) |
| `JIRA_EMAIL` | The email address of the account the API token belongs to |
| `JIRA_TOKEN` | A Jira API token — **not** your password |

Create an API token at
[id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens).

Two more are optional, and only needed if your team uses Tempo for time
tracking:

| Variable | Description |
| --- | --- |
| `TEMPO_URL` | Tempo API base URL, e.g. `https://api.tempo.io` |
| `TEMPO_TOKEN` | A Tempo API token |

Without them jira-tui runs normally and hides the worklog features — the
`LOGGED` column, the *Total Logged* panel line and the `w` keybind.

Copy [`.env.example`](.env.example) as a starting point:

```bash
cp .env.example .env
$EDITOR .env
source .env
jira-tui
```

Any method of exporting the variables works — direnv, your shell profile, a
secrets manager.

## Usage

```
jira-tui              start the app
jira-tui --version    print version information
jira-tui --debug      write debug-level detail to the log
jira-tui --help       show usage
```

Press `?` inside the app for the full, always-current keybinding list. The
essentials:

| Key | Action |
| --- | --- |
| `j` / `k` | Down / up |
| `enter` | Open issue |
| `esc` | Back |
| `gt` / `gT` | Next / previous tab |
| `P` | Project picker |
| `B` | Saved boards |
| `/` | Filter the list (`/#tag` filters by local tag) |
| `ctrl+s` | Search issues |
| `t` | Transition |
| `a` | Assign |
| `c` | Comment |
| `w` | Log work (needs Tempo) |
| `#` | Edit local tags |
| `ctrl+r` | Refresh |
| `?` | Help |
| `q` | Quit |

## What jira-tui assumes about your Jira

- **Jira Cloud REST API v3.** Jira Server / Data Center is untested.
- **Issue grouping follows Jira's own status categories** (*To Do* /
  *In Progress* / *Done*), so any workflow and any language works. A few status
  and priority names are additionally recognised for finer ordering within a
  category; unrecognised ones simply sort after the ones that are.
- **Optional fields are discovered by name** at startup — start date, flagged,
  and block reason. If your instance names them differently, those specific
  features degrade gracefully rather than failing: a block reason with nowhere
  to go is posted as a comment instead.
- **Time tracking is Tempo-only.** Jira's native worklog API is not yet used as
  a fallback.

## Files jira-tui writes

| Path | Contents |
| --- | --- |
| `$XDG_STATE_HOME/jira-tui/state.json` | Local-only issue tags |
| `$XDG_STATE_HOME/jira-tui/debug.log` | Log output |

Both fall back to `~/.local/state/jira-tui/` when `XDG_STATE_HOME` is unset.
Issues themselves are not cached; everything else is in memory for the life of
the process.

## Development

```bash
make build     # build to bin/
make run       # go run
make test      # go test -race ./...
make vet
make lint      # golangci-lint
make fmt
make snapshot  # local goreleaser dry run into dist/
```

Releases are cut by pushing a `v*` tag; GitHub Actions runs GoReleaser, which
publishes the archives and updates the Homebrew tap.

## License

[MIT](LICENSE)
