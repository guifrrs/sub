package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"sub/internal/model"
	"sub/internal/service"
)

const (
	searchDebounceDelay = 500 * time.Millisecond
	maxVisibleItems     = 5
)

type screenState int

const (
	stateSearch screenState = iota
)

type Model struct {
	input   textinput.Model
	spinner spinner.Model
	service *service.SubtitleService

	state   screenState
	loading bool

	titles []model.Title
	cursor int

	statusMessage string
	errorMessage  string

	debounceToken int
	requestID     int
	searchCancel  context.CancelFunc
}

type debounceSearchMsg struct {
	token int
	query string
}

type searchResultMsg struct {
	requestID int
	titles    []model.Title
	err       error
}

func NewModel(app *service.SubtitleService) Model {
	input := textinput.New()
	input.Focus()
	input.Placeholder = "Search subtitles..."
	input.CharLimit = 120
	input.Width = 52

	loadingSpinner := spinner.New()
	loadingSpinner.Spinner = spinner.Dot
	loadingSpinner.Style = loadingStyle

	return Model{
		input:         input,
		spinner:       loadingSpinner,
		service:       app,
		state:         stateSearch,
		statusMessage: fmt.Sprintf("Type at least %d characters to search.", service.MinSearchQueryLength),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case debounceSearchMsg:
		return m.handleDebounce(msg)
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}

		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case searchResultMsg:
		return m.handleSearchResult(msg)
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "ctrl+c", "esc":
		m.cancelSearch()
		return m, tea.Quit
	}

	if m.loading {
		return m, nil
	}

	return m.updateSearchState(keyMsg)
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(appTitleStyle.Render("sub"))
	b.WriteString("\n\n")
	b.WriteString(inputBoxStyle.Render(m.input.View()))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString(m.spinner.View())
		b.WriteString("\n\n")
	}

	b.WriteString(sectionTitleStyle.Render("Search results"))
	b.WriteByte('\n')

	start, end := visibleWindowBounds(len(m.titles), m.cursor)
	for i := start; i < end; i++ {
		title := m.titles[i]
		kind := movieTagStyle.Render("MOVIE")
		if title.IsSeries() {
			kind = seriesTagStyle.Render("SERIES")
		}

		line := fmt.Sprintf("[%s] %s", kind, title.Name)
		if title.Year > 0 {
			line += fmt.Sprintf(" (%d)", title.Year)
		}

		b.WriteString(renderSelectableLine(i == m.cursor, line))
		b.WriteByte('\n')
	}

	if len(m.titles) == 0 {
		b.WriteString(hintStyle.Render("  (no results yet)"))
		b.WriteByte('\n')
	} else if len(m.titles) > maxVisibleItems {
		b.WriteByte('\n')
		b.WriteString(hintStyle.Render(fmt.Sprintf("Showing %d-%d of %d.", start+1, end, len(m.titles))))
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	if m.errorMessage != "" {
		b.WriteString(errorStyle.Render("Error: " + m.errorMessage))
		b.WriteByte('\n')
	}
	if m.statusMessage != "" {
		b.WriteString(statusStyle.Render("Status: " + m.statusMessage))
		b.WriteByte('\n')
	}
	b.WriteString(hintStyle.Render("Press ctrl+c to quit."))
	b.WriteByte('\n')

	return b.String()
}

