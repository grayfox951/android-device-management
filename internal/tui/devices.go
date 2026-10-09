package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"adm/internal/android"
	"adm/internal/i18n"
)

// devLoadedInit reports whether the model already knows the current device, so
// the devices screen can be shown without another scan.
func (m *Model) devLoadedInit() bool {
	if m.haveDev {
		return true
	}
	return false
}

// scanDevices queries adb and fastboot, then enriches each entry with codename
// and bootloader state.
func (m *Model) scanDevices() tea.Cmd {
	m.scanning = true
	m.scanErr = nil
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Make sure the daemon is up so a fresh boot does not report nothing.
		_ = android.StartServer(ctx)

		devices, err := android.Scan(ctx)
		for i := range devices {
			android.Enrich(ctx, &devices[i])
		}
		return devicesMsg{devices: devices, err: err}
	}
}

// selectDevice stores the chosen device and loads its category menu.
func (m *Model) selectDevice(d android.Device) {
	m.device = d
	m.haveDev = true
	m.showFull = false
	m.loadCategories()
}

// loadCategories rebuilds the main menu for the current device mode.
func (m *Model) loadCategories() {
	m.cats, _ = android.ByCategory(m.device.Mode)
	m.catIdx = 0
}

func (m *Model) updateDevices(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case devicesMsg:
		m.scanning = false
		m.scanErr = msg.err

		// Keep the current selection across a rescan when the device is
		// still present, so a refresh does not lose the user's place.
		prev := ""
		if m.haveDev {
			prev = m.device.Serial
		}
		m.devices = msg.devices
		if prev != "" {
			for i := range m.devices {
				if m.devices[i].Serial == prev {
					m.devIdx = i
					return m, nil
				}
			}
		}

		// A single ready device is selected straight away.
		if len(m.devices) == 1 {
			m.devIdx = 0
			m.enrichAndSelect(0)
			return m, nil
		}
		if m.devIdx >= len(m.devices) {
			m.devIdx = 0
		}
		return m, nil

	case tcpMsg:
		m.scanning = false
		m.tcpBusy = false
		m.notice = i18nT(msg.key, msg.args...)
		// The device list is stale the moment it joins, so refresh it.
		return m, m.scanDevices()

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

		// The TCP host field owns the keyboard while it is open. This has to be
		// checked before the key switch below, which returns for every key.
		if m.tcpPrompt {
			return m.updateTCPPrompt(msg)
		}

		switch msg.String() {
		case "up", "k":
			if m.devIdx > 0 {
				m.devIdx--
			}
		case "down", "j":
			if m.devIdx < len(m.devices)-1 {
				m.devIdx++
			}
		case "home":
			m.devIdx = 0
		case "end":
			m.devIdx = len(m.devices) - 1
		case "r":
			return m, m.scanDevices()
		case "enter":
			if len(m.devices) == 0 {
				return m, m.scanDevices()
			}
			m.enrichAndSelect(m.devIdx)
		case "t":
			m.openTCPPrompt()
			return m, m.ti.Focus()
		case "c":
			return m, disconnectTCP()
		}
		return m, nil
	}
	return m, nil
}

// openTCPPrompt shows the host field. It does not toggle: once the field is
// open every key belongs to it, so t has to be typeable for hostnames.
func (m *Model) openTCPPrompt() {
	if m.tcpPrompt {
		return
	}
	m.tcpPrompt = true
	m.ti.SetValue("")
	m.ti.Placeholder = i18n.Get("dev.tcp.host")
}

func (m *Model) closeTCPPrompt() {
	m.tcpPrompt = false
	m.ti.Blur()
	m.ti.Placeholder = ""
	m.ti.SetValue("")
}

