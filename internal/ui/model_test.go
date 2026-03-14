package ui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"sub/internal/model"
)

func TestEpisodesWindowShowsAtMostFiveAndScrolls(t *testing.T) {
	m := NewModel(nil)
	m.state = stateEpisodes
	m.episodes = sampleEpisodes(8)
	m.cursor = 0

	for range 6 {
		updated, _ := m.updateEpisodeState(keyDownMsg())
		m = updated.(Model)
	}

	view := stripANSI(m.View())
	if strings.Count(view, "S01E") > 5 {
		t.Fatalf("expected at most 5 episodes rendered, got view:\n%s", view)
	}

	if strings.Contains(view, "S01E01") {
		t.Fatalf("expected first episode to scroll out of view, got view:\n%s", view)
	}

	if !strings.Contains(view, "S01E07") {
		t.Fatalf("expected current cursor episode to be visible, got view:\n%s", view)
	}
}

func TestSeriesSelectionShouldPromptSeasonStep(t *testing.T) {
	m := NewModel(nil)
	m.hasTitle = true
	m.activeTitle = model.Title{Name: "Some Series", Type: model.TitleTypeSeries}

	updated, _ := m.Update(episodesResultMsg{episodes: []model.Episode{
		{Season: 1, Number: 1, EpisodeName: "Pilot"},
		{Season: 2, Number: 1, EpisodeName: "Back Again"},
	}})
	m = updated.(Model)

	if !strings.Contains(strings.ToLower(m.statusMessage), "season") {
		t.Fatalf("expected status to prompt season selection, got: %q", m.statusMessage)
	}

	view := strings.ToLower(stripANSI(m.View()))
	if !strings.Contains(view, "seasons") {
		t.Fatalf("expected seasons step in view, got:\n%s", view)
	}

	updated, _ = m.updateSeasonState(keyDownMsg())
	m = updated.(Model)
	updated, _ = m.updateSeasonState(keyEnterMsg())
	m = updated.(Model)

	if m.state != stateEpisodes {
		t.Fatalf("expected season selection to advance to episodes, got state %v", m.state)
	}

	if len(m.episodes) != 1 || m.episodes[0].Season != 2 {
		t.Fatalf("expected only selected season episodes, got %#v", m.episodes)
	}
}

func TestSearchNavigationDoesNotWrapAtBounds(t *testing.T) {
	m := NewModel(nil)
	m.state = stateSearch
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
	m.state = stateSearch
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
	m.state = stateSearch
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

func sampleEpisodes(total int) []model.Episode {
	episodes := make([]model.Episode, 0, total)
	for i := 1; i <= total; i++ {
		episodes = append(episodes, model.Episode{
			Season:      1,
			Number:      i,
			EpisodeName: fmt.Sprintf("Ep %d", i),
		})
	}

	return episodes
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

func keyEnterMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

func keyBackspaceMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyBackspace}
}

func keyRuneMsg(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}
