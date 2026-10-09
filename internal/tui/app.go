// Package tui implements the terminal interface: the theme, the screen router
// and one file per screen.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"adm/internal/i18n"
)

// Run starts the interface. An empty lang means "ask the user"; a non-empty one
// skips the picker and starts with the environment check.
func Run(initial screen, lang string) (Model, error) {
	m := New(lang)
	m.screen = initial
	if lang != "" {
		m.screen = screenPreflight
	}
	m.stack = nil

	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.prog = p

	raw, err := p.Run()
	if err != nil {
		return m, err
	}
	if final, ok := raw.(*Model); ok {
		return *final, final.err
	}
	return m, nil
}

// RunSetup runs only the environment check: the same preflight screen, prompts
// and install flow the program uses normally, but it quits as soon as adb and
// fastboot are accounted for instead of moving on to the device list.
func RunSetup(lang string) (Model, error) {
	m := New(lang)
	m.setupOnly = true
	m.screen = screenPreflight
	m.stack = nil

	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.prog = p

	raw, err := p.Run()
	if err != nil {
		return m, err
	}
	if final, ok := raw.(*Model); ok {
		return *final, final.err
	}
	return m, nil
}

// typing reports whether a text field currently owns the keyboard. Global keys
// must stand aside, otherwise typing a host like 10.0.0.4 would trigger the
// rescan shortcut and typing q would quit the program.
func (m Model) typing() bool {
	if m.tcpMode != tcpClosed {
		return true
	}
	return m.screen == screenArgs && m.ti.Focused()
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	switch m.screen {
	case screenLang:
		return nil
	case screenPreflight:
		return m.pre.start()
	case screenDevices:
		if !m.devLoadedInit() {
			return m.scanDevices()
		}
		return nil
	case screenOutput:
		return m.run.tick()
	}
	return nil
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Global keys work everywhere except while typing in a field.
	if !m.typing() {
		if handled, cmd := m.globalKeys(msg); handled {
			return m, cmd
		}
	}

	switch msg := msg.(type) {
	case guardDoneMsg:
		return m.handleGuard(msg)
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.run.resize(m.w, m.h)
		return m, nil
	}

	switch m.screen {
	case screenLang:
		return m.updateLang(msg)
	case screenPreflight:
		return m.updatePreflight(msg)
	case screenDevices:
		return m.updateDevices(msg)
	case screenMenu:
		return m.updateMenu(msg)
	case screenCommands:
		return m.updateCommands(msg)
	case screenArgs:
		return m.updateArgs(msg)
	case screenFile:
		return m.updateFile(msg)
	case screenOutput:
		return m.updateOutput(msg)
	}
	return m, nil
}

// globalKeys handles quit, cancel and rescan across all screens.
func (m *Model) globalKeys(msg tea.Msg) (bool, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return false, nil
	}

	switch key.String() {
	case "ctrl+c":
		m.quitting = true
		return true, tea.Quit
	}

	// A refusal from a command guard is modal: the first key closes it.
	if m.blocked != "" {
		m.blocked = ""
		return true, nil
	}

	// While a text field has focus every other key belongs to the field.
	if m.typing() {
		return false, nil
	}

	switch key.String() {
	case "q":
		m.quitting = true
		return true, tea.Quit

	case "esc":
		if m.confirm != confirmNone {
			m.confirm = confirmNone
			return true, nil
		}
		switch m.screen {
		case screenPreflight, screenDevices, screenMenu:
			m.confirm = confirmQuit
			return true, nil
		case screenOutput:
			if m.run.running {
				return true, nil // let the command finish
			}
		}
		m.pop()
		return true, nil

	case "ctrl+r":
		switch m.screen {
		case screenDevices, screenMenu, screenCommands:
			m.backTo(screenDevices)
			m.devLoadedInit()
			return true, m.scanDevices()
		}
	}
	return false, nil
}

// View implements tea.Model.
func (m *Model) View() string {
	if m.quitting {
		return ""
	}
	if m.w == 0 || m.h == 0 {
		return "…"
	}

	var body string
	switch m.screen {
	case screenLang:
		body = m.viewLang()
	case screenPreflight:
		body = m.viewPreflight()
	case screenDevices:
		body = m.viewDevices()
	case screenMenu:
		body = m.viewMenu()
	case screenCommands:
		body = m.viewCommands()
	case screenArgs:
		body = m.viewArgs()
	case screenFile:
		body = m.viewFile()
	case screenOutput:
		body = m.viewOutput()
	case screenAbout:
		body = m.viewAbout()
	}

	out := body
	if n := m.notice; n != "" {
		out = lipgloss.JoinVertical(lipgloss.Left, body, "", stWarn.Render(n))
	}
	if m.err != nil {
		out = lipgloss.JoinVertical(lipgloss.Left, out, "", stErr.Render(i18n.T("err.title")+": "+m.err.Error()))
	}

	// Keep the frame inside the terminal.
	if h := lipgloss.Height(out); m.h > 0 && h > m.h {
		out = lipgloss.NewStyle().MaxHeight(m.h).Render(out)
	}
	return out
}
