package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"sub/internal/opensubtitles"
	"sub/internal/service"
	"sub/internal/store"
	"sub/internal/ui"
)

func main() {
	client := opensubtitles.NewClient()
	localStore := store.NewLocalStore()
	app := service.NewSubtitleService(client, client, localStore)

	p := tea.NewProgram(ui.NewModel(app))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
