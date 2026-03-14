package ui

import (
	"context"
	"fmt"
	"sort"
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
	stateSeasons
	stateEpisodes
	stateSubtitles
)

type Model struct {
	input   textinput.Model
	spinner spinner.Model
	service *service.SubtitleService

	state   screenState
	loading bool

	titles      []model.Title
	seasons     []int
	allEpisodes []model.Episode
	episodes    []model.Episode
	subtitles   []model.Subtitle

	activeTitle   model.Title
	hasTitle      bool
	activeSeason  int
	activeEpisode model.Episode
	hasEpisode    bool

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

type episodesResultMsg struct {
	episodes []model.Episode
	err      error
}

type subtitlesResultMsg struct {
	subtitles []model.Subtitle
	err       error
}

type downloadResultMsg struct {
	result model.DownloadResult
	err    error
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
	case episodesResultMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			m.statusMessage = "Could not load episodes."
			return m, nil
		}

		m.errorMessage = ""
		m.allEpisodes = msg.episodes
		m.cursor = 0
		if len(m.allEpisodes) == 0 {
			m.statusMessage = "No episodes found for this series."
			m.state = stateSearch
			m.seasons = nil
			m.episodes = nil
			return m, nil
		}

		m.seasons = collectSeasons(m.allEpisodes)
		if len(m.seasons) == 0 {
			m.episodes = m.allEpisodes
			m.activeSeason = 0
			m.state = stateEpisodes
			m.statusMessage = "Select an episode and press enter."
			return m, nil
		}

		m.activeSeason = 0
		m.episodes = nil
		m.state = stateSeasons
		m.statusMessage = "Select a season and press enter."
		return m, nil
	case subtitlesResultMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			m.statusMessage = "Could not load subtitles."
			return m, nil
		}

		m.errorMessage = ""
		m.subtitles = msg.subtitles
		m.cursor = 0
		m.state = stateSubtitles
		if len(m.subtitles) == 0 {
			m.statusMessage = "No subtitles found for this selection."
			if m.hasTitle && m.activeTitle.IsSeries() {
				m.state = stateEpisodes
			} else {
				m.state = stateSearch
			}
			return m, nil
		}

		m.statusMessage = "Select a subtitle and press enter to download."
		return m, nil
	case downloadResultMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			m.statusMessage = "Download failed."
			return m, nil
		}

		m.errorMessage = ""
		m.statusMessage = fmt.Sprintf("Downloaded %d subtitle file(s) to %s.", len(msg.result.ExtractedPaths), msg.result.OutputDir)
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancelSearch()
			return m, tea.Quit
		case "esc":
			if m.state == stateSearch {
				m.cancelSearch()
				return m, tea.Quit
			}
			return m.navigateBack()
		}

		if m.loading {
			return m, nil
		}

		switch m.state {
		case stateSearch:
			return m.updateSearchState(msg)
		case stateSeasons:
			return m.updateSeasonState(msg)
		case stateEpisodes:
			return m.updateEpisodeState(msg)
		case stateSubtitles:
			return m.updateSubtitleState(msg)
		}
	}

	return m, nil
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

	switch m.state {
	case stateSearch:
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
	case stateSeasons:
		b.WriteString(sectionTitleStyle.Render("Seasons"))
		b.WriteByte('\n')
		start, end := visibleWindowBounds(len(m.seasons), m.cursor)
		for i := start; i < end; i++ {
			line := fmt.Sprintf("Season %d", m.seasons[i])
			b.WriteString(renderSelectableLine(i == m.cursor, line))
			b.WriteByte('\n')
		}
		if len(m.seasons) == 0 {
			b.WriteString(hintStyle.Render("  (no seasons loaded)"))
			b.WriteByte('\n')
		} else if len(m.seasons) > maxVisibleItems {
			b.WriteByte('\n')
			b.WriteString(hintStyle.Render(fmt.Sprintf("Showing %d-%d of %d.", start+1, end, len(m.seasons))))
			b.WriteByte('\n')
		}
	case stateEpisodes:
		b.WriteString(sectionTitleStyle.Render("Episodes"))
		b.WriteByte('\n')
		start, end := visibleWindowBounds(len(m.episodes), m.cursor)
		for i := start; i < end; i++ {
			episode := m.episodes[i]
			line := fmt.Sprintf("S%02dE%02d - %s", episode.Season, episode.Number, episode.EpisodeName)
			b.WriteString(renderSelectableLine(i == m.cursor, line))
			b.WriteByte('\n')
		}
		if len(m.episodes) == 0 {
			b.WriteString(hintStyle.Render("  (no episodes loaded)"))
			b.WriteByte('\n')
		} else if len(m.episodes) > maxVisibleItems {
			b.WriteByte('\n')
			b.WriteString(hintStyle.Render(fmt.Sprintf("Showing %d-%d of %d.", start+1, end, len(m.episodes))))
			b.WriteByte('\n')
		}
	case stateSubtitles:
		b.WriteString(sectionTitleStyle.Render("Subtitles"))
		b.WriteByte('\n')
		start, end := visibleWindowBounds(len(m.subtitles), m.cursor)
		for i := start; i < end; i++ {
			sub := m.subtitles[i]
			trusted := "no"
			if sub.Trusted {
				trusted = "yes"
			}

			releaseName := strings.TrimSpace(sub.ReleaseName)
			if releaseName == "" {
				releaseName = "(no release name)"
			}

			line := fmt.Sprintf("#%d %s (%s) downloads=%d trusted=%s", sub.ID, releaseName, sub.Format, sub.Downloads, trusted)
			b.WriteString(renderSelectableLine(i == m.cursor, line))
			b.WriteByte('\n')
		}
		if len(m.subtitles) == 0 {
			b.WriteString(hintStyle.Render("  (no subtitles loaded)"))
			b.WriteByte('\n')
		} else if len(m.subtitles) > maxVisibleItems {
			b.WriteByte('\n')
			b.WriteString(hintStyle.Render(fmt.Sprintf("Showing %d-%d of %d.", start+1, end, len(m.subtitles))))
			b.WriteByte('\n')
		}
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
		m.hasTitle = true
		m.activeTitle = selected
		m.activeSeason = 0
		m.hasEpisode = false
		m.seasons = nil
		m.allEpisodes = nil
		m.episodes = nil
		m.subtitles = nil
		m.errorMessage = ""
		m.loading = true
		m.cancelSearch()

		if selected.IsSeries() {
			m.statusMessage = ""
			return m, tea.Batch(listEpisodesCmd(m.service, selected), m.spinner.Tick)
		}

		m.statusMessage = ""
		return m, tea.Batch(listSubtitlesForTitleCmd(m.service, selected), m.spinner.Tick)
	}

	previousQuery := m.input.Value()
	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)

	if m.input.Value() == previousQuery {
		return m, inputCmd
	}

	m.state = stateSearch
	m.seasons = nil
	m.allEpisodes = nil
	m.episodes = nil
	m.subtitles = nil
	m.activeSeason = 0
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

