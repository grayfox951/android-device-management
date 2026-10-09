package tui

import (
	"context"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"adm/internal/i18n"
	"adm/internal/runner"
	"adm/internal/sysinfo"
)

type preflightMsg struct {
	adb      string
	fastboot string
	missing  []string
	info     sysinfo.Info
}

type installDoneMsg struct{ err error }

// start probes for the platform tools.
func (p *preflight) start() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		adbOut, adbErr := probe(ctx, "adb")
		fbOut, fbErr := probe(ctx, "fastboot")

		var missing []string
		if adbErr != nil {
			missing = append(missing, "adb")
		}
		if fbErr != nil {
			missing = append(missing, "fastboot")
		}
		return preflightMsg{adb: adbOut, fastboot: fbOut, missing: missing, info: sysinfo.Detect()}
	}
}

// probe runs "<bin> version" and returns the first line of output.
func probe(ctx context.Context, bin string) (string, error) {
	args := []string{"version"}
	if bin == "fastboot" {
		args = []string{"--version"}
	}
	res := runner.RunQuiet(ctx, bin, args...)
	if res.Err != nil {
		return "", res.Err
	}
	if res.Code != 0 {
		return "", runner.ErrExit{Command: bin, Code: res.Code}
	}
	line := strings.TrimSpace(res.Out)
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		line = line[:idx]
	}
	if line == "" {
		return i18n.Get("common.found"), nil
	}
	return line, nil
}

// install launches the distribution install command with the terminal released
// so sudo can prompt for a password on the real TTY.
func install(plan sysinfo.Plan) tea.Cmd {
	line := plan.Command
	if line == "" {
		line = strings.Join(plan.Packages, " ")
	}
	cmd := exec.Command("sh", "-c", line)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return installDoneMsg{err: err}
	})
}

// runManual executes a command line the user typed themselves.
func runManual(line string) tea.Cmd {
	l := line
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		res := runner.RunShell(ctx, l)
		if res.Err != nil {
			return installDoneMsg{err: res.Err}
		}
		if res.Code != 0 {
			return installDoneMsg{err: runner.ErrExit{Command: l, Code: res.Code}}
		}
		return installDoneMsg{err: nil}
	}
}

// finishSetup records the outcome of a setup run. On a terminal the result
// stays on screen until a key is pressed, so the user gets to read it; without
// a terminal there is nobody to read it and quitting at once avoids a hang.
func (m *Model) finishSetup(out SetupOutcome) (tea.Model, tea.Cmd) {
	m.outcome = out
	if !isTerminalFn() {
		m.quitting = true
		return m, tea.Quit
	}
	m.awaitingSetupExit = true
	return m, nil
}

func (m *Model) updatePreflight(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case preflightMsg:
		m.pre.adb = msg.adb
		m.pre.fastboot = msg.fastboot
		m.pre.missing = msg.missing
		m.pre.info = msg.info
		m.pre.phase = phaseChecking

		if len(msg.missing) == 0 {
			m.pre.phase = phaseReady
			// In setup mode the check is the whole job: report success and
			// quit instead of carrying on to the device list.
			if m.setupOnly {
				return m.finishSetup(SetupOK)
			}
			m.push(screenDevices)
			return m, m.scanDevices()
		}
		m.pre.phase = phaseMissing
		m.pre.plan = msg.info.InstallCommand()
		if !msg.info.Known() {
			// Without a known distribution we can only ask for a command.
			m.pre.manual = ""
			m.pre.manualOK = false
			return m, nil
		}
		return m, nil

	case installDoneMsg:
		if msg.err != nil {
			m.pre.phase = phaseFailed
			if m.setupOnly {
				return m.finishSetup(SetupFailed)
			}
			m.notice = i18n.T("pre.failed")
			return m, nil
		}
		// Re-check whether the tools actually appeared.
		return m, m.pre.start()

	case tea.KeyMsg:
		if m.awaitingSetupExit {
			m.quitting = true
			return m, tea.Quit
		}

		switch m.pre.phase {
		case phaseMissing:
			switch msg.String() {
			case "up", "k", "left", "right", "h", "l":
				m.pre.idx = 1 - m.pre.idx
			case "y":
				m.pre.idx = 0
			case "n":
				m.pre.idx = 1
			case "m":
				if m.pre.info.Known() {
					m.pre.phase = phaseManual
					m.pre.idx = 0
				}
				return m, nil
			case "enter", " ":
				return m.applyPreflightChoice()
			}
		case phaseManual:
			switch msg.String() {
			case "enter":
				cmd := strings.TrimSpace(m.pre.manual)
				if cmd == "" {
					return m, nil
				}
				m.pre.phase = phaseInstalling
				return m, runManual(cmd)
			case "esc":
				m.pre.phase = phaseMissing
			case "ctrl+u":
				m.pre.manual = ""
			case "backspace":
				if n := len(m.pre.manual); n > 0 {
					m.pre.manual = m.pre.manual[:n-1]
				}
			default:
				if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
					m.pre.manual += string(msg.Runes)
				}
			}
		case phaseFailed:
			switch msg.String() {
			case "enter", "r":
				return m, m.pre.start()
			case "q", "esc":
				m.quitting = true
				return m, tea.Quit
			}
		case phaseDeclined:
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *Model) applyPreflightChoice() (tea.Model, tea.Cmd) {
	if !m.pre.info.Known() {
		// No distro: jump straight into manual entry, where cancel quits.
		m.pre.phase = phaseManual
		return m, nil
	}
	if m.pre.idx == 0 {
		m.pre.phase = phaseInstalling
		return m, install(m.pre.plan)
	}
	m.pre.phase = phaseDeclined
	if m.setupOnly {
		return m.finishSetup(SetupDeclined)
	}
	m.quitting = true
	return m, tea.Quit
}