// updateTCPPrompt feeds the host field and acts on enter and esc.
func (m *Model) updateTCPPrompt(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, isKey := msg.(tea.KeyMsg)
	if isKey {
		switch key.String() {
		case "enter":
			host := normaliseHost(m.ti.Value())
			m.closeTCPPrompt()
			if host == "" {
				return m, nil
			}
			m.tcpBusy = true
			return m, connectTCP(host)
		case "esc":
			m.closeTCPPrompt()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

// normaliseHost trims the field and supplies the default adb port, which is
// what a phone listens on unless wireless debugging picked another one.
func normaliseHost(raw string) string {
	host := strings.TrimSpace(raw)
	if host == "" {
		return ""
	}
	if strings.ContainsAny(host, ": ") || strings.Contains(host, ":") {
		return host
	}
	return host + ":5555"
}

func connectTCP(host string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout()
		defer cancel()

		var lines []string
		ok := false
		for _, r := range android.ConnectTCP(ctx, host) {
			s := r.Trimmed()
			if s == "" {
				continue
			}
			lines = append(lines, s)
			if strings.Contains(s, "connected to") {
				ok = true
			}
		}
		if ok {
			return tcpMsg{key: "dev.tcp.ok", args: []any{host}}
		}
		// Surface the tool's own words rather than inventing a message.
		detail := host
		if len(lines) > 0 {
			detail = strings.Join(lines, " · ")
		}
		return tcpMsg{key: "dev.tcp.fail", args: []any{detail}}
	}
}

// enrichAndSelect re-reads the properties of a device and enters the menu.
func (m *Model) enrichAndSelect(idx int) {
	if idx < 0 || idx >= len(m.devices) {
		return
	}
	d := m.devices[idx]
	ctx, cancel := contextWithTimeout()
	android.Enrich(ctx, &d)
	cancel()
	m.selectDevice(d)
	m.screen = screenMenu
	m.stack = nil
}

// disconnectTCP drops every network device from both tools.
func disconnectTCP() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout()
		defer cancel()

		for _, r := range android.DisconnectTCP(ctx) {
			_ = r
		}
		return tcpMsg{key: "dev.tcp.done", args: nil}
	}
}

// tcpMsg reports the outcome of adb connect / fastboot connect.
type tcpMsg struct {
	key  string
	args []any
}

func (m Model) viewDevices() string {
	var body []string

	switch {
	case m.scanning:
		body = append(body, m.spinnerView(i18n.Get("common.loading")))
	case m.scanErr != nil && len(m.devices) == 0:
		body = append(body, stErr.Render(i18n.T("err.adb", m.scanErr)))
	case len(m.devices) == 0:
		body = append(body, stWarn.Render(i18n.Get("dev.none")))
		body = append(body, "", m.para(i18n.Get("dev.none.hint")))
	default:
		body = append(body, stHeader.Render(i18n.Get("dev.pick")), "")
		body = append(body, m.deviceRows()...)
	}

	if len(m.devices) > 0 {
		switch {
		case len(m.devices) == 1:
			body = append(body, "", stItemMuted.Render(i18n.Get("dev.one")))
		default:
			body = append(body, "", stItemMuted.Render(i18n.T("dev.many", len(m.devices))))
		}
	}

	if m.tcpBusy {
		body = append(body, "", m.spinnerView(i18n.Get("dev.tcp.working")))
	}

	if m.tcpPrompt {
		body = append(body, "", stLabel.Render(i18n.Get("dev.tcp.connect")))
		body = append(body, m.ti.View())
		body = append(body, m.hint(keyCap("enter")+" — "+i18n.Get("common.ok")+
			"  ·  "+keyCap("esc")+" — "+i18n.Get("common.cancel")))
	}

	box := boxStyle(m).Render(lipgloss.JoinVertical(lipgloss.Left, body...))

	foot := lipgloss.JoinVertical(lipgloss.Left,
		m.hint(keyCap("↑↓")+" — "+i18n.Get("common.select")+
			"  ·  "+keyCap("enter")+" — "+i18n.Get("common.ok")+
			"  ·  "+keyCap("r")+" — "+i18n.Get("common.refresh")),
		m.hint(keyCap("t")+" — "+i18n.Get("dev.tcp.connect")+
			"  ·  "+keyCap("c")+" — "+i18n.Get("dev.tcp.disconn")+
			"  ·  "+keyCap("q")+" — "+i18n.Get("common.quit")),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header(i18n.Get("dev.title")),
		"",
		box,
		"",
		foot,
		m.quitConfirm(),
	)
}

