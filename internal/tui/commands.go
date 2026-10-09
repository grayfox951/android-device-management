package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"adm/internal/android"
	"adm/internal/i18n"
)

// toggleRow is the index of the "full functionality" row when it is present.
func (m Model) toggleRow() int {
	if !m.fullToggle {
		return -1
	}
	return len(m.list)
}

func (m *Model) updateCommands(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	// A pending warning swallows every other key until it is answered.
	if m.confirm != confirmNone {
		switch key.String() {
		case "y", "enter":
			m.confirm = confirmNone
			return m.beginArgs(m.pending)
		case "esc", "n":
			m.confirm = confirmNone
			return m, nil
		}
		return m, nil
	}

	last := len(m.list)
	if m.fullToggle {
		last++
	}

	switch key.String() {
	case "up", "k":
		if m.listIdx > 0 {
			m.listIdx--
		}
	case "down", "j":
		if m.listIdx < last-1 {
			m.listIdx++
		}
	case "home":
		m.listIdx = 0
	case "end":
		m.listIdx = last - 1
	case "f":
		// Toggle the restricted list on a locked bootloader.
		if m.device.Boot != android.BootUnlocked {
			return m.toggleFull()
		}
	case "esc":
		m.pop()
	case "enter":
		if m.fullToggle && m.listIdx == len(m.list) {
			return m.toggleFull()
		}
		return m.runSelected()
	}
	return m, nil
}

// toggleFull switches between the safe list and the unrestricted one, keeping
// the same category selected.
func (m *Model) toggleFull() (tea.Model, tea.Cmd) {
	m.showFull = !m.showFull
	m.list = m.filterCommands(m.cat)
	m.listIdx = 0
	return m, nil
}

// runSelected validates the chosen command and moves on to the right next step.
func (m *Model) runSelected() (tea.Model, tea.Cmd) {
	if m.listIdx < 0 || m.listIdx >= len(m.list) {
		return m, nil
	}
	c := m.list[m.listIdx]
	m.pending = c
	m.notice = ""

	// A locked bootloader gets a warning before anything that normally needs
	// an unlocked one, and a second one for destructive commands.
	if m.confirmsLocked(c) {
		m.confirm = confirmLocked
		return m, nil
	}
	if c.Risk {
		m.confirm = confirmRisk
		return m, nil
	}
	return m.beginArgs(c)
}

// beginArgs starts argument collection, or jumps straight to running when the
// command takes no input.
func (m *Model) beginArgs(c android.Command) (tea.Model, tea.Cmd) {
	m.pending = c
	m.argVals = map[string]string{}
	m.argIdx = 0
	m.pickingCh = false

	if !c.NeedsArgs() {
		return m.launch(c)
	}
	m.push(screenArgs)

	arg := c.Args[0]
	m.ti.Placeholder = arg.Label
	m.ti.SetValue(arg.Default)
	if arg.Kind == android.ArgChoice {
		m.pickingCh = true
		m.chIdx = choiceIndex(arg, arg.Default)
		return m, nil
	}
	m.pickingCh = false
	return m, m.ti.Focus()
}

func choiceIndex(a android.Arg, def string) int {
	for i, c := range a.Choices {
		if c.Value == def {
			return i
		}
	}
	return 0
}

func (m Model) viewCommands() string {
	maxLabel := 0
	for _, c := range m.list {
		if n := lipgloss.Width(c.Label); n > maxLabel {
			maxLabel = n
		}
	}
	// Long command labels must not crowd out the description on a narrow
	// terminal, so they share the space and are truncated if needed.
	labelW := clamp(maxLabel, 16, m.w/2)
	descW := m.listDescWidth(labelW)

	var rows []string
	for i, c := range m.list {
		sel := i == m.listIdx

		// A fixed two cell prefix keeps every label on the same column.
		mark := " "
		switch {
		case c.Root:
			mark = "▲"
		case c.Risk:
			mark = "!"
		}
		switch mark {
		case "▲":
			mark = stDanger.Render(mark)
		case "!":
			mark = stWarn.Render(mark)
		}

		label := selector(sel, labelW, truncate(c.Label, labelW))
		desc := i18n.Get(c.DescKey)
		if desc == c.DescKey {
			desc = ""
		}

		line := mark + " " + label
		if desc != "" {
			line += "  " + stItemMuted.Render("— "+truncate(desc, descW))
		}
		rows = append(rows, line)
	}

	// The escape hatch is a real, selectable row rather than a hint, so a
	// locked device is never a dead end.
	if m.fullToggle {
		rows = append(rows, "")
		label := i18n.Get("cmd.locked.unlock_all")
		if m.showFull {
			label = i18n.Get("cmd.locked.back_safe")
		}
		rows = append(rows, stWarn.Render("⚠ ")+
			selector(m.listIdx == len(m.list), maxLabel+4, label))
	}

	title := i18n.T("cmd.list", i18n.Get("menu.cat."+m.cat))
	body := []string{
		stHeader.Render(title),
		m.deviceHeader(),
		"",
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	}

	if len(m.list) == 0 && m.fullToggle {
		body = append(body, "", stItemMuted.Render(i18n.Get("cmd.locked.notice")))
	}

	if blocked := m.blockedBox(); blocked != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			header(i18n.Get("cmd.title")),
			"",
			blocked,
		)
	}

	if m.confirm != confirmNone {
		// Showing a second bordered box inside the list breaks the frame, so a
		// pending question replaces the list until it is answered.
		return lipgloss.JoinVertical(lipgloss.Left,
			header(i18n.Get("cmd.title")),
			"",
			m.confirmBox(),
			"",
			m.hint(keyCap("y")+" — "+i18n.Get("common.yes")+
				"  ·  "+keyCap("esc")+" — "+i18n.Get("common.no")),
		)
	}

	foot := keyCap("↑↓") + " — " + i18n.Get("common.select") +
		"  ·  " + keyCap("enter") + " — " + i18n.Get("common.run")
	if m.device.Boot != android.BootUnlocked {
		foot += "  ·  " + keyCap("f") + " — " + toggleLabel(m.showFull)
	}
	foot += "  ·  " + keyCap("esc") + " — " + i18n.Get("common.back")

	return lipgloss.JoinVertical(lipgloss.Left,
		header(i18n.Get("cmd.title")),
		"",
		boxStyle(m).Render(lipgloss.JoinVertical(lipgloss.Left, body...)),
		"",
		m.hint(foot),
	)
}

func toggleLabel(full bool) string {
	if full {
		return i18n.Get("cmd.locked.back_safe")
	}
	return i18n.Get("cmd.locked.unlock_all")
}

// listDescWidth leaves room for the command column, the box frame and the
// mark column.
func (m Model) listDescWidth(labelW int) int {
	w := m.w - labelW - 14
	if w < 10 {
		return 10
	}
	if w > 70 {
		return 70
	}
	return w
}

func (m Model) confirmBox() string {
	var title, body string
	switch m.confirm {
	case confirmLocked:
		title = i18n.Get("cmd.warn.title")
		body = i18n.Get("cmd.warn.body")
	case confirmRisk:
		title = i18n.Get("cmd.risk")
		body = i18n.Get("cmd.risk.body")
	}
	return boxStyle(m).Render(lipgloss.JoinVertical(lipgloss.Left,
		stDanger.Render("⚠ "+title),
		"",
		stItem.Render(wrapText(body, m.w-12)),
		"",
		stLogCmd.Render("$ "+truncate(previewArgv(m.pending, m.argVals), m.w-12)),
	))
}
