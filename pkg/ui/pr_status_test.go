package ui

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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

// stubExecCommand swaps runBdCommand for a recording stub and returns a
// restore func. No exec-stubbing helper existed elsewhere in pkg/ui's tests
// (recon: no os/exec usage in pkg/ui/*_test.go), so runBdCommand is a
// package-level var function value that tests can reassign directly — the
// minimal seam needed without inventing a bigger mocking framework.
func stubExecCommand(record func(name string, args ...string)) func() {
	original := runBdCommand
	runBdCommand = func(args ...string) error {
		record("bd", args...)
		return nil
	}
	return func() { runBdCommand = original }
}

// stubExecCommandErr swaps runBdCommand for a stub that always fails, so
// tests can assert the failure-path status message.
func stubExecCommandErr(err error) func() {
	original := runBdCommand
	runBdCommand = func(args ...string) error {
		return err
	}
	return func() { runBdCommand = original }
}

func TestHandlePRStatusKeys_RequestReview(t *testing.T) {
	os.Setenv("BV_TEST_MODE", "1")
	defer os.Unsetenv("BV_TEST_MODE")

	m := newTestModel()
	m.width, m.height = 80, 24
	m.prStatus = NewPRStatusModel(m.theme)
	m.prStatus.SetData([]model.Issue{{ID: "app-1", Title: "Fix login", Metadata: map[string]any{"pr_ci_status": "passing"}}})
	m.focused = focusPRStatus

	var ranArgs []string
	restoreExec := stubExecCommand(func(name string, args ...string) { ranArgs = append([]string{name}, args...) })
	defer restoreExec()

	m.handlePRStatusKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	want := []string{"bd", "update", "app-1", "--add-label", "review-requested", "--json"}
	if !reflect.DeepEqual(ranArgs, want) {
		t.Fatalf("expected exec args %v, got %v", want, ranArgs)
	}
}

func TestHandlePRStatusKeys_RequestCIFix(t *testing.T) {
	os.Setenv("BV_TEST_MODE", "1")
	defer os.Unsetenv("BV_TEST_MODE")

	m := newTestModel()
	m.width, m.height = 80, 24
	m.prStatus = NewPRStatusModel(m.theme)
	m.prStatus.SetData([]model.Issue{{ID: "app-1", Title: "Fix login", Metadata: map[string]any{"pr_ci_status": "failing"}}})
	m.focused = focusPRStatus

	var ranArgs []string
	restoreExec := stubExecCommand(func(name string, args ...string) { ranArgs = append([]string{name}, args...) })
	defer restoreExec()

	m.handlePRStatusKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	want := []string{"bd", "update", "app-1", "--add-label", "ci-fix-requested", "--json"}
	if !reflect.DeepEqual(ranArgs, want) {
		t.Fatalf("expected exec args %v, got %v", want, ranArgs)
	}
}

func TestHandlePRStatusKeys_NoOpWhenNoRowSelected(t *testing.T) {
	os.Setenv("BV_TEST_MODE", "1")
	defer os.Unsetenv("BV_TEST_MODE")

	m := newTestModel()
	m.prStatus = NewPRStatusModel(m.theme) // no data — Selected() is nil
	m.focused = focusPRStatus

	var called bool
	restoreExec := stubExecCommand(func(name string, args ...string) { called = true })
	defer restoreExec()

	m = m.handlePRStatusKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	if called {
		t.Fatal("expected no exec call when no row is selected")
	}
	if m.statusMsg == "" || !m.statusIsError {
		t.Fatalf("expected an error status message when nothing is selected, got statusMsg=%q statusIsError=%v", m.statusMsg, m.statusIsError)
	}
}