// deviceRows renders one block per device: serial, codename, model and the
// bootloader badge coloured by lock state.
func (m Model) deviceRows() []string {
	var rows []string

	// Split whatever room is left between the three identifying columns so
	// the block never runs past the right edge on a narrow terminal.
	avail := m.w - 30
	if avail < 30 {
		avail = 30
	}
	serialW := clamp(avail/3, 12, 26)
	codeW := clamp(avail/4, 10, 20)
	modelW := avail - serialW - codeW
	if modelW < 6 {
		modelW = 6
	}

	for i, d := range m.devices {
		sel := i == m.devIdx

		marker := "  "
		if sel {
			marker = stCursor.Render("▸ ")
		}

		serialStyle := stValue
		if sel {
			serialStyle = stSel
		}

		codename := d.Codename
		if codename == "" {
			codename = i18n.Get("common.none")
		}
		model := d.Model
		if model == "" {
			model = "—"
		}

		title := lipgloss.JoinHorizontal(lipgloss.Top,
			marker,
			serialStyle.Render(padRight(truncate(d.Serial, serialW), serialW)),
			stItemMuted.Render(" · "),
			stItem.Render(padRight(truncate(codename, codeW), codeW)),
			stItemMuted.Render(" · "),
			stItemMuted.Render(truncate(model, modelW)),
		)

		badgeText := badge(d.Boot, i18n.Get("dev.locked"), i18n.Get("dev.unlocked"), i18n.Get("dev.unknown"))

		state := i18n.Get("common.none")
		switch d.State {
		case "device":
			state = i18n.Get("dev.state.device")
		case "unauthorized":
			state = i18n.Get("dev.state.unauth")
		case "offline":
			state = i18n.Get("dev.state.offline")
		case "recovery":
			state = i18n.Get("dev.state.recovery")
		case "sideload":
			state = i18n.Get("dev.state.sideload")
		case "fastboot", "bootloader":
			state = i18n.Get("dev.state.bootloader")
		}

		detail := lipgloss.JoinHorizontal(lipgloss.Top,
			"     ",
			stItemMuted.Render(padRight(string(d.Mode), 10)),
			stItemMuted.Render(padRight(truncate(state, 20), 20)),
			badgeText,
		)

		rows = append(rows, title, detail, "")
	}
	return rows
}

func (m Model) quitConfirm() string {
	if m.confirm != confirmQuit {
		return ""
	}
	return boxStyle(m).Render(lipgloss.JoinVertical(lipgloss.Left,
		boldQuestion(i18n.Get("common.exit")+"?"),
		m.hint(keyCap("y")+" — "+i18n.Get("common.yes")+
			"  ·  "+keyCap("esc")+" — "+i18n.Get("common.no")),
	))
}

// deviceHeader renders the compact device bar shown above every menu.
func (m Model) deviceHeader() string {
	if !m.haveDev {
		return ""
	}
	d := m.device
	codename := d.Codename
	if codename == "" {
		codename = "—"
	}
	badgeText := badge(d.Boot, i18n.Get("dev.locked"), i18n.Get("dev.unlocked"), i18n.Get("dev.unknown"))

	return lipgloss.JoinHorizontal(lipgloss.Top,
		stLabel.Render(i18n.Get("menu.device")+": "),
		stValue.Render(truncate(d.Serial, 22)),
		stItemMuted.Render("  ·  "),
		stItemMuted.Render(truncate(codename, 16)),
		stItemMuted.Render("  ·  "),
		stItemMuted.Render(string(d.Mode)),
		stItemMuted.Render("  ·  "),
		badgeText,
	)
}

// lockedNotice is the explanatory strip shown on a locked device.
func (m Model) lockedNotice() string {
	if !m.lockedLimited() {
		return ""
	}
	return stWarn.Render("⚠ ") + stItemMuted.Render(i18n.Get("dev.sel.locked"))
}
