package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"adm/internal/android"
	"adm/internal/i18n"
)

// menuEntry is one row of the main menu.
type menuEntry struct {
	id    string
	label string
	cat   string
}

func (m *Model) menuEntries() []menuEntry {
	var entries []menuEntry
	for _, cat := range m.cats {
		label := i18n.Get("menu.cat." + cat)
		if label == "menu.cat."+cat {
			label = cat
		}
		entries = append(entries, menuEntry{id: "cat:" + cat, label: label, cat: cat})
	}
	entries = append(entries,
		menuEntry{id: "act:switch", label: i18n.Get("menu.switch")},
		menuEntry{id: "act:lang", label: i18n.Get("menu.lang")},
		menuEntry{id: "act:about", label: i18n.Get("menu.about")},
	)
	return entries
}

func (m *Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	entries := m.menuEntries()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirm == confirmQuit {
			switch msg.String() {
			case "y", "enter":
				m.confirm = confirmNone
				m.quitting = true
				return m, tea.Quit
			case "esc", "n":
				m.confirm = confirmNone
			}
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.catIdx > 0 {
				m.catIdx--
			}
		case "down", "j":
			if m.catIdx < len(entries)-1 {
				m.catIdx++
			}
		case "home":
			m.catIdx = 0
		case "end":
			m.catIdx = len(entries) - 1
		case "enter":
			return m.activateMenu(entries)
		}
	}
	return m, nil
}

func (m *Model) activateMenu(entries []menuEntry) (tea.Model, tea.Cmd) {
	if m.catIdx < 0 || m.catIdx >= len(entries) {
		return m, nil
	}
	e := entries[m.catIdx]
	switch e.id {
	case "act:switch":
		m.backTo(screenDevices)
		m.haveDev = false
		return m, m.scanDevices()
	case "act:lang":
		m.push(screenLang)
		m.langIdx = langIndex(m.lang)
		return m, nil
	case "act:about":
		m.push(screenAbout)
		return m, nil
	}

	// A category: build the command list for the current device.
	m.cat = e.cat
	m.list = m.filterCommands(e.cat)
	m.listIdx = 0
	if len(m.list) == 0 && !m.fullToggle {
		m.notice = i18n.Get("cmd.nothing")
		return m, nil
	}
	m.push(screenCommands)
	return m, nil
}

// filterCommands returns the commands of a category, honouring the locked
// bootloader restriction unless the user asked for the full list. It also
// records whether the "full functionality" row should be offered.
func (m *Model) filterCommands(cat string) []android.Command {
	_, grouped := android.ByCategory(m.device.Mode)
	all := grouped[cat]

	// Anything other than an unlocked bootloader can be restricted, and gets
	// the escape hatch row at the bottom of the list.
	m.fullToggle = m.device.Boot != android.BootUnlocked

	out := make([]android.Command, 0, len(all))
	for _, c := range all {
		if m.lockedLimited() && !c.AvailableOn(m.device.Boot) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func langIndex(l i18n.Lang) int {
	for i, v := range i18n.Order {
		if v == l {
			return i
		}
	}
	return 0
}

func (m Model) viewMenu() string {
	entries := m.menuEntries()

	maxW := 0
	for _, e := range entries {
		if n := lipgloss.Width(e.label); n > maxW {
			maxW = n
		}
	}

	var rows []string
	for i, e := range entries {
		sel := i == m.catIdx
		prefix := "  "
		if e.cat == "" {
			prefix = stItemMuted.Render("› ")
		}
		rows = append(rows, prefix+selector(sel, maxW, e.label))
		if i == len(entries)-4 {
			rows = append(rows, "")
		}
	}

	body := []string{
		stHeader.Render(i18n.Get("menu.choose")),
		"",
		m.deviceHeader(),
		"",
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	}

	if notice := m.lockedNotice(); notice != "" {
		body = append(body, "", notice)
	}

	box := boxStyle(m).Render(lipgloss.JoinVertical(lipgloss.Left, body...))

	foot := m.hint(keyCap("↑↓") + " — " + i18n.Get("common.select") +
		"  ·  " + keyCap("enter") + " — " + i18n.Get("common.ok") +
		"  ·  " + keyCap("r") + " — " + i18n.Get("common.refresh") +
		"  ·  " + keyCap("q") + " — " + i18n.Get("common.quit"))

	return lipgloss.JoinVertical(lipgloss.Left,
		header(i18n.Get("menu.title")),
		"",
		box,
		"",
		foot,
		m.quitConfirm(),
	)
}