func (m Model) updateSeasonState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		m.moveCursor(-1, len(m.seasons))
		return m, nil
	case "down":
		m.moveCursor(1, len(m.seasons))
		return m, nil
	case "enter":
		if len(m.seasons) == 0 {
			return m, nil
		}

		season := m.seasons[m.cursor]
		episodes := filterEpisodesBySeason(m.allEpisodes, season)
		if len(episodes) == 0 {
			m.statusMessage = fmt.Sprintf("No episodes found for season %d.", season)
			return m, nil
		}

		m.activeSeason = season
		m.episodes = episodes
		m.cursor = 0
		m.errorMessage = ""
		m.state = stateEpisodes
		m.statusMessage = fmt.Sprintf("Season %d selected. Choose an episode and press enter.", season)
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) updateEpisodeState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		m.moveCursor(-1, len(m.episodes))
		return m, nil
	case "down":
		m.moveCursor(1, len(m.episodes))
		return m, nil
	case "enter":
		if len(m.episodes) == 0 {
			return m, nil
		}

		selected := m.episodes[m.cursor]
		m.activeEpisode = selected
		m.hasEpisode = true
		m.errorMessage = ""
		m.loading = true
		m.statusMessage = ""

		return m, tea.Batch(listSubtitlesForEpisodeCmd(m.service, selected), m.spinner.Tick)
	default:
		return m, nil
	}
}

