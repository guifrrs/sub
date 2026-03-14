package ui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"sub/internal/model"
)

func TestSearchWindowShowsAtMostFiveAndScrolls(t *testing.T) {
	m := NewModel(nil)
	m.titles = sampleTitles(8)
	m.cursor = 0

	for range 6 {
		updated, _ := m.updateSearchState(keyDownMsg())
		m = updated.(Model)
	}

	view := stripANSI(m.View())
	if strings.Count(view, "[MOVIE]") > 5 {
		t.Fatalf("expected at most 5 items rendered, got view:\n%s", view)
	}

	if strings.Contains(view, "Title 01") {
		t.Fatalf("expected first title to scroll out of view, got view:\n%s", view)
	}

	if !strings.Contains(view, "Title 07") {
		t.Fatalf("expected current cursor title to be visible, got view:\n%s", view)
	}
}

func TestSearchNavigationDoesNotWrapAtBounds(t *testing.T) {
	m := NewModel(nil)
	m.titles = []model.Title{
		{Name: "Alpha", Type: model.TitleTypeMovie},
		{Name: "Beta", Type: model.TitleTypeSeries},
	}
	m.cursor = 1

	updated, _ := m.updateSearchState(keyDownMsg())
	m = updated.(Model)

	if m.cursor != 1 {
		t.Fatalf("expected cursor to stay at last index, got %d", m.cursor)
	}

	updated, _ = m.updateSearchState(keyUpMsg())
	m = updated.(Model)
	updated, _ = m.updateSearchState(keyUpMsg())
	m = updated.(Model)

	if m.cursor != 0 {
		t.Fatalf("expected cursor to stay at first index, got %d", m.cursor)
	}
}

func TestSearchTypingJKeepsInputEditing(t *testing.T) {
	m := NewModel(nil)
	m.titles = []model.Title{{Name: "Alpha"}, {Name: "Beta"}}
	m.cursor = 1

	updated, _ := m.updateSearchState(keyRuneMsg('j'))
	m = updated.(Model)

	if m.input.Value() != "j" {
		t.Fatalf("expected input to receive typed j, got %q", m.input.Value())
	}

	if m.cursor != 0 {
		t.Fatalf("expected cursor reset from query update, got %d", m.cursor)
	}
}

func TestClearingInputInvalidatesPendingDebounce(t *testing.T) {
	m := NewModel(nil)
	m.input.SetValue("ju")
	m.debounceToken = 1

	updated, _ := m.updateSearchState(keyBackspaceMsg())
	m = updated.(Model)

	if m.input.Value() != "j" {
		t.Fatalf("expected input to be cleared to one character, got %q", m.input.Value())
	}

	updated, cmd := m.handleDebounce(debounceSearchMsg{token: 1, query: "ju"})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("expected stale debounce command to be ignored")
	}

	if m.loading {
		t.Fatalf("expected loading to remain false after clearing input")
	}
}

func sampleTitles(total int) []model.Title {
	titles := make([]model.Title, 0, total)
	for i := 1; i <= total; i++ {
		titles = append(titles, model.Title{
			Name: fmt.Sprintf("Title %02d", i),
			Type: model.TitleTypeMovie,
		})
	}

	return titles
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func keyDownMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyDown}
}

func keyUpMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyUp}
}

func keyBackspaceMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyBackspace}
}

func keyRuneMsg(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}
