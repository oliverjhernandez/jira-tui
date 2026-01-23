# jira-tui Roadmap

This document tracks planned features and improvements for jira-tui.

## High Priority

### Bug Fixes

- [ ] **Fix worklog totals inconsistency** - Worklog totals column changes/disappears when navigating between list and detail views
- [ ] **Consistent width management** - Ensure all views (list, detail, modals) use the same width calculation for consistent layout
- [ ] **Dynamic column widths** - Calculate column widths dynamically based on terminal width
- [ ] **Terminal resize handling** - Handle terminal resize events gracefully without breaking layout

### UI/UX Improvements

- [ ] **Top panel across all views** - Display info panel (user, projects, status counts) consistently across list view, detail view, and modals
- [ ] **Include worklog totals in top panel** - Add total logged time across all issues to the info panel
- [ ] **Contextual footer keybindings** - Show footer with relevant keybindings for current view/modal
- [ ] **Description formatting** - Format description text properly (preserve line breaks, formatting)
- [ ] **Empty description handling** - Handle missing/empty descriptions gracefully
- [ ] **Revamp modals** - Improve modal appearance, size, and positioning:
  - Transition view
  - Assignable users search
  - Edit description
  - Edit priority
  - Post comment
  - Post worklog
  - Post estimate
  - Cancel reason
- [ ] **Fix command descriptions** - Review and improve help text/keybind descriptions across all views and modals
- [ ] **Loading screen** - Add initial loading screen when starting the app (implement last)

### List View Enhancements

- [ ] **Secondary sort by priority** - Within status groups, sort issues by priority (High → Medium → Low)
- [ ] **Real-time filtering** - Implement `/` filter system for live issue filtering by Key, Summary, or Status
- [ ] **Tab system** - Add tab structure (Active Work | Completed) with `[` `]` navigation
- [ ] **Completed tab** - JQL query for Done issues: `status = Done AND updated >= -7d`, sorted by completion date

### Core Features

- [ ] **Manual refresh command** - Add keybind to manually refresh current view data
- [ ] **Data persistence** - Cache issues, worklogs, and user data locally to improve startup performance
- [ ] **Auto-refresh persistence** - Refresh cached data every few minutes to sync with remote changes
- [ ] **Cancel issue from list view** - Add ability to transition issue to cancelled status directly from list view
- [ ] **Pre-fetch next/previous issue** - Background fetch of adjacent issues in detail view for instant navigation
- [ ] **API caching** - Implement caching strategy to avoid redundant API calls (transitions, users, etc.)
- [ ] **Optimistic UI updates** - Update UI immediately, sync with API in background

### Workflow & Time Tracking

- [ ] **Transition validation** - Check required fields before allowing transitions, show warnings for missing prerequisites
- [ ] **Original Estimate field** - Display in header, add `o` keybind to edit, validate format (e.g., "8h", "2d")
- [ ] **Time logging on Done** - Automatic worklog prompt when transitioning to Done status
- [ ] **Standalone time logging** - `l` keybind to log time without transition
- [ ] **Worklog history** - Display all worklog entries in detail view with author, time, date, comment

## Medium Priority

### New Views

- [ ] **Unassigned tasks view** - Dedicated view to browse and manage unassigned issues
- [ ] **Search view** - Search for specific issues by key, summary, assignee, etc.
- [ ] **Team Work tab** - View issues assigned to others in same project
- [ ] **All Issues tab** - No assignee filter, show all issues in projects
- [ ] **Reports & Analytics** - Expanded Completed tab with:
  - Configurable timeframes (this week, last week, last month, custom range)
  - Metrics: total issues completed, total time spent, velocity
  - Summary statistics
  - Export capability

### Issue Management

- [ ] **Link issues** - Add ability to link current issue to another issue (blocks, is blocked by, relates to, etc.)
- [ ] **Create new issue** - Form to create new Jira issues from the TUI with required fields
- [ ] **Bulk actions from search** - Perform actions (assign, comment, edit) on issues found via search
- [ ] **Edit/delete own comments** - Allow modifying or removing comments you posted
- [ ] **Custom fields support** - Display and edit custom Jira fields
- [ ] **Time tracking column** - Optional "Logged" column in list view showing total time per issue

### Project Management

- [ ] **Project filtering** - Filter issues by specific projects
- [ ] **Quick project switching** - Keybinding to show project list and switch contexts
- [ ] **Project metadata display** - Show project info in detail view
- [ ] **Per-project configuration** - Custom fields, workflows, etc. per project
- [ ] **Group by project** - Optional alternative to mixed list view

### Notifications

- [ ] **Desktop notifications** - Notify user of:
  - Issues assigned to them
  - Comments on their issues
  - Status changes
  - Due dates approaching

## Low Priority / Future Exploration

### Automation

- [ ] **Automated workflows** - Configure custom automation rules:
  - Auto-assign based on criteria
  - Auto-transition on conditions
  - Scheduled actions
  - Templates for common operations
- [ ] **Macro/script support** - Allow users to define reusable action sequences

### Advanced Features

- [ ] **Multiple project views** - Switch between different project contexts
- [ ] **Custom filters** - Save and apply custom JQL filters
- [ ] **Time tracking reports** - Visualize worklog data and time spent
- [ ] **Keyboard shortcut customization** - Allow users to rebind keys

## Implementation Notes

### Architecture Considerations

- Use consistent `panelWidth` calculation across all views
- Implement proper caching layer with TTL for persistence
- Consider background goroutines for auto-refresh without blocking UI
- Modal system could benefit from a reusable component approach

### Technical Debt

- Refactor modal rendering into shared components
- Consolidate width/height calculations into a layout manager
- Add proper error handling for persistence layer
- Improve test coverage

## Completed Features

- [x] Information panel in list view
- [x] Tempo worklog time column
- [x] Move Validación issues to Done section
- [x] Refresh worklogs after posting
- [x] Align info panel with list columns

---

**Last Updated**: 2026-01-22
