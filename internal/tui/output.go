package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"adm/internal/i18n"
)

func (m *Model) updateOutput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.run.poll()
		m.run.refreshViewport()
		return m, m.run.tick()

	case saveMsg:
		if msg.err != nil {
			m.notice = i18n.T("cmd.save.fail", msg.err)
		} else {
			m.notice = i18n.T("cmd.saved", msg.path)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter":
			if m.run.running {
				return m, nil
			}
			m.pop()
			return m, nil

		case "y":
			if m.run.running {
				return m, nil
			}
			return m, m.saveLog()

		case "g":
			m.run.vp.GotoTop()
			return m, nil
		case "G":
			m.run.vp.GotoBottom()
			return m, nil
		}

		var cmd tea.Cmd
		m.run.vp, cmd = m.run.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) viewOutput() string {
	r := m.run

	var status string
	switch {
	case r.running:
		status = stWarn.Render(m.spinnerView(i18n.Get("common.working")))
	case r.code == 0:
		status = stGood.Render("✓ " + i18n.Get("common.done") +
			"  " + stItemMuted.Render(formatDuration(r.took)))
	default:
		status = stErr.Render("✗ " + i18n.T("common.exitcode", r.code) +
			"  " + stItemMuted.Render(formatDuration(r.took)))
	}

	body := []string{
		stLabel.Render("$ ") + stLogCmd.Render(r.cmdLine),
		"",
		status,
	}

	if r.redirect != "" && r.done {
		body = append(body, stItemMuted.Render("→ "+i18n.Get("cmd.need.file")+": ")+stValue.Render(r.redirect))
	}

	if r.total > maxLines {
		body = append(body, stItemMuted.Render(
			i18n.T("file.size", itoa(r.total-maxLines))+" … "+
				i18n.T("file.size", itoa(r.total))))
	}

	box := boxStyle(m).Render(lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinVertical(lipgloss.Left, body...),
		"",
		r.vp.View(),
	))

	foot := m.hint(keyCap("↑↓") + " — " + i18n.Get("cmd.out") +
		"  ·  " + keyCap("g") + "/" + keyCap("G") + " — top/bottom")
	if r.running {
		foot += "  ·  " + stItemMuted.Render(i18n.Get("common.working"))
	} else {
		foot += "  ·  " + keyCap("y") + " — " + i18n.Get("cmd.save") +
			"  ·  " + keyCap("esc") + " — " + i18n.Get("common.back")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header(i18n.Get("cmd.out")),
		"",
		box,
		"",
		m.hint(foot),
	)
}
