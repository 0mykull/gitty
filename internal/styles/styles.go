package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette - pink, blue, purple, white, red theme
var (
	// Primary colors
	Pink   = lipgloss.Color("#FF6B9D") // Hot pink
	Purple = lipgloss.Color("#A855F7") // Purple
	Blue   = lipgloss.Color("#60A5FA") // Blue
	Cyan   = lipgloss.Color("#22D3EE") // Cyan
	White  = lipgloss.Color("#FFFFFF") // White
	Red    = lipgloss.Color("#F87171") // Red
	Green  = lipgloss.Color("#4ADE80") // Green
	Yellow = lipgloss.Color("#FBBF24") // Yellow

	// Main theme colors
	Primary   = Pink
	Secondary = Purple
	Accent    = Blue
	Success   = Green
	Warning   = Yellow
	Error     = Red
	Info      = Cyan

	// Text colors
	TextPrimary   = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#FFFFFF"}
	TextSecondary = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#D1D5DB"}
	TextMuted     = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#9CA3AF"}
	Border        = lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#6B7280"}
	BorderAccent  = Purple
)

// Icons for beautiful display
var Icons = struct {
	Git       string
	Branch    string
	Commit    string
	Push      string
	Pull      string
	Add       string
	Reset     string
	Publish   string
	Open      string
	AI        string
	Config    string
	Check     string
	Cross     string
	Arrow     string
	Dot       string
	Star      string
	Lightning string
	Folder    string
	File      string
	Warning   string
	Info      string
	Lazygit   string
	Quit      string
}{
	Git:       "",
	Branch:    "",
	Commit:    "",
	Push:      "",
	Pull:      "",
	Add:       "",
	Reset:     "",
	Publish:   "",
	Open:      "",
	AI:        "",
	Config:    "",
	Check:     "",
	Cross:     "",
	Arrow:     "",
	Dot:       "",
	Star:      "",
	Lightning: "",
	Folder:    "",
	File:      "",
	Warning:   "",
	Info:      "",
	Lazygit:   "",
	Quit:      "",
}

// Base styles
var (
	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Pink).
			MarginBottom(1)

	// Box styles with borders
	TitleBoxStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Pink).
			Padding(0, 0) // Removed border padding

	BranchBoxStyle = lipgloss.NewStyle().
			Foreground(Cyan).
			Bold(true).
			Padding(0, 0) // Removed border padding

	BoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(1, 2)

	AccentBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Purple).
			Padding(1, 2)

	// List item styles
	ListItemStyle = lipgloss.NewStyle().
			Foreground(TextPrimary).
			PaddingLeft(2)

	ListItemSelectedStyle = lipgloss.NewStyle().
				Foreground(Pink).
				Bold(true).
				PaddingLeft(0)

	ListItemDescStyle = lipgloss.NewStyle().
				Foreground(TextMuted).
				PaddingLeft(4)

	// Status styles
	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Info)

	// Spinner style
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(Pink)

	// Help style
	HelpStyle = lipgloss.NewStyle().
			Foreground(TextMuted).
			MarginTop(1)

	// Header style
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Purple).
			MarginBottom(1)

	// Divider
	DividerStyle = lipgloss.NewStyle().
			Foreground(Border)
)

// Render helpers
func RenderSuccess(msg string) string {
	return SuccessStyle.Render(Icons.Check + " " + msg)
}

func RenderError(msg string) string {
	return ErrorStyle.Render(Icons.Cross + " " + msg)
}

func RenderWarning(msg string) string {
	return WarningStyle.Render(Icons.Warning + " " + msg)
}

func RenderInfo(msg string) string {
	return InfoStyle.Render(Icons.Info + " " + msg)
}

// Divider returns a styled horizontal divider
func Divider(width int) string {
	line := ""
	for i := 0; i < width; i++ {
		line += "─"
	}
	return DividerStyle.Render(line)
}

// StatusBadge renders a colored status badge
func StatusBadge(status string) string {
	var style lipgloss.Style
	switch status {
	case "success", "clean":
		style = lipgloss.NewStyle().
			Background(Success).
			Foreground(lipgloss.Color("#000000")).
			Padding(0, 1).
			Bold(true)
	case "error", "dirty":
		style = lipgloss.NewStyle().
			Background(Error).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1).
			Bold(true)
	case "warning":
		style = lipgloss.NewStyle().
			Background(Warning).
			Foreground(lipgloss.Color("#000000")).
			Padding(0, 1).
			Bold(true)
	default:
		style = lipgloss.NewStyle().
			Background(Info).
			Foreground(lipgloss.Color("#000000")).
			Padding(0, 1).
			Bold(true)
	}
	return style.Render(status)
}
