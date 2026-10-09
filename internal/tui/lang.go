package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"adm/internal/i18n"
)

func (m *Model) updateLang(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k", "ctrl+p":
			if m.langIdx > 0 {
				m.langIdx--
			}
		case "down", "j", "ctrl+n":
			if m.langIdx < len(i18n.Order)-1 {
				m.langIdx++
			}
		case "home", "g":
			m.langIdx = 0
		case "end", "G":
			m.langIdx = len(i18n.Order) - 1
		case "enter", " ":
			m.lang = i18n.Order[m.langIdx]
			m.applyLanguage()
			m.screen = screenPreflight
			return m, m.pre.start()
		case "esc":
			// Nothing behind the language screen, so treat esc as quit.
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) viewLang() string {
	var rows []string
	for i, l := range i18n.Order {
		name := i18n.Names[l]
		mark := "  "
		if i == m.langIdx {
			mark = stCursor.Render("▸ ")
		}
		hint := ""
		if l == m.lang {
			hint = "  " + stOK.Render("•")
		}
		rows = append(rows, mark+name+hint)
	}

	list := lipgloss.JoinVertical(lipgloss.Left, rows...)

	box := accentBox(m).Render(lipgloss.JoinVertical(lipgloss.Left,
		stHeader.Render(i18n.Get("lang.title")),
		"",
		list,
	))

	foot := m.hint(i18n.Get("lang.hint"))

	return lipgloss.JoinVertical(lipgloss.Left,
		center(m.w, m.h-lipgloss.Height(foot)-2, lipgloss.JoinVertical(lipgloss.Center,
			stTitle.Render(appName),
			stSubtitle.Render(i18n.Get("app.tagline")),
			"",
			box,
		)),
		"",
		foot,
	)
}

func (m Model) viewAbout() string {
	body := fmt.Sprintf(i18n.Get("about.body"), version)
	author := i18n.T("about.author", author)

	box := boxStyle(m).Render(lipgloss.JoinVertical(lipgloss.Left,
		stTitle.Render(appName),
		stLabel.Render(author),
		"",
		stValue.Render(wrapText(body, m.w-10)),
	))
	return lipgloss.JoinVertical(lipgloss.Left,
		header(i18n.Get("menu.about")),
		"",
		box,
		"",
		m.hint(keyCap("esc")+" — "+i18n.Get("common.back")+
			"  ·  "+keyCap("q")+" — "+i18n.Get("common.quit")),
	)
}