func (m Model) updateSearchState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		m.moveCursor(-1, len(m.titles))
		return m, nil
	case "down":
		m.moveCursor(1, len(m.titles))
		return m, nil
	case "enter":
		if len(m.titles) == 0 {
			return m, nil
		}

		selected := m.titles[m.cursor]
		m.errorMessage = ""
		m.statusMessage = fmt.Sprintf("Selected %q. Next steps (series/episode/subtitle) will come in follow-up PRs.", selected.Name)
		return m, nil
	}

	previousQuery := m.input.Value()
	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)

	if m.input.Value() == previousQuery {
		return m, inputCmd
	}

	m.cursor = 0
	m.errorMessage = ""

	query := strings.TrimSpace(m.input.Value())
	if len(query) < service.MinSearchQueryLength {
		m.cancelSearch()
		m.debounceToken++
		m.requestID++
		m.loading = false
		m.titles = nil
		m.statusMessage = fmt.Sprintf("Type at least %d characters to search.", service.MinSearchQueryLength)
		return m, inputCmd
	}

	m.debounceToken++
	m.statusMessage = "Waiting for debounce before searching..."

	return m, tea.Batch(inputCmd, debounceSearchCmd(query, m.debounceToken))
}

func (m Model) handleDebounce(msg debounceSearchMsg) (tea.Model, tea.Cmd) {
	if msg.token != m.debounceToken || m.state != stateSearch {
		return m, nil
	}

	query := strings.TrimSpace(msg.query)
	if len(query) < service.MinSearchQueryLength {
		return m, nil
	}

	if query != strings.TrimSpace(m.input.Value()) {
		return m, nil
	}

	m.cancelSearch()
	m.requestID++
	requestID := m.requestID

	ctx, cancel := context.WithCancel(context.Background())
	m.searchCancel = cancel
	m.loading = true
	m.errorMessage = ""
	m.statusMessage = ""

	return m, tea.Batch(searchTitlesCmd(ctx, m.service, requestID, query), m.spinner.Tick)
}

func (m Model) handleSearchResult(msg searchResultMsg) (tea.Model, tea.Cmd) {
	if msg.requestID != m.requestID {
		return m, nil
	}

	m.searchCancel = nil
	m.loading = false
	if msg.err != nil {
		m.errorMessage = msg.err.Error()
		m.statusMessage = "Could not fetch search results."
		return m, nil
	}

	m.errorMessage = ""
	m.titles = msg.titles
	m.cursor = 0
	if len(m.titles) == 0 {
		m.statusMessage = "No titles found for this query."
		return m, nil
	}

	m.statusMessage = fmt.Sprintf("Found %d result(s).", len(m.titles))
	return m, nil
}

func (m *Model) moveCursor(delta int, size int) {
	if size <= 0 {
		m.cursor = 0
		return
	}

	next := m.cursor + delta
	if next < 0 {
		next = 0
	}
	if next >= size {
		next = size - 1
	}

	m.cursor = next
}

func renderSelectableLine(selected bool, content string) string {
	if selected {
		return selectedItemStyle.Render("> " + content)
	}

	return "  " + content
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}

	return b
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}

	return b
}

func visibleWindowBounds(total int, cursor int) (start int, end int) {
	if total <= 0 {
		return 0, 0
	}

	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}

	start = 0
	if cursor >= maxVisibleItems {
		start = cursor - maxVisibleItems + 1
	}

	end = minInt(start+maxVisibleItems, total)
	if end-start < maxVisibleItems && start > 0 {
		start = maxInt(0, end-maxVisibleItems)
	}

	return start, end
}

func (m *Model) cancelSearch() {
	if m.searchCancel != nil {
		m.searchCancel()
		m.searchCancel = nil
	}
}

func debounceSearchCmd(query string, token int) tea.Cmd {
	return tea.Tick(searchDebounceDelay, func(time.Time) tea.Msg {
		return debounceSearchMsg{token: token, query: query}
	})
}

func searchTitlesCmd(parent context.Context, app *service.SubtitleService, requestID int, query string) tea.Cmd {
	return func() tea.Msg {
		if app == nil {
			return searchResultMsg{requestID: requestID, err: fmt.Errorf("service is not configured")}
		}

		ctx, cancel := context.WithTimeout(parent, 12*time.Second)
		defer cancel()

		titles, err := app.SearchTitles(ctx, query)
		return searchResultMsg{
			requestID: requestID,
			titles:    titles,
			err:       err,
		}
	}
}
