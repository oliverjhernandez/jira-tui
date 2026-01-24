# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

jira-tui is a terminal user interface for Jira built with Go using the Bubble Tea framework. It integrates with both the Jira REST API v3 and Tempo API for time tracking.

## Build and Run

```bash
# Build the binary
go build -o bin/jira-tui ./cmd/jira-tui

# Run the application
./bin/jira-tui

# Run directly without building
go run ./cmd/jira-tui
```

## Required Environment Variables

The application requires these environment variables (typically in `.env`):
- `JIRA_URL` - Jira instance base URL
- `JIRA_EMAIL` - User email for authentication
- `JIRA_TOKEN` - Jira API token
- `TEMPO_URL` - Tempo API base URL
- `TEMPO_TOKEN` - Tempo API token

## Architecture

### Bubble Tea Pattern

The app follows the Elm architecture via Bubble Tea:
- **Model** (`cmd/jira-tui/main.go`): Central state struct holding all application state including current view mode, issues, cursors, and UI components
- **Update**: Handles messages and returns new state + commands. View-specific update handlers are in separate files (e.g., `list.go`, `detail.go`)
- **View**: Renders current state to string. Each view mode has its own render function

### View Modes

The app uses `viewMode` enum to track current screen (`cmd/jira-tui/commands.go:12-22`):
- `listView` - Main issue list grouped by status category
- `detailView` - Single issue detail with comments
- `transitionView` - Status transition picker
- `assignableUsersSearchView` - User search for assignment
- `editDescriptionView`, `editPriorityView`, `postCommentView`, `postWorklogView`, `postEstimateView` - Edit forms

### Key Packages

- `cmd/jira-tui/` - Main application, Bubble Tea model, view handlers
- `internal/jira/` - Jira and Tempo API client (`client.go`)
- `internal/config/` - Environment config loading
- `internal/ui/` - Lipgloss styles and rendering helpers using Catppuccin theme

### Command Pattern

Async operations use Bubble Tea commands that return messages:
- Commands are defined in `commands.go` (e.g., `fetchMyIssues()`, `fetchIssueDetail()`)
- Each command returns a `tea.Cmd` that performs API calls and returns a message type
- Messages are handled in `Update()` to update state

### Issue Classification

Issues are grouped into sections by status category (`classifyIssues()` in `commands.go:291`):
- "In Progress" (`indeterminate`)
- "To Do" (`new`)
- "Done" (`done`)

The `Projects` variable in `main.go:18` defines which Jira projects to fetch statuses from.

## Debug Logging

The app writes debug logs to `debug.log` in the current directory.

## Coding Style

- Do not add comments to code
