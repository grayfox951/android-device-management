package tui

import "github.com/charmbracelet/lipgloss"

// The palette is kept apart from the styles: the model needs the accent colour
// before any style exists, and a colour on its own is easier to change.
// Palette follows the Catppuccin Mocha scheme: a cool dark background with
// pastel accents that stay legible on both light and dark terminals.
var (
	colAccent    = lipgloss.AdaptiveColor{Light: "#1e66f5", Dark: "#89b4fa"}
	colAccent2   = lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#cba6f7"}
	colSuccess   = lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"}
	colWarning   = lipgloss.AdaptiveColor{Light: "#df8e1d", Dark: "#f9e2af"}
	colDanger    = lipgloss.AdaptiveColor{Light: "#d20f39", Dark: "#f38ba8"}
	colMuted     = lipgloss.AdaptiveColor{Light: "#6c7086", Dark: "#7f849c"}
	colText      = lipgloss.AdaptiveColor{Light: "#4e4e69", Dark: "#cdd6f4"}
	colSurface   = lipgloss.AdaptiveColor{Light: "#e6e6ef", Dark: "#1e1e2e"}
	colOverlay   = lipgloss.AdaptiveColor{Light: "#dcdce4", Dark: "#313244"}
	colHighlight = lipgloss.AdaptiveColor{Light: "#ccdafd", Dark: "#45475a"}
)