func (m Model) updateSubtitleState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		m.moveCursor(-1, len(m.subtitles))
		return m, nil
	case "down":
		m.moveCursor(1, len(m.subtitles))
		return m, nil
	case "enter":
		if len(m.subtitles) == 0 {
			return m, nil
		}

		selected := m.subtitles[m.cursor]
		m.loading = true
		m.errorMessage = ""
		m.statusMessage = ""

		return m, tea.Batch(downloadSubtitleCmd(m.service, selected), m.spinner.Tick)
	default:
		return m, nil
	}
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

func maxInt(a int, b int) int {
	if a > b {
		return a
	}

	return b
}

func collectSeasons(episodes []model.Episode) []int {
	if len(episodes) == 0 {
		return nil
	}

	seen := make(map[int]struct{})
	seasons := make([]int, 0)
	for _, episode := range episodes {
		if episode.Season <= 0 {
			continue
		}

		if _, ok := seen[episode.Season]; ok {
			continue
		}

		seen[episode.Season] = struct{}{}
		seasons = append(seasons, episode.Season)
	}

	sort.Ints(seasons)
	return seasons
}

func filterEpisodesBySeason(episodes []model.Episode, season int) []model.Episode {
	if len(episodes) == 0 {
		return nil
	}

	filtered := make([]model.Episode, 0)
	for _, episode := range episodes {
		if episode.Season == season {
			filtered = append(filtered, episode)
		}
	}

	return filtered
}

func (m Model) navigateBack() (tea.Model, tea.Cmd) {
	m.errorMessage = ""
	switch m.state {
	case stateSubtitles:
		if m.hasTitle && m.activeTitle.IsSeries() {
			m.state = stateEpisodes
			if len(m.episodes) > 0 {
				m.statusMessage = "Back to episodes."
			} else {
				m.state = stateSeasons
				m.statusMessage = "Back to seasons."
			}
		} else {
			m.state = stateSearch
			m.statusMessage = "Back to search results."
		}
		m.cursor = 0
	case stateEpisodes:
		if len(m.seasons) > 0 {
			m.state = stateSeasons
			m.statusMessage = "Back to seasons."
		} else {
			m.state = stateSearch
			m.statusMessage = "Back to search results."
		}
		m.cursor = 0
	case stateSeasons:
		m.state = stateSearch
		m.statusMessage = "Back to search results."
		m.cursor = 0
	}

	return m, nil
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

func listEpisodesCmd(app *service.SubtitleService, series model.Title) tea.Cmd {
	return func() tea.Msg {
		if app == nil {
			return episodesResultMsg{err: fmt.Errorf("service is not configured")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		episodes, err := app.ListEpisodes(ctx, series)
		return episodesResultMsg{episodes: episodes, err: err}
	}
}

func listSubtitlesForTitleCmd(app *service.SubtitleService, title model.Title) tea.Cmd {
	return func() tea.Msg {
		if app == nil {
			return subtitlesResultMsg{err: fmt.Errorf("service is not configured")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		subtitles, err := app.ListSubtitlesForTitle(ctx, title)
		return subtitlesResultMsg{subtitles: subtitles, err: err}
	}
}

func listSubtitlesForEpisodeCmd(app *service.SubtitleService, episode model.Episode) tea.Cmd {
	return func() tea.Msg {
		if app == nil {
			return subtitlesResultMsg{err: fmt.Errorf("service is not configured")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		subtitles, err := app.ListSubtitlesForEpisode(ctx, episode)
		return subtitlesResultMsg{subtitles: subtitles, err: err}
	}
}

func downloadSubtitleCmd(app *service.SubtitleService, subtitle model.Subtitle) tea.Cmd {
	return func() tea.Msg {
		if app == nil {
			return downloadResultMsg{err: fmt.Errorf("service is not configured")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := app.DownloadSubtitle(ctx, subtitle)
		return downloadResultMsg{result: result, err: err}
	}
}
