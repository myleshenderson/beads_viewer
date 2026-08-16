package ui

import (
	"strings"
	"testing"

	"github.com/Dicklesworthstone/beads_viewer/pkg/model"
)

func TestPRStatusModel_FiltersToTrackedIssues(t *testing.T) {
	issues := []model.Issue{
		{ID: "app-1", Title: "Fix login", Metadata: map[string]any{"pr_ci_status": "passing", "pr_review_comment_count": float64(0)}, ExternalRef: stringPtr("https://github.com/acme/app/pull/1")},
		{ID: "app-2", Title: "Not PR-tracked"},
	}
	m := NewPRStatusModel(Theme{})
	m.SetData(issues)
	m.SetSize(80, 20)

	out := m.View()

	if !strings.Contains(out, "app-1") {
		t.Errorf("expected tracked issue app-1 in view, got:\n%s", out)
	}
	if strings.Contains(out, "app-2") {
		t.Errorf("expected untracked issue app-2 to be filtered out, got:\n%s", out)
	}
}

func TestPRStatusModel_ShowsCIAndReviewStatus(t *testing.T) {
	issues := []model.Issue{
		{ID: "app-1", Title: "Fix login", Metadata: map[string]any{"pr_ci_status": "failing", "pr_review_comment_count": float64(3)}, ExternalRef: stringPtr("https://github.com/acme/app/pull/1")},
	}
	m := NewPRStatusModel(Theme{})
	m.SetData(issues)
	m.SetSize(80, 20)

	out := m.View()

	if !strings.Contains(out, "failing") {
		t.Errorf("expected CI status 'failing' in view, got:\n%s", out)
	}
	if !strings.Contains(out, "3") {
		t.Errorf("expected review comment count '3' in view, got:\n%s", out)
	}
}

func TestPRStatusModel_MoveDownAndSelected(t *testing.T) {
	issues := []model.Issue{
		{ID: "app-1", Title: "First", Metadata: map[string]any{"pr_ci_status": "passing"}},
		{ID: "app-2", Title: "Second", Metadata: map[string]any{"pr_ci_status": "passing"}},
	}
	m := NewPRStatusModel(Theme{})
	m.SetData(issues)
	m.SetSize(80, 20)

	if got := m.Selected(); got == nil || got.ID != "app-1" {
		t.Fatalf("expected app-1 selected initially, got %+v", got)
	}

	m.MoveDown()
	if got := m.Selected(); got == nil || got.ID != "app-2" {
		t.Fatalf("expected app-2 selected after MoveDown, got %+v", got)
	}

	m.MoveDown() // at the end — must not go out of bounds
	if got := m.Selected(); got == nil || got.ID != "app-2" {
		t.Fatalf("expected MoveDown to stay on app-2 at the end, got %+v", got)
	}

	m.MoveUp()
	if got := m.Selected(); got == nil || got.ID != "app-1" {
		t.Fatalf("expected app-1 selected after MoveUp, got %+v", got)
	}
}

func TestPRStatusModel_SelectedNilWhenEmpty(t *testing.T) {
	m := NewPRStatusModel(Theme{})
	m.SetData(nil)
	m.SetSize(80, 20)

	if got := m.Selected(); got != nil {
		t.Fatalf("expected nil Selected() on an empty filtered list, got %+v", got)
	}
}
