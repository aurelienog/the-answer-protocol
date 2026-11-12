package layout

import "github.com/charmbracelet/lipgloss"

var medievalBorder = lipgloss.Border{
    Top:         "═",
    Bottom:      "═",
    Left:        "║",
    Right:       "║",
    TopLeft:     "╔",
    TopRight:    "╗",
    BottomLeft:  "╚",
    BottomRight: "╝",
    MiddleLeft:  "╠",
    MiddleRight: "╣",
}

var (
    Primary   = lipgloss.Color("#C9A44C") // Ancient Gold
    Secondary = lipgloss.Color("#477FA8") // Arcane Blue
    Success   = lipgloss.Color("#39845A") // Emerald
    Danger    = lipgloss.Color("#A83232") // Blood Red
    Warning   = lipgloss.Color("#D69B36") // Amber
    Info      = lipgloss.Color("#75A9C9") // Magic Blue
    Muted     = lipgloss.Color("#77736A") // Stone
    Accent    = lipgloss.Color("#9166A8") // Mystic Purple
)

var (
    PanelStyle = lipgloss.NewStyle().
        Border(medievalBorder).
        Padding(1, 2)

    StatusStyle = lipgloss.NewStyle().
        Foreground(Muted)

    InputStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(Success)

    TitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(Primary)
)