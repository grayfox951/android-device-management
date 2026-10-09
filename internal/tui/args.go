package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"adm/internal/android"
	"adm/internal/i18n"
)

func (m *Model) updateArgs(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.argIdx >= len(m.pending.Args) {
		return m.launch(m.pending)
	}
	arg := m.pending.Args[m.argIdx]

	switch m.pickingCh {
	case true:
		key, ok := msg.(tea.KeyMsg)
		if !ok {
			return m, nil
		}
		switch key.String() {
		case "up", "k", "left":
			if m.chIdx > 0 {
				m.chIdx--
			}
		case "down", "j", "right":
			if m.chIdx < len(arg.Choices)-1 {
				m.chIdx++
			}
		case "home":
			m.chIdx = 0
		case "end":
			m.chIdx = len(arg.Choices) - 1
		case "esc":
			return m, nil
		case "enter":
			m.commitArg(arg.Choices[m.chIdx].Value)
			return m.advanceArg()
		}
		return m, nil
	}

	// Text entry mode: the input owns the keyboard except for the browser.
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.ti, cmd = m.ti.Update(msg)
		return m, cmd
	}

	switch key.String() {
	case "esc":
		m.ti.Blur()
		return m, nil
	case "enter":
		m.commitArg(m.ti.Value())
		m.ti.Blur()
		return m.advanceArg()
	case "tab":
		if arg.Kind == android.ArgFile || arg.Kind == android.ArgDir {
			m.openFileBrowser(arg)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

func (m *Model) commitArg(v string) {
	m.argVals[m.pending.Args[m.argIdx].Key] = v
}

// advanceArg moves to the next argument or launches the command.
func (m *Model) advanceArg() (tea.Model, tea.Cmd) {
	m.argIdx++
	if m.argIdx >= len(m.pending.Args) {
		return m.launch(m.pending)
	}

	arg := m.pending.Args[m.argIdx]
	prev, _ := m.argVals[arg.Key]
	if prev == "" {
		prev = arg.Default
	}
	m.ti.Placeholder = arg.Label
	m.ti.SetValue(prev)

	if arg.Kind == android.ArgChoice {
		m.pickingCh = true
		m.chIdx = choiceIndex(arg, prev)
		return m, nil
	}
	m.pickingCh = false
	return m, m.ti.Focus()
}

// openFileBrowser starts the file browser for a file or folder argument.
func (m *Model) openFileBrowser(arg android.Arg) {
	m.fb = newFileBrowser(arg.Kind, m.argVals[arg.Key])
	m.fb.viewW = m.w
	m.push(screenFile)
}

func (m Model) viewArgs() string {
	c := m.pending
	if len(c.Args) == 0 {
		// Nothing to collect: show the command itself instead of crashing.
		return lipgloss.JoinVertical(lipgloss.Left,
			header(i18n.Get("cmd.args.title")),
			"",
			boxStyle(m).Render(lipgloss.JoinVertical(lipgloss.Left,
				stValue.Render(c.Label),
				"",
				m.para(i18n.Get("cmd.need.args")),
			)),
		)
	}

	body := []string{
		stHeader.Render(i18n.Get("cmd.args.title")),
		"",
		stValue.Render(c.Label),
	}

	if desc := i18n.Get(c.DescKey); desc != c.DescKey {
		body = append(body, stItemMuted.Render(desc))
	}

	body = append(body, "", rule(40))

	// Progress: which argument we are on.
	for i, a := range c.Args {
		var mark string
		switch {
		case i < m.argIdx:
			mark = stOK.Render("✓")
		case i == m.argIdx:
			mark = stCursor.Render("▸")
		default:
			mark = stItemMuted.Render("·")
		}
		val := m.argVals[a.Key]
		line := mark + " " + stLabel.Render(a.Label)
		if val != "" {
			line += "  " + stValue.Render(truncate(val, 40))
		}
		body = append(body, line)
	}

	if m.argIdx < len(c.Args) {
		body = append(body, "", m.currentArgWidget(c.Args[m.argIdx]))
	}

	if blocked := m.blockedBox(); blocked != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			header(i18n.Get("cmd.args.title")),
			"",
			blocked,
		)
	}

	if m.confirm != confirmNone {
		// A pending question replaces the form so the boxes do not nest.
		return lipgloss.JoinVertical(lipgloss.Left,
			header(i18n.Get("cmd.args.title")),
			"",
			m.confirmBox(),
		)
	}

	box := boxStyle(m).Render(lipgloss.JoinVertical(lipgloss.Left, body...))

	return lipgloss.JoinVertical(lipgloss.Left,
		header(i18n.Get("cmd.args.title")),
		"",
		box,
		"",
		m.argsFoot(),
	)
}

func (m Model) argsFoot() string {
	foot := keyCap("enter") + " — " + i18n.Get("common.ok") +
		"  ·  " + keyCap("esc") + " — " + i18n.Get("common.cancel")
	if len(m.pending.Args) > 0 && m.argIdx < len(m.pending.Args) {
		if a := m.pending.Args[m.argIdx]; a.Kind == android.ArgFile || a.Kind == android.ArgDir {
			foot += "  ·  " + keyCap("tab") + " — " + i18n.Get("file.pick.file")
		}
	}
	return m.hint(foot)
}

// currentArgWidget renders the input for the argument being filled in.
func (m Model) currentArgWidget(arg android.Arg) string {
	switch arg.Kind {
	case android.ArgChoice:
		return m.choiceWidget(arg)
	case android.ArgPackage:
		return m.ti.View() + "\n" + m.para(i18n.Get("cmd.need.pkg"))
	case android.ArgPartition:
		return m.ti.View() + "\n" + m.para(i18n.Get("cmd.need.part"))
	case android.ArgFile:
		return m.ti.View() + "\n" + m.hint(keyCap("tab")+" — "+i18n.Get("cmd.need.file"))
	case android.ArgDir:
		return m.ti.View() + "\n" + m.hint(keyCap("tab")+" — "+i18n.Get("cmd.need.dir"))
	case android.ArgRemotePath:
		return m.ti.View() + "\n" + m.para(i18n.Get("cmd.need.remote"))
	default:
		return m.ti.View()
	}
}

func (m Model) choiceWidget(arg android.Arg) string {
	maxW := 0
	for _, c := range arg.Choices {
		if n := lipgloss.Width(c.Label); n > maxW {
			maxW = n
		}
	}
	var rows []string
	for i, c := range arg.Choices {
		label := c.Label
		if label == "" {
			label = stItemMuted.Render("(" + i18n.Get("common.none") + ")")
			label = padRight(label, maxW)
		}
		rows = append(rows, "  "+selector(i == m.chIdx, maxW, label))
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		stHeader.Render(i18n.Get("cmd.need.choose")),
		"",
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

// previewArgv renders the command line that will be executed, used in the
// confirmation boxes before anything irreversible happens.
func previewArgv(c android.Command, vals map[string]string) string {
	if c.Build == nil {
		return c.Label
	}
	full := map[string]string{}
	for _, a := range c.Args {
		full[a.Key] = a.Default
	}
	for k, v := range vals {
		full[k] = v
	}
	argv := c.Build(android.Device{Serial: "…"}, full)
	return strings.Join(argv, " ")
}