func (m Model) viewPreflight() string {
	p := m.pre

	var rows []string
	rows = append(rows, toolRow("adb", p.adb, len(p.missing) > 0 && contains(p.missing, "adb")))
	rows = append(rows, toolRow("fastboot", p.fastboot, len(p.missing) > 0 && contains(p.missing, "fastboot")))

	headerTitle := i18n.Get("pre.title")
	if m.setupOnly {
		headerTitle = i18n.Get("pre.setup.title")
	}

	body := []string{
		stHeader.Render(headerTitle),
		"",
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	}

	switch p.phase {
	case phaseChecking:
		body = append(body, "", m.hint(i18n.Get("pre.checking")))

	case phaseMissing:
		body = append(body, "")
		body = append(body, stBad.Render(i18n.T("pre.missing", strings.Join(p.missing, ", "))))
		body = append(body, "")
		body = append(body, stLabel.Render(i18n.T("pre.distro", p.info.Display())))
		body = append(body, stLabel.Render(i18n.T("pre.pkg", strings.Join(p.plan.Packages, " "))))
		if len(p.plan.Alternates) > 0 {
			body = append(body, stLabel.Render(i18n.T("pre.alt", strings.Join(p.plan.Alternates, " · "))))
		}
		if cmd := p.plan.Command; cmd != "" {
			body = append(body, "", stLogCmd.Render("$ "+cmd))
		}
		body = append(body, "", stWarn.Render(i18n.Get("pre.sudo")))
		body = append(body, "")
		body = append(body, boldQuestion(i18n.Get("pre.question")))
		body = append(body, m.preChoices())
		body = append(body, "", m.para(i18n.Get("pre.explain")))

	case phaseManual:
		body = append(body, "", stLabel.Render(i18n.Get("pre.nodetect")))
		body = append(body, stValue.Render("$ "+m.pre.manual+stCursor.Render("▏")))
		body = append(body, "", m.hint(keyCap("enter")+" — "+i18n.Get("common.run")+"  ·  "+keyCap("esc")+" — "+i18n.Get("common.back")))

	case phaseInstalling:
		body = append(body, "", m.spinnerView(i18n.Get("pre.installing")))

	case phaseFailed:
		body = append(body, "", stErr.Render(i18n.Get("pre.failed")))
		if m.setupOnly {
			body = append(body, "", stWarn.Render(i18n.Get("pre.setup.keepgoing")))
		}
		body = append(body, "", m.hint(keyCap("enter")+" — "+i18n.Get("pre.recheck")+"  ·  "+keyCap("q")+" — "+i18n.Get("common.quit")))

	case phaseReady:
		if m.setupOnly {
			body = append(body, "", stGood.Render(i18n.Get("pre.setup.done")))
		} else {
			body = append(body, "", stGood.Render(i18n.Get("pre.done")))
		}
	}

	box := boxStyle(m).Render(lipgloss.JoinVertical(lipgloss.Left, body...))

	title := i18n.Get("pre.title")
	if m.setupOnly {
		title = i18n.Get("pre.setup.title")
	}

	var foot []string
	if m.setupOnly {
		foot = append(foot, m.hint(i18n.Get("pre.setup.foot")))
		if m.awaitingSetupExit {
			foot = append(foot, m.hint(keyCap("enter")+" — "+i18n.Get("pre.setup.ack")))
		}
	} else {
		foot = append(foot,
			m.hint(i18n.Get("pre.start.tip")),
			m.hint(i18n.Get("pre.udev.tip")),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header(title),
		"",
		box,
		"",
		lipgloss.JoinVertical(lipgloss.Left, foot...),
	)
}

// preChoices renders the yes/no prompt shown when tools are missing.
func (m Model) preChoices() string {
	yes := selector(m.pre.idx == 0, 24, i18n.Get("common.yes"))
	no := selector(m.pre.idx == 1, 24, i18n.Get("common.no"))
	row := yes + "   " + no
	hint := m.hint(keyCap("enter") + " — " + i18n.Get("common.confirm") +
		"  ·  " + keyCap("m") + " — " + i18n.Get("pre.manual"))
	return lipgloss.JoinVertical(lipgloss.Left, row, hint)
}

func (m Model) spinnerView(text string) string {
	return m.sp.View() + " " + text
}

func boldQuestion(s string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(colAccent2).Render(s)
}

func toolRow(name, detail string, missing bool) string {
	label := lipgloss.NewStyle().Width(10).Render(name)
	if missing {
		return label + stBad.Render("✗") + "  " + stItemMuted.Render(i18n.Get("common.missing"))
	}
	if detail == "" {
		detail = i18n.Get("common.found")
	}
	return label + stOK.Render("✓") + "  " + truncate(detail, 60)
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
