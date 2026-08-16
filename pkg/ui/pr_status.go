package ui

import (
	"fmt"
	"strings"

	"github.com/Dicklesworthstone/beads_viewer/pkg/model"
)

type prStatusRow struct {
	issue model.Issue
}

// PRStatusModel renders a focused list of issues that pr-monitor is actively
// tracking, showing each PR's CI status and outstanding review comments.
type PRStatusModel struct {
	theme  Theme
	rows   []prStatusRow
	cursor int
	width  int
	height int
}

// NewPRStatusModel creates a new PR status view.
func NewPRStatusModel(theme Theme) PRStatusModel {
	return PRStatusModel{theme: theme}
}

// SetData filters to issues pr-monitor is actively tracking — identified by
// the presence of the pr_ci_status metadata key. An issue with no such key
// was either never PR-tracked or pr-monitor hasn't synced it yet; this view
// treats that as "not shown", not an error.
func (m *PRStatusModel) SetData(issues []model.Issue) {
	m.rows = m.rows[:0]
	for _, issue := range issues {
		if _, ok := issue.Metadata["pr_ci_status"]; !ok {
			continue
		}
		m.rows = append(m.rows, prStatusRow{issue: issue})
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// SetSize sets the available rendering dimensions.
func (m *PRStatusModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// MoveUp moves the cursor up by one, clamped at the top.
func (m *PRStatusModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// MoveDown moves the cursor down by one, clamped at the bottom.
func (m *PRStatusModel) MoveDown() {
	if m.cursor < len(m.rows)-1 {
		m.cursor++
	}
}

// Selected returns the currently-highlighted row's issue, or nil if the
// filtered list is empty.
func (m *PRStatusModel) Selected() *model.Issue {
	if len(m.rows) == 0 {
		return nil
	}
	return &m.rows[m.cursor].issue
}

// prCIStatus extracts the pr_ci_status metadata value as a string. pr-monitor
// always writes this key as a JSON string ("passing"/"failing"/"pending"), so
// it decodes as a Go string; a malformed value of another JSON type is
// rendered as "unknown" rather than panicking on a bad type assertion.
func prCIStatus(issue model.Issue) string {
	v, ok := issue.Metadata["pr_ci_status"]
	if !ok {
		return "unknown"
	}
	s, ok := v.(string)
	if !ok {
		return "unknown"
	}
	return s
}

// prReviewCommentCount extracts the pr_review_comment_count metadata value
// for display. It decodes as a JSON number (Go float64), not a string, so
// it's formatted with %v rather than assumed to already be a string.
func prReviewCommentCount(issue model.Issue) string {
	v, ok := issue.Metadata["pr_review_comment_count"]
	if !ok {
		return "-"
	}
	return fmt.Sprintf("%v", v)
}

// prExternalRef returns the PR URL, or "" if unset.
func prExternalRef(issue model.Issue) string {
	if issue.ExternalRef == nil {
		return ""
	}
	return *issue.ExternalRef
}

// View renders the PR status list.
func (m PRStatusModel) View() string {
	if len(m.rows) == 0 {
		return m.theme.Renderer.NewStyle().Faint(true).Render("No PR-tracked issues.")
	}

	var b strings.Builder
	for i, row := range m.rows {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		ci := prCIStatus(row.issue)
		reviewCount := prReviewCommentCount(row.issue)
		ref := prExternalRef(row.issue)

		ciColor := m.theme.Subtext
		switch ci {
		case "passing":
			ciColor = m.theme.Open
		case "failing":
			ciColor = m.theme.Blocked
		case "pending":
			ciColor = m.theme.Task
		}
		ciStyle := m.theme.Renderer.NewStyle().Foreground(ciColor)

		line := fmt.Sprintf("%s%s  %s  [%s]  comments:%s  %s",
			cursor, row.issue.ID, row.issue.Title, ciStyle.Render(ci), reviewCount, ref)

		if i == m.cursor {
			line = m.theme.Renderer.NewStyle().Background(m.theme.Highlight).Render(line)
		}

		b.WriteString(line)
		if i < len(m.rows)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