func TestHandlePRStatusKeys_RequestReviewSetsSuccessStatus(t *testing.T) {
	os.Setenv("BV_TEST_MODE", "1")
	defer os.Unsetenv("BV_TEST_MODE")

	m := newTestModel()
	m.prStatus = NewPRStatusModel(m.theme)
	m.prStatus.SetData([]model.Issue{{ID: "app-1", Title: "Fix login", Metadata: map[string]any{"pr_ci_status": "passing"}}})
	m.focused = focusPRStatus

	restoreExec := stubExecCommand(func(name string, args ...string) {})
	defer restoreExec()

	m = m.handlePRStatusKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	if m.statusIsError {
		t.Fatalf("expected success status, got error: %q", m.statusMsg)
	}
	if !strings.Contains(m.statusMsg, "app-1") {
		t.Errorf("expected status message to mention app-1, got %q", m.statusMsg)
	}
}

func TestHandlePRStatusKeys_RequestReviewSetsFailureStatus(t *testing.T) {
	os.Setenv("BV_TEST_MODE", "1")
	defer os.Unsetenv("BV_TEST_MODE")

	m := newTestModel()
	m.prStatus = NewPRStatusModel(m.theme)
	m.prStatus.SetData([]model.Issue{{ID: "app-1", Title: "Fix login", Metadata: map[string]any{"pr_ci_status": "passing"}}})
	m.focused = focusPRStatus

	restoreExec := stubExecCommandErr(fmt.Errorf("bd: command not found"))
	defer restoreExec()

	m = m.handlePRStatusKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	if !m.statusIsError {
		t.Fatalf("expected error status, got success: %q", m.statusMsg)
	}
	if !strings.Contains(m.statusMsg, "app-1") || !strings.Contains(m.statusMsg, "command not found") {
		t.Errorf("expected status message to mention app-1 and the underlying error, got %q", m.statusMsg)
	}
}

func TestHandlePRStatusKeys_RequestCIFixSetsFailureStatus(t *testing.T) {
	os.Setenv("BV_TEST_MODE", "1")
	defer os.Unsetenv("BV_TEST_MODE")

	m := newTestModel()
	m.prStatus = NewPRStatusModel(m.theme)
	m.prStatus.SetData([]model.Issue{{ID: "app-1", Title: "Fix login", Metadata: map[string]any{"pr_ci_status": "failing"}}})
	m.focused = focusPRStatus

	restoreExec := stubExecCommandErr(fmt.Errorf("label add failed"))
	defer restoreExec()

	m = m.handlePRStatusKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	if !m.statusIsError {
		t.Fatalf("expected error status, got success: %q", m.statusMsg)
	}
	if !strings.Contains(m.statusMsg, "app-1") {
		t.Errorf("expected status message to mention app-1, got %q", m.statusMsg)
	}
}

func TestHandlePRStatusKeys_OpenSetsStatusForMissingURL(t *testing.T) {
	os.Setenv("BV_TEST_MODE", "1")
	defer os.Unsetenv("BV_TEST_MODE")

	m := newTestModel()
	m.prStatus = NewPRStatusModel(m.theme)
	m.prStatus.SetData([]model.Issue{{ID: "app-1", Title: "No PR yet", Metadata: map[string]any{"pr_ci_status": "pending"}}}) // no ExternalRef
	m.focused = focusPRStatus

	m = m.handlePRStatusKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	if !m.statusIsError {
		t.Fatalf("expected error status when no PR URL is set, got success: %q", m.statusMsg)
	}
	if !strings.Contains(m.statusMsg, "app-1") {
		t.Errorf("expected status message to mention app-1, got %q", m.statusMsg)
	}
}

func TestHandlePRStatusKeys_OpenSetsSuccessStatus(t *testing.T) {
	os.Setenv("BV_TEST_MODE", "1")
	defer os.Unsetenv("BV_TEST_MODE")

	m := newTestModel()
	m.prStatus = NewPRStatusModel(m.theme)
	m.prStatus.SetData([]model.Issue{{ID: "app-1", Title: "Fix login", Metadata: map[string]any{"pr_ci_status": "passing"}, ExternalRef: stringPtr("https://github.com/acme/app/pull/1")}})
	m.focused = focusPRStatus

	m = m.handlePRStatusKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	if m.statusIsError {
		t.Fatalf("expected success status, got error: %q", m.statusMsg)
	}
	if !strings.Contains(m.statusMsg, "app-1") {
		t.Errorf("expected status message to mention app-1, got %q", m.statusMsg)
	}
}
